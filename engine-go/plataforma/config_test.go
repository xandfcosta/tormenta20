package plataforma

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
	sandboxEnv(t, "PORT", "DATABASE_URL", "JWT_SECRET", "CORS_ORIGIN", "COOKIE_SECURE", "ADMIN_EMAILS")
	t.Setenv("APP_ENV", string(EnvProduction))
	t.Setenv("ENV_FILE", writeEnvFile(t, strings.Join([]string{
		"PORT=8080",
		"DATABASE_URL=file:./data/t20-prod.db",
		"JWT_SECRET=6f1c1a0d9e2b",
		"ADMIN_EMAILS=Dono@T20.local",
	}, "\n")))

	cfg, err := LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.AppEnv != EnvProduction || cfg.Port != "8080" {
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
		EnvDevelopment: DevCORSOrigins,
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
			if got := strings.Join(cfg.CORSOrigins, ","); got != want {
				t.Errorf("CORSOrigins = %q, want %q", got, want)
			}
		})
	}
}

// A lista é o que permite os três apelidos do mesmo servidor de dev (e o IP da
// LAN, quando alguém abre a mesa pela rede em desenvolvimento). Espaço em
// branco e vírgula sobrando somem: uma origem VAZIA seria pior que nenhuma,
// porque o go-chi lê lista vazia como "aceite TODAS" (ALE-119).
func TestCORSOriginParsesAList(t *testing.T) {
	casos := []struct {
		nome string
		raw  string
		quer []string
	}{
		{"uma só", "http://localhost:5173", []string{"http://localhost:5173"}},
		{
			"os apelidos do loopback",
			"http://localhost:5173, http://[::1]:5173 ,http://127.0.0.1:5173",
			[]string{"http://localhost:5173", "http://[::1]:5173", "http://127.0.0.1:5173"},
		},
		{"vírgula sobrando", "http://localhost:5173,,", []string{"http://localhost:5173"}},
		{"vazio não vira origem vazia", "  ,  ", nil},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			got := SplitOrigins(caso.raw)
			if len(got) != len(caso.quer) {
				t.Fatalf("SplitOrigins(%q) = %q, queria %q", caso.raw, got, caso.quer)
			}
			for i, origem := range caso.quer {
				if got[i] != origem {
					t.Errorf("origem %d = %q, queria %q", i, got[i], origem)
				}
			}
		})
	}
}

// Meio par de TLS derruba o boot, em QUALQUER ambiente (ALE-118).
//
// Cair para HTTP em silêncio é o pior dos mundos: quem escreveu meio par ligou
// `COOKIE_SECURE=true` junto, e aí o navegador DESCARTA o cookie de sessão. O
// login não conclui, não há erro em lugar nenhum, e a tela só volta ao início —
// e quem procura o defeito procura no login, não no `.env`.
func TestValidateRefusesHalfConfiguredTLS(t *testing.T) {
	casos := []struct {
		nome, cert, key string
		querErro        bool
	}{
		{nome: "os dois vazios é HTTP puro, o padrão", cert: "", key: "", querErro: false},
		{nome: "o par inteiro é HTTPS", cert: "/etc/t20/cert.pem", key: "/etc/t20/key.pem", querErro: false},
		{nome: "certificado sem chave", cert: "/etc/t20/cert.pem", key: "", querErro: true},
		{nome: "chave sem certificado", cert: "", key: "/etc/t20/key.pem", querErro: true},
	}
	for _, tc := range casos {
		t.Run(tc.nome, func(t *testing.T) {
			// Desenvolvimento de propósito: é onde alguém experimenta HTTPS pela
			// primeira vez, e onde a validação de produção não olharia.
			cfg := Config{AppEnv: EnvDevelopment, TLSCertFile: tc.cert, TLSKeyFile: tc.key}

			err := cfg.Validate()

			if tc.querErro != (err != nil) {
				t.Fatalf("Validate() = %v, querErro %v", err, tc.querErro)
			}
			if err == nil {
				return
			}
			// O erro tem de nomear a variável QUE FALTA e mostrar o valor que
			// está lá — sem isso o dono relê o `.env` inteiro procurando o typo.
			if !strings.Contains(err.Error(), "TLS_CERT_FILE") || !strings.Contains(err.Error(), "TLS_KEY_FILE") {
				t.Errorf("o erro tem de nomear as duas variáveis, veio %q", err)
			}
			if preenchido := tc.cert + tc.key; !strings.Contains(err.Error(), preenchido) {
				t.Errorf("o erro tem de mostrar o caminho já escrito (%q), veio %q", preenchido, err)
			}
		})
	}
}

// O esquema é derivado do par, e não uma segunda variável que possa discordar
// dele: um `SCHEME=https` com TLS desligado seria mentira anunciada no log.
func TestSchemeFollowsTheCertificatePair(t *testing.T) {
	semTLS := Config{}
	comTLS := Config{TLSCertFile: "/etc/t20/cert.pem", TLSKeyFile: "/etc/t20/key.pem"}

	if got := semTLS.Scheme(); got != "http" {
		t.Errorf("Scheme() = %q sem certificado, esperava http", got)
	}
	if got := comTLS.Scheme(); got != "https" {
		t.Errorf("Scheme() = %q com o par completo, esperava https", got)
	}
}

// O par vem do ambiente como qualquer outra configuração — sem isto o default
// existe só no Go e ninguém o descobre.
func TestLoadConfigReadsTheCertificatePair(t *testing.T) {
	sandboxEnv(t, "TLS_CERT_FILE", "TLS_KEY_FILE")
	t.Setenv("ENV_FILE", writeEnvFile(t, strings.Join([]string{
		"TLS_CERT_FILE=/etc/t20/cert.pem",
		"TLS_KEY_FILE=/etc/t20/key.pem",
	}, "\n")))

	cfg, err := LoadConfig()

	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.TLSEnabled() {
		t.Fatalf("TLSEnabled() falso com o par no arquivo: cert=%q key=%q", cfg.TLSCertFile, cfg.TLSKeyFile)
	}
}
