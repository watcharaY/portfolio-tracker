// Command server runs the Portfolio Tracker API, plus the built web UI when WEB_DIR is set.
//
//	server              migrate, seed the default user and serve
//	server migrate      apply pending migrations and exit
//	server healthcheck  exit 0 when the local server answers /api/ready with 200
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

	run := app.Run
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			run = app.Migrate
		case "healthcheck":
			run = app.Healthcheck
		default:
			slog.Error("unknown command", "cmd", os.Args[1], "usage", "server [migrate|healthcheck]")
			os.Exit(2)
		}
	}
	if err := run(ctx, config.Load()); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
