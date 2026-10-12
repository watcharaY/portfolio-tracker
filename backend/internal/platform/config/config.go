package config

import (
	"os"
	"time"
)

type Config struct {
	Addr         string
	DatabaseURL  string
	CookieSecure bool
	SessionTTL   time.Duration
	CORSOrigin   string
	WebDir       string
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
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
