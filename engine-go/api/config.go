// Package api is the HTTP layer of the Go port of the NestJS backend: chi router,
// middleware (CORS + cookie/bearer JWT auth), and per-domain handlers. Deps live
// here and in cmd/api — never in engine/, so the WASM build stays dep-free.
package api

import (
	"os"
	"strings"
)

// Config mirrors backend/.env. The Go API runs on its own PORT + DB file next to
// the Nest server during the migration; the Vite proxy flips to it at cutover.
type Config struct {
	Port         string
	DatabasePath string
	JWTSecret    string
	JWTExpiresIn string
	CookieName   string
	CookieSecure bool
	CORSOrigin   string
}

// LoadConfig reads the environment with the same defaults as the Nest backend,
// except PORT/DatabasePath default to non-conflicting values so both servers can
// run side by side until the big-bang cutover.
func LoadConfig() Config {
	return Config{
		Port:         env("PORT", "3001"),
		DatabasePath: stripFilePrefix(env("DATABASE_URL", "file:./t20-go.db")),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		JWTExpiresIn: env("JWT_EXPIRES_IN", "7d"),
		CookieName:   env("COOKIE_NAME", "t20_session"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
		CORSOrigin:   env("CORS_ORIGIN", "http://localhost:5173"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// stripFilePrefix turns a Prisma-style "file:./dev.db" URL into a plain path.
func stripFilePrefix(url string) string {
	return strings.TrimPrefix(url, "file:")
}
