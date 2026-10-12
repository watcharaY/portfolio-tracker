// Command server runs the Portfolio Tracker API and serves the embedded web UI.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/watcharaY/portfolio-tracker/backend/internal/app"
	"github.com/watcharaY/portfolio-tracker/backend/internal/platform/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, config.Load()); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
