package app

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/watcharaY/portfolio-tracker/backend/internal/platform/config"
)

// Healthcheck asks the running server on this host whether it is ready. It lets
// the distroless image, which has no curl, define a Docker HEALTHCHECK.
func Healthcheck(ctx context.Context, cfg config.Config) error {
	_, port, err := net.SplitHostPort(cfg.Addr)
	if err != nil {
		return fmt.Errorf("ADDR %q: %w", cfg.Addr, err)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+net.JoinHostPort("127.0.0.1", port)+"/api/ready", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("/api/ready answered %d", resp.StatusCode)
	}
	return nil
}
