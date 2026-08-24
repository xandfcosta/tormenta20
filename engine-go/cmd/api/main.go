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

func main() {
	cfg, err := plataforma.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
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
	mux := buildMux(cfg, srv)

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

// buildMux monta ou o binário único de produção (SPA + /api/* na mesma porta),
// ou a forma de desenvolvimento, em que o Vite serve o front e tira o /api antes
// de encaminhar para cá.
//
// O socket.io tinha caminho PRÓPRIO aqui ("/socket.io/"), fora do `Router()` e
// por isso fora do CORS e do `requireAuth` — o que obrigava o `guardSocketOrigin`
// a repetir a política de origem por conta. Com SSE o tempo real é uma rota como
// as outras e essa exceção sumiu (ALE-253).
func buildMux(cfg plataforma.Config, srv *api.Server) *http.ServeMux {
	mux := http.NewServeMux()
	// O piloto Datastar (ALE-219): uma PÁGINA renderizada pelo servidor, ao lado
	// da SPA. Fora do `/api` de propósito — o jogador abre e favorita esta URL.
	// Vive nos dois formatos porque o `ServeMux` casa pelo prefixo mais longo.
	// Apagar esta linha é metade da saída do piloto.
	mux.Handle("/piloto/", http.StripPrefix("/piloto", srv.PilotoRouter()))
	// A PORTA DA FRENTE é do servidor desde a ALE-231, e o desvio acontece AQUI
	// e não dentro da SPA por uma razão medida: a rota `/` dela fazia o desvio em
	// JavaScript, o que obriga o navegador a baixar e executar o aplicativo
	// inteiro só para sair dele. Em desenvolvimento isso são ~1600 módulos — o
	// `lucide-solid` publica um arquivo por ícone e o Vite se RECUSA a
	// pré-empacotá-lo, porque ele vem em JSX-fonte —, e a carga desperdiçada
	// esgotou o pool de conexões do Chromium: página em branco, sem erro de
	// JavaScript nenhum.
	//
	// No mux e não no `spaHandler` porque em desenvolvimento o `spaHandler` nem
	// é montado (sem `STATIC_DIR`), e aí o desvio só existiria em produção — o
	// e2e mediria uma coisa e o jogador veria outra.
	//
	// `"/{$}"` casa a raiz EXATA; `"/"` casaria o app inteiro. A rota `/` da SPA
	// continua existindo para quem já está DENTRO dela e chama
	// `navigate({ to: '/' })` — as duas cobrem casos que a outra não alcança.
	mux.HandleFunc("/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/piloto/", http.StatusFound)
	})
	if cfg.StaticDir == "" {
		mux.Handle("/", srv.Router())
		return mux
	}
	mux.Handle("/api/", http.StripPrefix("/api", srv.Router()))
	mux.Handle("/", spaHandler(cfg.StaticDir))
	log.Printf("serving built frontend from %s", cfg.StaticDir)
	return mux
}

// announce prints where to point a browser. When this process serves the SPA it
// also prints the LAN addresses, because the players open the app from their own
// machines and the host would otherwise have to go read `ip addr` (ALE-119).
func announce(cfg plataforma.Config) {
	log.Printf("t20 %s server listening on :%s (%s, db=%s)", cfg.AppEnv, cfg.Port, cfg.Scheme(), cfg.DatabasePath)
	if cfg.TLSEnabled() && !cfg.CookieSecure {
		log.Print("  aviso: há TLS e COOKIE_SECURE=false — o cookie de sessão viaja sem a marca Secure")
	}
	if cfg.StaticDir == "" {
		return
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

// spaHandler serves the built SPA from dir: an existing file (JS/CSS/wasm assets) is
// served directly, anything else falls back to index.html so client-side (TanStack)
// routes resolve on a hard refresh. Mirrors what the Vite dev server does implicitly.
func spaHandler(dir string) http.Handler {
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rooted Clean so "../" can't escape dir; serve the file when it exists.
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			serveMaybeCompressed(w, r, p)
			return
		}
		serveMaybeCompressed(w, r, index)
	})
}

// serveMaybeCompressed entrega o irmão `.br`/`.gz` pré-comprimido quando o
// navegador aceita e ele existe, e o arquivo cru quando não (ALE-153).
//
// O `net/http` não comprime nada sozinho, e ninguém percebeu: o navegador PEDE
// `gzip, br` e recebia os 3,7 MB crus do `t20.wasm` — 4,9 MB de carga fria que
// comprimidos são 1,1 MB. Comprimir na REQUISIÇÃO seria pagar CPU da máquina do
// mestre a cada jogador que entra; o build pré-comprime uma vez
// (`frontend/scripts/precompress-dist.sh`) e aqui só se escolhe a variante.
//
// Ausência de irmão é caminho normal, não erro: um build sem o passo de
// compressão continua servindo o app, só mais pesado.
func serveMaybeCompressed(w http.ResponseWriter, r *http.Request, file string) {
	// O `Content-Type` sai do nome ORIGINAL: o `ServeFile` o adivinharia pela
	// extensão do irmão, e `application/octet-stream` faz o
	// `WebAssembly.instantiateStreaming` recusar o wasm.
	if ctype := contentTypeFor(file); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	// Sem `Vary`, um proxy no meio serviria a resposta comprimida para quem não
	// aceita — e o contrário, que é pior.
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("Cache-Control", cacheControlFor(file))

	for _, variant := range []struct{ encoding, ext string }{{"br", ".br"}, {"gzip", ".gz"}} {
		if !plataforma.AcceptsEncoding(r.Header.Get("Accept-Encoding"), variant.encoding) {
			continue
		}
		if info, err := os.Stat(file + variant.ext); err != nil || info.IsDir() {
			continue
		}
		w.Header().Set("Content-Encoding", variant.encoding)
		http.ServeFile(w, r, file+variant.ext)
		return
	}
	http.ServeFile(w, r, file)
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
