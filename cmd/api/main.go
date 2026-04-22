package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/netologist/ai-support-platform/internal/app/provider"
)

func main() {
	config, err := provider.LoadConfig()
	if err != nil {
		slog.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	// single lifecycle context
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runtime, err := provider.NewRuntime(ctx, config)
	if err != nil {
		slog.Error("build runtime", slog.Any("error", err))
		os.Exit(1)
	}
	defer runtime.CloseFn()

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = runtime.Server.Shutdown(shutdownCtx)
	}()

	slog.Info("starting api server", slog.String("addr", runtime.Server.Addr))

	err = runtime.Server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		slog.Error("api server stopped", slog.Any("error", err))
		os.Exit(1)
	}
}
