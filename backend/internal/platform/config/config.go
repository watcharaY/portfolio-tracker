package config

import (
	"cmp"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	CookieSecure bool
	SessionTTL   time.Duration
	CORSOrigin   string
	WebDir       string
	// TrustedProxies is TRUSTED_PROXIES split on commas.
	TrustedProxies     []string
	RateLimitPerMinute int
	// LogFormat is "json" (default) or "text".
	LogFormat string
}

func Load() Config {
	ttl, err := time.ParseDuration(env("SESSION_TTL", "168h"))
	if err != nil {
		ttl = 7 * 24 * time.Hour
	}
	return Config{
		Addr:         env("ADDR", ":8080"),
		DatabaseURL:  env("DATABASE_URL", "postgres://portfolio:portfolio@localhost:5432/portfolio?sslmode=disable"),
		CookieSecure: env("COOKIE_SECURE", "false") == "true",
		SessionTTL:   ttl,
		CORSOrigin:   os.Getenv("CORS_ORIGIN"),
		WebDir:       os.Getenv("WEB_DIR"),

		TrustedProxies:     splitList(os.Getenv("TRUSTED_PROXIES")),
		RateLimitPerMinute: envInt("RATE_LIMIT_PER_MINUTE", 300),
		LogFormat:          env("LOG_FORMAT", "json"),
	}
}

func splitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envInt(key string, def int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n > 0 {
		return n
	}
	return def
}

func env(key, def string) string {
	return cmp.Or(os.Getenv(key), def)
}
