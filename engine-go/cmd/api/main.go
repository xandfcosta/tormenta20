// Command api is the app's HTTP server. It opens + migrates the SQLite database
// and serves the domain routes the frontend consumes — via the Vite proxy in
// dev, and directly alongside the built SPA in production (STATIC_DIR).
//
// The environment comes from `.env.<APP_ENV>` next to the package (ALE-119):
// `air` boots it as development, `pnpm start` as production.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"t20engine/api"
	"t20engine/db"
	"t20engine/engine"
	"t20engine/plataforma"
)

// SONDA DE SAÚDE, para o contêiner ter como se examinar (ALE-273).
//
// A imagem é `distroless`: não tem shell, `curl` nem `wget`, e é por isso que a
// sonda mora AQUI. O `HEALTHCHECK` do compose chama o próprio binário com
// `-health`, ele bate no `/health` de si mesmo e sai 0 ou 1.
//
// Não é um segundo modo do programa: é uma requisição HTTP e um código de saída,
// e ela lê a MESMA `PORT` que o servidor escuta — apontar para uma porta escrita
// à mão daria uma sonda que reprova um servidor saudável no dia em que a porta
// mudasse.
func sondaDeSaude(cfg plataforma.Config) int {
	cliente := &http.Client{Timeout: 3 * time.Second}
	resp, err := cliente.Get(fmt.Sprintf("http://127.0.0.1:%s/health", cfg.Port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "health: %v\n", err)
		return 1
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "health: %s\n", resp.Status)
		return 1
	}
	return 0
}

func main() {
	cfg, err := plataforma.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	// A sonda sai ANTES de abrir o banco: ela é um cliente do servidor que já
	// está de pé, e abrir o SQLite de novo aqui poria um segundo escritor no
	// mesmo arquivo a cada 30 segundos.
	if len(os.Args) > 1 && os.Args[1] == "-health" {
		os.Exit(sondaDeSaude(cfg))
	}
	// Fatal, not a warning: a production boot with a forgeable signing key is
	// worse than no boot at all, and a warning scrolls away.
	if err := cfg.Validate(); err != nil {
		log.Fatalf("config: %v", err)
	}

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer func() { _ = database.Close() }()

	srv := api.NewServer(cfg, database, primeCatalogs(cfg.CatalogPath))
	mux := plataforma.Gzip(buildMux(srv))

	// Um sinal encerra a mesa com ordem, em vez de no meio de uma gravação
	// (ALE-157): sem isto, um Ctrl-C durante um `VACUUM INTO` ou um persist do
	// rastreador morria no meio, e o `defer database.Close()` acima NUNCA
	// rodava — o processo morre por sinal antes de qualquer defer.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go srv.ScheduleBackups(ctx)

	announce(cfg) // last, so the address to open is the final line on the screen
	if err := serve(ctx, cfg, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
	// O `Shutdown` do `net/http` espera as REQUISIÇÕES, e a gravação do estado
	// da sessão NÃO é uma: ela é disparada em goroutine depois da resposta, para
	// o mestre não esperar o disco no meio do turno. Sem esta linha o último
	// estado da noite pode ser cortado pelo `defer database.Close()` lá em cima
	// — justamente a gravação que a janela de 10s do `serve` existe para deixar
	// terminar, e que ela não alcança.
	//
	// Aqui e não dentro do `serve`: o `defer` do banco é de MAIN, e esperar tem
	// de acontecer antes dele. `log.Fatalf` acima pula os defers de qualquer
	// forma, mas esse caminho é o de erro de listener — não há mesa no ar para
	// perder.
	srv.EsperaOSegundoPlano()
}

// httpServerFor monta o servidor com os timeouts da casa. Separado da `serve`
// para os testes poderem afirmar as escolhas — inclusive a AUSÊNCIA do
// `WriteTimeout`, que é a mais fácil de alguém "consertar" sem saber.
func httpServerFor(cfg plataforma.Config, mux http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		// Dito com todas as letras porque o padrão do Go pode mudar de versão, e
		// porque um telefone antigo na mesa negociando TLS 1.0 seria uma queda
		// silenciosa de segurança. Ignorado quando não há TLS (ALE-118).
		TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
}

// serve sobe o HTTP e espera o sinal para desligar com ordem.
//
// Os timeouts são escolhidos, não copiados de um exemplo (ALE-157):
//
//   - `ReadHeaderTimeout` existe porque sem ele uma conexão que abre e nunca
//     manda o cabeçalho segura uma goroutine para sempre (slowloris);
//   - `IdleTimeout` recolhe conexões ociosas do keep-alive;
//   - `WriteTimeout` fica de FORA de propósito. Ele mataria o fluxo SSE, que é
//     conexão longa por natureza, e o download do wasm de 780 KB numa rede
//     ruim. É o timeout que parece obrigatório e é justamente o errado aqui.
func serve(ctx context.Context, cfg plataforma.Config, mux http.Handler) error {
	server := httpServerFor(cfg, mux)
	falhou := make(chan error, 1)
	go func() {
		if err := escutar(server, cfg); err != nil && !errors.Is(err, http.ErrServerClosed) {
			falhou <- err
		}
	}()

	select {
	case err := <-falhou:
		return err
	case <-ctx.Done():
	}

	log.Print("encerrando: esperando as requisições em curso")
	// A janela existe para a gravação em curso terminar; passado o prazo, o
	// desligamento continua — travar o encerramento seria trocar um problema
	// por outro.
	prazo, cancelar := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelar()
	if err := server.Shutdown(prazo); err != nil {
		log.Printf("encerramento forçado: %v", err)
	}
	return nil
}

// escutar sobe o listener: HTTPS quando o par de certificados está configurado,
// HTTP puro quando não (ALE-118). Os dois caminhos são o MESMO processo servindo
// SPA, API e fluxo de eventos — o TLS não acrescenta um segundo runtime.
//
// Um pedido `http://` chegando numa porta com TLS recebe "Client sent an HTTP
// request to an HTTPS server" do próprio net/http. Feio, mas VISÍVEL — que é o
// oposto do que acontecia se a configuração caísse para HTTP em silêncio.
func escutar(server *http.Server, cfg plataforma.Config) error {
	if !cfg.TLSEnabled() {
		return server.ListenAndServe()
	}
	// Os caminhos vão aqui, e não pré-carregados no `TLSConfig`: assim o erro
	// de um arquivo ausente ou ilegível NOMEIA o arquivo, e ele sobe pelo
	// `log.Fatalf("listen: …")` do main.
	return server.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
}

// primeCatalogs loads the rules catalogs for the mutation validators, best
// effort: auth + read + vitals work without them; item/creation writes need them.
func primeCatalogs(path string) *engine.Catalogs {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Printf("catalogs: %v — mutation validators disabled", err)
		return nil
	}
	catalogs, err := engine.PrimeEngineCatalogs(raw)
	if err != nil {
		log.Printf("catalogs: prime failed: %v", err)
		return nil
	}
	log.Printf("catalogs primed from %s", path)
	return catalogs
}

// buildMux monta o binário único: as cenas do piloto, a API em `/api/` e os
// endereços antigos, tudo na mesma porta.
//
// Ele já teve DOIS formatos — um de produção, que servia o `dist` da SPA na
// raiz, e um de desenvolvimento, em que o Vite servia o front e tirava o `/api`
// antes de encaminhar para cá. A SPA saiu na ALE-272 (fatia 10c) e os dois
// viraram um: não há mais front para servir nem proxy para atravessar, e o
// `STATIC_DIR` deixou de existir junto com o `spaHandler`.
//
// O socket.io também teve caminho PRÓPRIO aqui ("/socket.io/"), fora do
// `Router()` e por isso fora do CORS e do `requireAuth`. Com SSE o tempo real é
// uma rota como as outras e essa exceção sumiu (ALE-253). O padrão se repete:
// toda exceção neste mux acabou saindo.
func buildMux(srv *api.Server) *http.ServeMux {
	mux := http.NewServeMux()
	// O piloto Datastar (ALE-219): as cenas, renderizadas pelo servidor. Fora do
	// `/api` de propósito — o jogador abre e favorita esta URL.
	mux.Handle("/piloto/", http.StripPrefix("/piloto", srv.PilotoRouter()))
	// A PORTA DA FRENTE. `"/{$}"` casa a raiz EXATA; `"/"` casaria tudo.
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/piloto/", http.StatusFound)
	})
	// OS ENDEREÇOS ANTIGOS (ALE-272, fatia 10a). Eram cascas da SPA — um
	// `beforeLoad` que mandava para o piloto — e nessa forma morreriam com ela.
	api.MontaEnderecosAntigos(mux)
	// As FONTES que a folha pede por caminho absoluto (`/fonts/…`). Elas eram
	// servidas pelo `dist` da SPA em produção, e é por isso que o binário sem
	// SPA desenhava toda tela com uma serifada do sistema.
	mux.Handle("/fonts/", srv.FontesDoPiloto())
	mux.Handle("/favicon.svg", srv.FaviconDoPiloto())
	// A API JSON fica sob `/api/`, e agora em TODO ambiente. Ela vivia na raiz
	// em desenvolvimento porque o Vite tirava o prefixo antes de encaminhar; sem
	// Vite, dois endereços para a mesma API seriam duas coisas para lembrar.
	mux.Handle("/api/", http.StripPrefix("/api", srv.Router()))
	return mux
}

// announce diz onde apontar o navegador, com os endereços da REDE junto: os
// jogadores abrem o app das máquinas deles, e sem esta linha o dono da mesa
// teria de ir ler `ip addr` (ALE-119).
func announce(cfg plataforma.Config) {
	log.Printf("t20 %s server listening on :%s (%s, db=%s)", cfg.AppEnv, cfg.Port, cfg.Scheme(), cfg.DatabasePath)
	if cfg.TLSEnabled() && !cfg.CookieSecure {
		log.Print("  aviso: há TLS e COOKIE_SECURE=false — o cookie de sessão viaja sem a marca Secure")
	}
	for _, url := range lanURLs(cfg) {
		log.Printf("  players can open %s", url)
	}
}

// lanURLs lists this host's non-loopback IPv4 addresses as URLs. The server
// binds every interface, so these already work — they are just not discoverable
// from the log line above.
//
// O esquema vem da config, e não é detalhe: este log É o endereço que o mestre
// lê e repassa para a mesa. Com TLS ligado e `http://` impresso, os quatro
// telefones batem num 400 e o sintoma parece do app (ALE-118).
func lanURLs(cfg plataforma.Config) []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("interfaces: %v — LAN address unknown", err)
		return nil
	}
	var urls []string
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue
		}
		urls = append(urls, fmt.Sprintf("%s://%s:%s", cfg.Scheme(), ipNet.IP, cfg.Port))
	}
	return urls
}

// tiposFora são as extensões que a tabela do `mime` do Go não conhece e que
// este servidor precisa acertar. Uma tabela nossa, e não `mime.AddExtensionType`
// no boot, porque o `mime` também lê o `/etc/mime.types` do HOST: o mesmo
// binário responderia diferente em duas máquinas, e o teste passaria verde na
// que tem o arquivo (ALE-118).
var tiposFora = map[string]string{
	// Sem isto o manifest sai como `text/plain`, adivinhado pelo CONTEÚDO. O
	// Chromium engole isso — medido: servido como `text/plain` ele mesmo assim
	// parseia o manifest inteiro, sem erro. O tipo entra aqui porque é o que a
	// especificação exige (um tipo de JSON) e porque não é o Chrome que decide
	// sozinho: quem depende do manifest é também o Safari e o Firefox, e um
	// `text/plain` é a diferença entre "funciona" e "funciona neste navegador".
	".webmanifest": "application/manifest+json",
}

// contentTypeFor devolve o tipo do arquivo pelo nome, preferindo a tabela da
// casa à do sistema.
func contentTypeFor(file string) string {
	ext := filepath.Ext(file)
	if ctype, ok := tiposFora[ext]; ok {
		return ctype
	}
	return mime.TypeByExtension(ext)
}

// cacheControlFor decide por quanto tempo o navegador pode guardar (ALE-157).
//
// O Vite carimba um hash no nome de cada asset (`admin-BvY0grFM.js`), então
// aquele arquivo NUNCA muda: mudar o conteúdo muda o nome. Isso é o que
// autoriza `immutable` por um ano — sem hash no nome seria mentira, e um
// jogador ficaria com a versão velha até limpar o cache.
//
// O resto revalida a cada carga: o `index.html` porque é ele quem aponta para
// os assets novos (guardá-lo congelaria o app inteiro numa versão), e o
// `t20.wasm` porque não é hasheado — a revalidação custa um 304 vazio, que o
// `ServeFile` já responde pelo `Last-Modified`.
func cacheControlFor(file string) string {
	if strings.Contains(filepath.ToSlash(file), "/assets/") {
		return "public, max-age=31536000, immutable"
	}
	return "no-cache"
}
