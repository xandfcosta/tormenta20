// Package api is the app's HTTP layer: chi router, middleware (CORS +
// cookie/bearer JWT auth), and per-domain handlers. Deps live here and in
// cmd/api — never in engine/, so the WASM build stays dep-free.

package api

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// AppEnv names the environment. It picks which `.env.<AppEnv>` file LoadConfig
// reads AND how strict Validate is — the two are one decision, so they share
// one variable (APP_ENV) instead of drifting apart.
type AppEnv string

const (
	EnvDevelopment AppEnv = "development"
	EnvProduction  AppEnv = "production"
)

// DevJWTSecret is the throwaway signing key `.env.development` ships with. It is
// committed on purpose — a dev token is worthless — which is exactly why
// Validate refuses it in production: a copied file must not become the key that
// signs sessions for everyone on the LAN (ALE-119).
const DevJWTSecret = "t20-dev-secret"

// Config is the server's environment, read once at startup.
type Config struct {
	AppEnv AppEnv
	// AdminEmails is the closed list of accounts that administer the table. The
	// role lives HERE and not in a database column on purpose (ALE-120): there is
	// no promote endpoint, so the only way to become admin is editing this file
	// on the host — no HTTP bug can turn a player into one. The price is that
	// changing it takes an edit plus a restart.
	AdminEmails  []string
	Port         string
	DatabasePath string
	JWTSecret    string
	JWTExpiresIn string
	CookieName   string
	CookieSecure bool
	// CORSOrigin is the ONE browser origin allowed to call the API cross-origin —
	// the Vite dev server. Empty means no CORS middleware at all, which is the
	// production shape: the binary serves the SPA itself, so every call is
	// same-origin and no other site has any business reaching it.
	CORSOrigin string
	// CatalogPath is the primeEngineCatalogs payload (items/races/…) the API loads
	// at startup for its mutation validators. Defaults to the committed snapshot.
	CatalogPath string
	// BackupDir is where the admin screen writes snapshots — the same directory
	// the `pnpm db:backup` script uses, so a backup made either way shows up in
	// both places (ALE-120). Relative to engine-go/, which is the server's CWD.
	BackupDir string
	// StaticDir, when set, is the built frontend (frontend/dist) served by cmd/api in
	// production: the server then owns the app + API + socket as a single binary and
	// routes /api/* to the domain (no Vite to strip the prefix). Empty in dev, where
	// Vite serves the front and proxies /api + /socket.io.
	StaticDir string
	// WSVitalsWriteThroughLive mirrors every realtime vitals patch/delta back to the
	// Character row (clamped to the fresh max), so a mid-combat page refresh sees the
	// latest PV/PM. Opt-in (default off) — the session-end commit is the baseline.
	WSVitalsWriteThroughLive bool
}

// LoadConfig loads `.env.<APP_ENV>` (or ENV_FILE, when set) and reads the
// environment. APP_ENV defaults to development, so a bare `go run ./cmd/api`
// stays the dev setup it has always been.
//
//	APP_ENV=production ./bin/t20-api // → reads .env.production
func LoadConfig() (Config, error) {
	appEnv := AppEnv(env("APP_ENV", string(EnvDevelopment)))
	if err := loadEnvFile(env("ENV_FILE", ".env."+string(appEnv))); err != nil {
		return Config{}, err
	}
	return Config{
		AppEnv:       appEnv,
		AdminEmails:  splitEmails(os.Getenv("ADMIN_EMAILS")),
		Port:         env("PORT", "3001"),
		DatabasePath: stripFilePrefix(env("DATABASE_URL", "file:./data/t20-dev.db")),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		JWTExpiresIn: env("JWT_EXPIRES_IN", "7d"),
		CookieName:   env("COOKIE_NAME", "t20_session"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
		CORSOrigin:   env("CORS_ORIGIN", defaultCORSOrigin(appEnv)),
		BackupDir:    env("BACKUP_DIR", "../backups"),
		CatalogPath:  env("CATALOG_PATH", "parity/_catalogs.json"),
		StaticDir:    env("STATIC_DIR", ""),

		WSVitalsWriteThroughLive: os.Getenv("WS_VITALS_WRITETHROUGH_LIVE") == "1",
	}, nil
}

// Validate refuses a production boot that would misbehave in silence. Only the
// signing key qualifies today: empty or public, anyone who can reach the server
// mints their own session cookie and is every user at once. Development stays
// permissive on purpose — that is what makes it development.
func (c Config) Validate() error {
	if c.AppEnv != EnvProduction {
		return nil
	}
	if c.JWTSecret == "" || c.JWTSecret == DevJWTSecret {
		// Never echo the value: this error reaches logs the operator may paste.
		return fmt.Errorf(
			"JWT_SECRET is %s in %s — set your own in .env.production (openssl rand -hex 32)",
			secretFlaw(c.JWTSecret), c.AppEnv,
		)
	}
	// Registration needs an invite, and only an admin can issue one: a server
	// with no admin is a server nobody can ever join (ALE-120).
	if len(c.AdminEmails) == 0 {
		return fmt.Errorf("ADMIN_EMAILS is empty in %s — nobody could invite the players in", c.AppEnv)
	}
	return nil
}

// IsAdmin reports whether email administers the table. Case-insensitive, which
// is only safe because registration and login normalize the same way — without
// that, `Mestre@` could register as a SECOND account and be admin too.
func (c Config) IsAdmin(email string) bool {
	return slices.Contains(c.AdminEmails, normalizeEmail(email))
}

// splitEmails parses the comma-separated ADMIN_EMAILS, dropping blanks so a
// trailing comma or an empty variable yields no admin rather than an empty one.
func splitEmails(raw string) []string {
	var emails []string
	for _, part := range strings.Split(raw, ",") {
		if email := normalizeEmail(part); email != "" {
			emails = append(emails, email)
		}
	}
	return emails
}

func secretFlaw(secret string) string {
	if secret == "" {
		return "empty"
	}
	return "the public development secret"
}

// defaultCORSOrigin: dev needs the Vite origin (:5173) whitelisted because the
// SPA is served by a different port than the API. Production serves both from
// this binary, so the default is "no cross-origin caller".
func defaultCORSOrigin(appEnv AppEnv) string {
	if appEnv == EnvProduction {
		return ""
	}
	return "http://localhost:5173"
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
