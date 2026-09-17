// O pacote platform é o que NÃO é domínio: configuração, `.env`, compressão,
// negociação de codificação e os ajudantes de corpo e resposta HTTP.
//
// Ele não importa nada do projeto — é folha do grafo, como o `engine` —, e essa
// é a propriedade que o mantém honesto: qualquer coisa que precise saber o que é
// uma ficha ou um tabuleiro não cabe aqui.
package config

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"t20engine/infra/wire"
)

// AppEnv nomeia o ambiente. Ele escolhe qual `.env.<AppEnv>` o `LoadConfig` lê E
// quão rígido o `Validate` é — as duas coisas são uma decisão só, e por isso
// dividem uma variável (APP_ENV) em vez de derivar cada uma para um lado.
type AppEnv string

const (
	EnvDevelopment AppEnv = "development"
	EnvProduction  AppEnv = "production"
)

// DevJWTSecret é a chave de assinatura descartável que vem no
// `.env.development`. Ela é versionada de propósito — token de desenvolvimento
// não vale nada —, e é exatamente por isso que o `Validate` a recusa em
// produção: um arquivo copiado não pode virar a chave que assina a sessão de
// todo mundo na LAN.
const DevJWTSecret = "t20-dev-secret"

// Config é o ambiente do servidor, lido uma vez na subida.
type Config struct {
	AppEnv AppEnv
	// AdminEmails é a lista fechada de contas que administram a mesa. O papel
	// mora AQUI e não numa coluna do banco de propósito: não existe rota que
	// promova ninguém, então o único jeito de virar admin é editar este arquivo
	// no hospedeiro — nenhum defeito de HTTP transforma um jogador num. O preço
	// é que mudar a lista custa uma edição mais um reinício.
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
	// ligadas é todo site do mundo. `CORS_ORIGIN` no `.env` volta a montar o
	// middleware com a lista que se escrever.
	CORSOrigins []string
	// CatalogPath é o despejo de catálogos (itens, raças, …) que a API carrega na
	// subida para os validadores de mutação. O padrão é o instantâneo
	// versionado.
	CatalogPath string
	// BackupDir é onde a tela de admin escreve os instantâneos — a mesma pasta do
	// script `pnpm db:backup`, para um backup feito de qualquer um dos dois jeitos
	// aparecer nos dois lugares. Relativa a `engine-go/`, que é o CWD do
	// servidor.
	BackupDir string
	// BackupEvery é o intervalo do backup automático, e BackupKeep quantos
	// arquivos ficam. Um backup que depende de alguém lembrar é um backup que
	// não existe na noite em que importa. Zero em qualquer um dos dois DESLIGA o
	// automático — a mesa é do dono, e ele pode não querer.
	BackupEvery time.Duration
	BackupKeep  int
	// TLSCertFile e TLSKeyFile ligam o HTTPS NESTE processo. Vazios nos dois — o
	// padrão — o servidor fala HTTP puro.
	//
	// O TLS termina aqui e não num nginx/Caddy na frente porque a decisão da casa
	// é um processo só, sem proxy; pôr um na frente contraria isso e precisa ser
	// deliberado. Isto NÃO exclui o outro arranjo: quem terminar TLS fora deixa
	// estes dois vazios, mantém `COOKIE_SECURE=true` e continua funcionando.
	TLSCertFile string
	TLSKeyFile  string
	// BookPDF é o caminho do Tormenta 20 em PDF que o servidor entrega em
	// `/livro`, e VAZIO é o padrão: sem ele o botão "abrir no livro"
	// simplesmente não existe, e nada é servido.
	//
	// Por configuração e não embutido: o PDF está FORA do módulo Go
	// (`../t20-book.pdf`, e ignorado pelo git) e o `go:embed` não o alcança. E
	// servir o livro é decisão do dono da mesa — a rota publica o arquivo para
	// quem entrou na rede local.
	BookPDF string
	// BookPageOffset é quantas páginas o ARQUIVO tem antes da página impressa 1.
	//
	// Ela existe porque `#page=N` conta páginas do ARQUIVO e o catálogo grava a
	// página IMPRESSA (`bookPage`). Sem ela o botão abre seis páginas antes, no
	// MESMO capítulo, que é o tipo de erro que parece certo.
	BookPageOffset int
}

// LoadConfig lê o `.env.<APP_ENV>` (ou o ENV_FILE, quando escrito) e o
// ambiente. APP_ENV cai em development, para um `go run ./cmd/api` pelado
// continuar sendo o arranjo de desenvolvimento.
//
//	APP_ENV=production ./bin/t20-api // → lê o .env.production
func LoadConfig() (Config, error) {
	appEnv := AppEnv(env("APP_ENV", string(EnvDevelopment)))
	if err := LoadEnvFile(env("ENV_FILE", ".env."+string(appEnv))); err != nil {
		return Config{}, err
	}
	return Config{
		AppEnv:         appEnv,
		AdminEmails:    splitEmails(os.Getenv("ADMIN_EMAILS")),
		Port:           env("PORT", "3001"),
		DatabasePath:   stripFilePrefix(env("DATABASE_URL", "file:./data/t20-dev.db")),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTExpiresIn:   env("JWT_EXPIRES_IN", "7d"),
		CookieName:     env("COOKIE_NAME", "t20_session"),
		CookieSecure:   os.Getenv("COOKIE_SECURE") == "true",
		CORSOrigins:    SplitOrigins(env("CORS_ORIGIN", "")),
		BackupDir:      env("BACKUP_DIR", "../backups"),
		BackupEvery:    envDuration("BACKUP_EVERY", 24*time.Hour),
		BackupKeep:     envInt("BACKUP_KEEP", 7),
		CatalogPath:    env("CATALOG_PATH", "parity/_catalogs.json"),
		TLSCertFile:    os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:     os.Getenv("TLS_KEY_FILE"),
		BookPDF:        env("LIVRO_PDF", ""),
		BookPageOffset: envInt("LIVRO_ABERTURA", 6),
	}, nil
}

// Validate recusa uma subida que se comportaria mal em silêncio. Em produção é
// a chave de assinatura (vazia ou pública, qualquer um que alcance o servidor
// emite o próprio cookie, e é todo mundo de uma vez) e a lista de admins. Em
// QUALQUER ambiente é o par de TLS pela metade — ver `validateTLS`. Fora disso o
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
		// Nunca ecoe o valor: este erro cai em log que o operador pode colar.
		return fmt.Errorf(
			"JWT_SECRET is %s in %s — set your own in .env.production (openssl rand -hex 32)",
			secretFlaw(c.JWTSecret), c.AppEnv,
		)
	}
	// Cadastro precisa de convite, e só um admin emite: servidor sem admin é
	// servidor em que ninguém nunca entra.
	if len(c.AdminEmails) == 0 {
		return fmt.Errorf("ADMIN_EMAILS is empty in %s — nobody could invite the players in", c.AppEnv)
	}
	return nil
}

// TLSEnabled diz se é este processo que termina o TLS.
func (c Config) TLSEnabled() bool {
	return c.TLSCertFile != "" && c.TLSKeyFile != ""
}

// Scheme é o que vem antes do endereço que os jogadores digitam. Ele existe para
// a linha de log e a URL serem a mesma decisão: um servidor em HTTPS anunciando
// `http://` manda a mesa inteira para um endereço que responde 400.
func (c Config) Scheme() string {
	if c.TLSEnabled() {
		return "https"
	}
	return "http"
}

// validateTLS recusa um par de TLS pela metade. Cair para HTTP em silêncio seria
// o pior dos mundos: quem escreveu meio par ligou `COOKIE_SECURE=true` junto, e
// aí o navegador DESCARTA o cookie de sessão — o login não conclui, sem erro em
// lugar nenhum, e a tela só volta para o início.
func (c Config) validateTLS() error {
	if (c.TLSCertFile == "") == (c.TLSKeyFile == "") {
		return nil
	}
	missing, present, value := "TLS_KEY_FILE", "TLS_CERT_FILE", c.TLSCertFile
	if c.TLSCertFile == "" {
		missing, present, value = "TLS_CERT_FILE", "TLS_KEY_FILE", c.TLSKeyFile
	}
	return fmt.Errorf(
		"%s está vazio e %s=%q — o HTTPS precisa dos DOIS caminhos; deixe os dois vazios para servir HTTP",
		missing, present, value,
	)
}

// IsAdmin diz se o e-mail administra a mesa. Ignora a caixa, o que só é seguro
// porque cadastro e login normalizam do mesmo jeito — sem isso, `Mestre@` se
// cadastraria como uma SEGUNDA conta e seria admin também.
func (c Config) IsAdmin(email string) bool {
	return slices.Contains(c.AdminEmails, wire.NormalizeEmail(email))
}

// splitEmails lê o ADMIN_EMAILS separado por vírgula, descartando os vazios:
// vírgula sobrando ou variável vazia dá NENHUM admin, e não um admin vazio.
func splitEmails(raw string) []string {
	var emails []string
	for _, part := range strings.Split(raw, ",") {
		if email := wire.NormalizeEmail(part); email != "" {
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

// SplitOrigins lê o CORS_ORIGIN separado por vírgula, descartando os vazios:
// vírgula sobrando tem de dar NENHUMA origem, e não uma origem vazia — o go-chi
// lê um `AllowedOrigins` vazio como "libere TODAS", que com credenciais ligadas
// é todo site do mundo.
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

// stripFilePrefix transforma uma URL no estilo "file:./dev.db" num caminho.
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
