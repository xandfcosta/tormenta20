package api

import (
	"strings"
	"testing"
)

// Whoever holds the signing key can mint a token for any account, so an empty
// or public key in production is an open door on the LAN — the boot dies (ALE-119).
func TestValidateRefusesProductionWithoutItsOwnSecret(t *testing.T) {
	cases := []struct {
		name, secret string
		wantErr      bool
	}{
		{name: "empty signs with no key", secret: "", wantErr: true},
		{name: "the committed dev secret is public", secret: DevJWTSecret, wantErr: true},
		{name: "its own secret", secret: "6f1c1a0d9e2b", wantErr: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{AppEnv: EnvProduction, JWTSecret: tc.secret, AdminEmails: []string{"dono@t20.local"}}

			err := cfg.Validate()
			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), tc.secret) && tc.secret != "" {
				t.Errorf("the error must not echo the secret, got %q", err)
			}
		})
	}
}

// Registration needs an invite and only an admin issues one, so a production
// server with no admin is one nobody could ever join (ALE-120).
func TestValidateRefusesProductionWithoutAnAdmin(t *testing.T) {
	cfg := Config{AppEnv: EnvProduction, JWTSecret: "6f1c1a0d9e2b"}

	err := cfg.Validate()

	if err == nil {
		t.Fatal("esperado erro sem ADMIN_EMAILS em produção")
	}
	if !strings.Contains(err.Error(), "ADMIN_EMAILS") {
		t.Errorf("o erro tem de nomear a variável, veio %q", err)
	}
}

// Development stays permissive — that is what makes it development.
func TestValidateAcceptsDevelopmentWithoutASecret(t *testing.T) {
	if err := (Config{AppEnv: EnvDevelopment}).Validate(); err != nil {
		t.Fatalf("development must boot without a secret: %v", err)
	}
}

func TestLoadConfigReadsTheEnvironmentFile(t *testing.T) {
	sandboxEnv(t, "PORT", "DATABASE_URL", "JWT_SECRET", "STATIC_DIR", "CORS_ORIGIN", "COOKIE_SECURE", "ADMIN_EMAILS")
	t.Setenv("APP_ENV", string(EnvProduction))
	t.Setenv("ENV_FILE", writeEnvFile(t, strings.Join([]string{
		"PORT=8080",
		"DATABASE_URL=file:./data/t20-prod.db",
		"JWT_SECRET=6f1c1a0d9e2b",
		"STATIC_DIR=../frontend/dist",
		"ADMIN_EMAILS=Dono@T20.local",
	}, "\n")))

	cfg, err := LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AppEnv != EnvProduction || cfg.Port != "8080" || cfg.StaticDir != "../frontend/dist" {
		t.Errorf("LoadConfig() = %+v, want the production file's values", cfg)
	}
	// The DSN's "file:" prefix is stripped for the SQLite driver.
	if cfg.DatabasePath != "./data/t20-prod.db" {
		t.Errorf("DatabasePath = %q, want %q", cfg.DatabasePath, "./data/t20-prod.db")
	}
	// Normalized on the way in, which is what lets IsAdmin ignore case safely.
	if !cfg.IsAdmin("dono@t20.local") || !cfg.IsAdmin("DONO@t20.local") {
		t.Errorf("AdminEmails = %q, esperado reconhecer o dono em qualquer caixa", cfg.AdminEmails)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("the shipped production shape must validate: %v", err)
	}
}

// In production this binary serves the SPA, so every call is same-origin and no
// origin is whitelisted; dev needs the Vite one because the SPA is on :5173.
func TestLoadConfigDefaultsCORSPerEnvironment(t *testing.T) {
	cases := map[AppEnv]string{
		EnvProduction:  "",
		EnvDevelopment: "http://localhost:5173",
	}
	for appEnv, want := range cases {
		t.Run(string(appEnv), func(t *testing.T) {
			sandboxEnv(t, "CORS_ORIGIN")
			t.Setenv("APP_ENV", string(appEnv))
			t.Setenv("ENV_FILE", writeEnvFile(t, "# no CORS_ORIGIN here\n"))

			cfg, err := LoadConfig()

			if err != nil {
				t.Fatalf("LoadConfig: %v", err)
			}
			if cfg.CORSOrigin != want {
				t.Errorf("CORSOrigin = %q, want %q", cfg.CORSOrigin, want)
			}
		})
	}
}
