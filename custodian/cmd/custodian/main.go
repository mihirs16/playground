// Command custodian is the always-on API that keeps the playground. It boots
// one HTTP server carrying both the public and admin surfaces over an
// in-process SQLite database, with the three outward edges (S3, source clients,
// OTLP) wired to their real implementations.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mihirs16/playground/custodian/internal/config"
	"github.com/mihirs16/playground/custodian/internal/edges"
	"github.com/mihirs16/playground/custodian/internal/server"
	"github.com/mihirs16/playground/custodian/internal/storage"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("custodian exited", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg := config.Load()

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	edgeSet := edges.Real(cfg, logger)
	srv := server.New(cfg, db, edgeSet, logger)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go srv.Poller().Run(ctx, logger)

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("custodian listening", "addr", cfg.Addr)
		serveErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return edgeSet.Telemetry.Shutdown(shutdownCtx)
	}
}
