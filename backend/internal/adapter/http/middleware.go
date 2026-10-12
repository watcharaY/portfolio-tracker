package http

import (
	"log/slog"
	"slices"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// trustProxy maps TRUSTED_PROXIES onto Fiber's allowlist. Only a request whose
// peer is listed may set the client IP through X-Forwarded-For; the keywords
// "private" and "loopback" stand for those whole ranges.
func trustProxy(proxies []string) (bool, fiber.TrustProxyConfig) {
	tp := fiber.TrustProxyConfig{
		Private:  slices.Contains(proxies, "private"),
		Loopback: slices.Contains(proxies, "loopback"),
	}
	for _, p := range proxies {
		if p != "private" && p != "loopback" {
			tp.Proxies = append(tp.Proxies, p)
		}
	}
	return len(proxies) > 0, tp
}

// accessLog writes one structured line per request. It never logs bodies,
// cookies, query strings or headers other than the user agent, because the
// data here is personal finance.
func accessLog(log *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		if err := c.Next(); err != nil {
			// Run the error handler now so the logged status is the one sent.
			if herr := c.App().ErrorHandler(c, err); herr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}
		status := c.Response().StatusCode()
		attrs := []slog.Attr{
			slog.String("request_id", requestid.FromContext(c)),
			slog.String("method", c.Method()),
			slog.String("route", c.Route().Path),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Float64("latency_ms", float64(time.Since(start).Microseconds())/1000),
			slog.Int("bytes", len(c.Response().Body())),
			slog.String("ip", c.IP()),
			slog.String("user_agent", c.Get(fiber.HeaderUserAgent)),
		}
		if u := currentUser(c); u.ID != 0 {
			attrs = append(attrs, slog.Int64("user_id", u.ID))
		}
		log.LogAttrs(c.Context(), accessLevel(status), "request", attrs...)
		return nil
	}
}

func accessLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400 && status != fiber.StatusUnauthorized && status != fiber.StatusNotFound:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

// noStore keeps browsers and proxies from caching API responses.
func noStore(c fiber.Ctx) error {
	c.Set(fiber.HeaderCacheControl, "no-store")
	return c.Next()
}

// rateLimit allows max requests per minute per client IP. Over the limit it
// answers 429 with Retry-After and X-RateLimit-* headers and the API's JSON
// error body. skip exempts requests such as health checks.
func rateLimit(max int, skip func(fiber.Ctx) bool) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: time.Minute,
		Next:       skip,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(errorBody{Error: "too many requests, retry later"})
		},
	})
}

func isHealthCheck(c fiber.Ctx) bool {
	return c.Path() == "/api/health" || c.Path() == "/api/ready"
}
