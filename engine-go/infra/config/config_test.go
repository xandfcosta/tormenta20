package config

import (
	"strings"
	"testing"
)

// Quem tem a chave de assinatura cunha token para qualquer conta, então chave
// vazia ou pública em produção é porta aberta na rede local — o boot morre.
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

// Cadastro exige convite e só um admin cunha, então servidor de produção sem
// admin é servidor em que ninguém jamais entra.
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

// NENHUM ambiente libera origem por fábrica, e o teste existe para prender que os
// DOIS concordam.
//
// Um desenvolvimento MAIS PERMISSIVO que a produção esconde defeito — e um
// default sobrevive ao motivo que o criou: uma porta liberada para um servidor
// de front que não existe mais é CORS concedendo credenciais a uma origem que
// ninguém é dono. Um processo só serve tudo na mesma porta nos dois ambientes,
// então mesma-origem é a resposta certa nos dois.
func TestLoadConfigDefaultsCORSPerEnvironment(t *testing.T) {
	cases := map[AppEnv]string{
		EnvProduction:  "",
		EnvDevelopment: "",
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

// A lista existe para quem PRECISAR de origem externa: ela não tem valor de
// fábrica, e continua sendo lida. Espaço em branco e vírgula sobrando somem —
// uma origem VAZIA é pior que nenhuma, porque o go-chi lê lista vazia como
// "aceite TODAS".
func TestCORSOriginParsesAList(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
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
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SplitOrigins(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("SplitOrigins(%q) = %q, queria %q", tc.raw, got, tc.want)
			}
			for i, origin := range tc.want {
				if got[i] != origin {
					t.Errorf("origem %d = %q, queria %q", i, got[i], origin)
				}
			}
		})
	}
}

// Meio par de TLS derruba o boot, em QUALQUER ambiente.
//
// Cair para HTTP em silêncio é o pior dos mundos: quem escreveu meio par ligou
// `COOKIE_SECURE=true` junto, e aí o navegador DESCARTA o cookie de sessão. O
// login não conclui, não há erro em lugar nenhum, e a tela só volta ao início —
// e quem procura o defeito procura no login, não no `.env`.
func TestValidateRefusesHalfConfiguredTLS(t *testing.T) {
	cases := []struct {
		name, cert, key string
		wantErr         bool
	}{
		{name: "os dois vazios é HTTP puro, o padrão", cert: "", key: "", wantErr: false},
		{name: "o par inteiro é HTTPS", cert: "/etc/t20/cert.pem", key: "/etc/t20/key.pem", wantErr: false},
		{name: "certificado sem chave", cert: "/etc/t20/cert.pem", key: "", wantErr: true},
		{name: "chave sem certificado", cert: "", key: "/etc/t20/key.pem", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Desenvolvimento de propósito: é onde alguém experimenta HTTPS pela
			// primeira vez, e onde a validação de produção não olharia.
			cfg := Config{AppEnv: EnvDevelopment, TLSCertFile: tc.cert, TLSKeyFile: tc.key}

			err := cfg.Validate()

			if tc.wantErr != (err != nil) {
				t.Fatalf("Validate() = %v, querErro %v", err, tc.wantErr)
			}
			if err == nil {
				return
			}
			// O erro tem de nomear a variável QUE FALTA e mostrar o valor que
			// está lá — sem isso o dono relê o `.env` inteiro procurando o typo.
			if !strings.Contains(err.Error(), "TLS_CERT_FILE") || !strings.Contains(err.Error(), "TLS_KEY_FILE") {
				t.Errorf("o erro tem de nomear as duas variáveis, veio %q", err)
			}
			if filled := tc.cert + tc.key; !strings.Contains(err.Error(), filled) {
				t.Errorf("o erro tem de mostrar o caminho já escrito (%q), veio %q", filled, err)
			}
		})
	}
}

// O esquema é derivado do par, e não uma segunda variável que possa discordar
// dele: um `SCHEME=https` com TLS desligado seria mentira anunciada no log.
func TestSchemeFollowsTheCertificatePair(t *testing.T) {
	withoutTLS := Config{}
	withTLS := Config{TLSCertFile: "/etc/t20/cert.pem", TLSKeyFile: "/etc/t20/key.pem"}

	if got := withoutTLS.Scheme(); got != "http" {
		t.Errorf("Scheme() = %q sem certificado, esperava http", got)
	}
	if got := withTLS.Scheme(); got != "https" {
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
