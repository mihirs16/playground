package edges

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// otlpTelemetry is the production OTLP sink: the OTel Go SDK exporting metrics,
// traces, and logs over OTLP/HTTP directly to a configured endpoint — no agent
// sidecar. The health gauge is a synchronous Int64Gauge set 1 (healthy) or 0
// (degraded) each time custodian self-assesses on the poll loop's timer.
type otlpTelemetry struct {
	gauge     metric.Int64Gauge
	providers []func(context.Context) error
}

// newOTLPTelemetry stands the three providers up against the endpoint, wiring
// the export token as a bearer header. endpoint and token are read from the
// environment at startup; their source is opaque to custodian. An empty
// endpoint, or an exporter that cannot be built, yields a no-op sink so a
// missing secret surfaces as absent telemetry rather than a boot failure.
func newOTLPTelemetry(endpoint, token string) Telemetry {
	if endpoint == "" {
		return noopTelemetry{}
	}

	telemetry, err := buildOTLPTelemetry(context.Background(), endpoint, token)
	if err != nil {
		return noopTelemetry{}
	}
	return telemetry
}

func buildOTLPTelemetry(ctx context.Context, endpoint, token string) (*otlpTelemetry, error) {
	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName("custodian")))
	if err != nil {
		return nil, err
	}

	var headers map[string]string
	if token != "" {
		headers = map[string]string{"Authorization": "Bearer " + token}
	}

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(endpoint),
		otlpmetrichttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(endpoint),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(endpoint),
		otlploghttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
	)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
	)
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
	)

	otel.SetMeterProvider(meterProvider)
	otel.SetTracerProvider(tracerProvider)
	otellog.SetLoggerProvider(loggerProvider)

	gauge, err := meterProvider.Meter("custodian").Int64Gauge(
		"health",
		metric.WithDescription("1 when custodian self-assesses healthy, 0 when degraded"),
	)
	if err != nil {
		return nil, err
	}

	return &otlpTelemetry{
		gauge: gauge,
		providers: []func(context.Context) error{
			meterProvider.Shutdown,
			tracerProvider.Shutdown,
			loggerProvider.Shutdown,
		},
	}, nil
}

func (t *otlpTelemetry) RecordHealth(ctx context.Context, healthy bool) {
	var value int64
	if healthy {
		value = 1
	}
	t.gauge.Record(ctx, value)
}

// Shutdown flushes and stops every provider, joining their errors so a single
// failing exporter does not hide the others.
func (t *otlpTelemetry) Shutdown(ctx context.Context) error {
	var errs error
	for _, shutdown := range t.providers {
		if err := shutdown(ctx); err != nil {
			errs = joinErr(errs, err)
		}
	}
	return errs
}

func joinErr(existing, next error) error {
	if existing == nil {
		return next
	}
	return fmt.Errorf("%w; %w", existing, next)
}

// noopTelemetry is the sink used when no OTLP endpoint is configured: it accepts
// the gauge and shuts down cleanly, so the rest of the box runs unchanged.
type noopTelemetry struct{}

func (noopTelemetry) RecordHealth(context.Context, bool) {}
func (noopTelemetry) Shutdown(context.Context) error     { return nil }
