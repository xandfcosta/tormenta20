// O pacote platform é o que NÃO é domínio: configuração, `.env`, compressão,
// negociação de codificação e os ajudantes de corpo e resposta HTTP.
//
// Ele não importa nada do projeto — é folha do grafo, como o `engine` —, e essa
// é a propriedade que o mantém honesto: qualquer coisa que precise saber o que é
// uma ficha ou um tabuleiro não cabe aqui.
//
// > Aqui morava um comentário abrindo com `Package api is the app's HTTP layer`,
// > descrevendo chi router, middleware e handlers — nenhum deles deste pacote.
// > Ele sobreviveu ao arquivo mudar de `api/` para cá, e uma linha em branco o
// > separava do `package`, então nem doc comment do Go ele era (ALE-321).
package platform

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
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
	// CORSOrigins são as origens de navegador liberadas a chamar a API de FORA
	// da origem dela. Vazio é o normal, e é o normal nos DOIS ambientes: um
	// processo só serve as cenas, a API e o fluxo ao vivo na mesma porta, então
	// toda chamada é mesma-origem e nenhum outro site tem o que fazer aqui.
	//
	// Vazio quer dizer NENHUM middleware de CORS montado, e não uma lista vazia:
	// o go-chi lê `AllowedOrigins` vazio como "libere TODAS", que com credenciais
	// ligadas é todo site do mundo (ALE-119).
	//
	// Ela continua existindo para quem precisar: `CORS_ORIGIN` no `.env` volta a
	// montar o middleware com a lista que se escrever.
	CORSOrigins []string
	// CatalogPath is the primeEngineCatalogs payload (items/races/…) the API loads
	// at startup for its mutation validators. Defaults to the committed snapshot.
	CatalogPath string
	// BackupDir is where the admin screen writes snapshots — the same directory
	// the `pnpm db:backup` script uses, so a backup made either way shows up in
	// both places (ALE-120). Relative to engine-go/, which is the server's CWD.
	BackupDir string
	// BackupEvery é o intervalo do backup automático, e BackupKeep quantos
	// arquivos ficam. Um backup que depende de alguém lembrar é um backup que
	// não existe na noite em que importa (ALE-157). Zero em qualquer um dos
	// dois DESLIGA o automático — a mesa é do dono, e ele pode não querer.
	BackupEvery time.Duration
	BackupKeep  int
	// TLSCertFile e TLSKeyFile ligam o HTTPS NESTE processo (ALE-118). Vazios
	// nos dois — o padrão — o servidor fala HTTP puro exatamente como antes.
	//
	// O TLS termina aqui, e não num nginx/Caddy na frente, porque a decisão
	// registrada no `engine-go/CLAUDE.md` é um processo só servindo SPA, API e
	// socket; pôr um proxy na frente contraria isso e precisa ser deliberado.
	//
	// Isto NÃO exclui o outro arranjo: quem terminar TLS fora (um túnel, um
	// proxy) deixa estes dois vazios, mantém `COOKIE_SECURE=true` e continua
	// funcionando — o processo segue falando HTTP para quem está na frente.
	TLSCertFile string
	TLSKeyFile  string
	// LivroPDF é o caminho do Tormenta 20 em PDF que o servidor entrega em
	// `/livro`, e VAZIO é o padrão: sem ele o botão "abrir no livro"
	// simplesmente não existe, e nada é servido.
	//
	// Por configuração e não embutido porque o PDF está FORA do módulo Go
	// (`../t20-book.pdf`, e ignorado pelo git) — `go:embed` não o alcança —,
	// e porque servir o livro é decisão do dono da mesa: a mesa conecta pela
	// rede local, então a rota publica o arquivo para quem entrou.
	LivroPDF string
	// LivroAbertura é quantas páginas o ARQUIVO tem antes da página impressa 1.
	//
	// Ela existe porque `#page=N` conta páginas do arquivo e o catálogo grava a
	// página IMPRESSA (`bookPage`). MEDIDO no PDF da casa (407 páginas), pelo
	// número no RODAPÉ: a página 295 do arquivo imprime "289", a 297 imprime
	// "291" e a 203 imprime "197" — três amostras em regiões distantes, todas
	// com abertura 6. Sem isto o botão abriria seis páginas antes, no MESMO
	// capítulo, que é o tipo de erro que parece certo.
	LivroAbertura int
}

// LoadConfig loads `.env.<APP_ENV>` (or ENV_FILE, when set) and reads the
// environment. APP_ENV defaults to development, so a bare `go run ./cmd/api`
// stays the dev setup it has always been.
//
//	APP_ENV=production ./bin/t20-api // → reads .env.production
func LoadConfig() (Config, error) {
	appEnv := AppEnv(env("APP_ENV", string(EnvDevelopment)))
	if err := LoadEnvFile(env("ENV_FILE", ".env."+string(appEnv))); err != nil {
		return Config{}, err
	}
	return Config{
		AppEnv:        appEnv,
		AdminEmails:   splitEmails(os.Getenv("ADMIN_EMAILS")),
		Port:          env("PORT", "3001"),
		DatabasePath:  stripFilePrefix(env("DATABASE_URL", "file:./data/t20-dev.db")),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		JWTExpiresIn:  env("JWT_EXPIRES_IN", "7d"),
		CookieName:    env("COOKIE_NAME", "t20_session"),
		CookieSecure:  os.Getenv("COOKIE_SECURE") == "true",
		CORSOrigins:   SplitOrigins(env("CORS_ORIGIN", "")),
		BackupDir:     env("BACKUP_DIR", "../backups"),
		BackupEvery:   envDuration("BACKUP_EVERY", 24*time.Hour),
		BackupKeep:    envInt("BACKUP_KEEP", 7),
		CatalogPath:   env("CATALOG_PATH", "parity/_catalogs.json"),
		TLSCertFile:   os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:    os.Getenv("TLS_KEY_FILE"),
		LivroPDF:      env("LIVRO_PDF", ""),
		LivroAbertura: envInt("LIVRO_ABERTURA", 6),
	}, nil
}

// Validate refuses a boot that would misbehave in silence. Em produção é a chave
// de assinatura (vazia ou pública, qualquer um que alcance o servidor emite o
// próprio cookie e é todo mundo de uma vez) e a lista de admins. Em QUALQUER
// ambiente é o par de TLS pela metade — ver validateTLS. Fora disso o
// desenvolvimento segue permissivo de propósito: é o que o faz desenvolvimento.
func (c Config) Validate() error {
	// ANTES do desvio de desenvolvimento: um par de TLS pela metade é erro de
	// digitação em qualquer ambiente, e é justamente em desenvolvimento que
	// alguém experimenta o HTTPS pela primeira vez.
	if err := c.validateTLS(); err != nil {
		return err
	}
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

// TLSEnabled reports whether this process terminates TLS itself.
func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// Scheme is what goes before the address the players type. It exists so the
// log line and the URL are the same decision: um servidor em HTTPS anunciando
// `http://` manda a mesa inteira para um endereço que responde 400 (ALE-118).
func (c Config) Scheme() string {
	if c.TLSEnabled() {
		return "https"
	}
	return "http"
}

// validateTLS recusa um par de TLS pela metade. Cair para HTTP em silêncio
// seria o pior dos mundos: quem escreveu meio par ligou `COOKIE_SECURE=true`
// junto, e aí o navegador DESCARTA o cookie de sessão — o login não conclui,
// sem erro em lugar nenhum, e a tela só volta para o início (ALE-118).
func (c Config) validateTLS() error {
	if (c.TLSCertFile == "") == (c.TLSKeyFile == "") {
		return nil
	}
	faltando, presente, valor := "TLS_KEY_FILE", "TLS_CERT_FILE", c.TLSCertFile
	if c.TLSCertFile == "" {
		faltando, presente, valor = "TLS_CERT_FILE", "TLS_KEY_FILE", c.TLSKeyFile
	}
	return fmt.Errorf(
		"%s está vazio e %s=%q — o HTTPS precisa dos DOIS caminhos; deixe os dois vazios para servir HTTP",
		faltando, presente, valor,
	)
}

// IsAdmin reports whether email administers the table. Case-insensitive, which
// is only safe because registration and login normalize the same way — without
// that, `Mestre@` could register as a SECOND account and be admin too.
func (c Config) IsAdmin(email string) bool {
	return slices.Contains(c.AdminEmails, NormalizeEmail(email))
}

// splitEmails parses the comma-separated ADMIN_EMAILS, dropping blanks so a
// trailing comma or an empty variable yields no admin rather than an empty one.
func splitEmails(raw string) []string {
	var emails []string
	for _, part := range strings.Split(raw, ",") {
		if email := NormalizeEmail(part); email != "" {
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

// Aqui morava o `DevCORSOrigins`, e com ele o único lugar em que o ambiente de
// DESENVOLVIMENTO era mais permissivo que o de produção (ALE-321).
//
// Ele liberava `http://localhost:5173` nas três grafias que o navegador trata
// como origens diferentes, e a razão era real enquanto durou: a SPA era servida
// pelo Vite naquela porta, a API respondia na :3001, e a grafia que ficasse de
// fora perdia o socket para um 403 que chegava na tela como "RECONECTANDO…" para
// sempre, sem erro em lugar nenhum (ALE-185).
//
// A SPA saiu na ALE-272. Desde então nada roda na :5173, e o que sobrou foi um
// middleware de CORS montado em desenvolvimento concedendo credenciais a uma
// origem que ninguém é dono — configuração morta que ainda FAZ coisa, que é pior
// que prosa morta.
//
// Hoje o default é vazio nos dois ambientes, e vazio quer dizer NENHUM
// middleware: um processo só serve as cenas, a API e o fluxo ao vivo na mesma
// porta, então o celular do jogador na LAN é mesma-origem e passa direto. Quem
// precisar de uma origem externa escreve `CORS_ORIGIN` no `.env` — a variável
// continua lida, só deixou de ter um valor de fábrica que ninguém pediu.

// SplitOrigins parses the comma-separated CORS_ORIGIN, dropping blanks: a
// trailing comma must yield NO origin rather than an empty one, and an empty
// one is worse than none — go-chi reads an empty AllowedOrigins list as "allow
// ALL", which with credentials on is every website (ALE-119).
func SplitOrigins(raw string) []string {
	var origins []string
	for _, part := range strings.Split(raw, ",") {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
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

// envDuration lê uma duração ("24h", "30m"). Valor inválido cai no padrão com
// aviso, em vez de derrubar o boot: um erro de digitação no `.env` não pode
// impedir a mesa de começar.
func envDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("config: %s=%q não é uma duração válida; usando %s", key, raw, fallback)
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		log.Printf("config: %s=%q não é um número; usando %d", key, raw, fallback)
		return fallback
	}
	return parsed
}
