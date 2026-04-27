package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/netologist/ai-support-platform/internal/app/provider"
)

func main() {
	config, err := provider.LoadConfig()
	if err != nil {
		slog.Error("load config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	runtime, err := provider.NewWorkerRuntime(ctx, config)
	if err != nil {
		slog.Error("build worker runtime", slog.Any("error", err))
		os.Exit(1)
	}
	defer runtime.Close()

	slog.Info("starting worker")
	if err := runtime.Run(ctx); err != nil && ctx.Err() == nil {
		slog.Error("consumer stopped", slog.Any("error", err))
		os.Exit(1)
	}
}
