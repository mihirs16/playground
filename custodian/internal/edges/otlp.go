package edges

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

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

// newOTLPTelemetry stands the three providers up against the endpoint, sending
// the configured Authorization header verbatim (see config.OTLPAuthorization).
//
// Misconfiguration must never masquerade as "telemetry off". An empty endpoint
// is telemetry deliberately disabled: a no-op sink and no error. A non-empty
// endpoint whose exporter cannot be built returns a no-op sink so the box still
// boots, plus a non-nil error the caller logs.
func newOTLPTelemetry(endpoint, authorization string, logger *slog.Logger) (Telemetry, error) {
	if endpoint == "" {
		return noopTelemetry{}, nil
	}

	telemetry, err := buildOTLPTelemetry(context.Background(), endpoint, authorization, logger)
	if err != nil {
		return noopTelemetry{}, err
	}
	return telemetry, nil
}

func buildOTLPTelemetry(ctx context.Context, endpoint, authorization string, logger *slog.Logger) (*otlpTelemetry, error) {
	// Route the SDK's async export failures through custodian's logger (a
	// process-global install). This is what makes a runtime auth rejection — the
	// Grafana Cloud 401 a bad credential produces on the periodic export loop —
	// loud rather than buried in the SDK's own internal log.
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Error("OTLP export failed", "error", err)
	}))

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName("custodian"),
		semconv.ServiceNamespace("playground"),
	))
	if err != nil {
		return nil, err
	}

	var headers map[string]string
	if authorization != "" {
		headers = map[string]string{"Authorization": authorization}
	}

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(signalURL(endpoint, "metrics")),
		otlpmetrichttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(signalURL(endpoint, "traces")),
		otlptracehttp.WithHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(signalURL(endpoint, "logs")),
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

// signalURL builds the per-signal OTLP/HTTP URL from the base endpoint, matching
// the standard OTEL_EXPORTER_OTLP_ENDPOINT convention: the endpoint is a base
// onto which "/v1/<signal>" is appended. custodian holds one endpoint for all
// three signals, so it must append the suffix itself — the SDK's WithEndpointURL
// takes the path verbatim and does not. For Grafana Cloud's ".../otlp" gateway
// this yields ".../otlp/v1/metrics", the path its gateway serves.
func signalURL(endpoint, signal string) string {
	return strings.TrimRight(endpoint, "/") + "/v1/" + signal
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
