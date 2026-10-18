package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/herrnan/laga/internal/app"
	"github.com/kelseyhightower/envconfig"
)

const appName = "LAGA"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))
	slog.SetDefault(logger)

	ctx, cleanup := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func(ctx context.Context, cleanup context.CancelFunc) {
		<-ctx.Done()
		cleanup()
	}(ctx, cleanup)

	var cfg app.Config
	err := envconfig.Process(appName, &cfg)
	if err != nil {
		slog.Error("Failed to process env vars", "error", err)
		os.Exit(1)
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	err = app.Run(ctx, logger, cfg)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to run", "error", err)
		os.Exit(1)
	}
	logger.Debug("shutdown successful")
}
