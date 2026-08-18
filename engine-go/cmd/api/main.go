// Command api is the app's HTTP server. It opens + migrates the SQLite database
// and serves the domain routes the frontend consumes — via the Vite proxy in
// dev, and directly alongside the built SPA in production (STATIC_DIR).
//
// The environment comes from `.env.<APP_ENV>` next to the package (ALE-119):
// `air` boots it as development, `pnpm start` as production.
package main

import (
	"context"
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
)

func main() {
	cfg, err := api.LoadConfig()
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
func httpServerFor(cfg api.Config, mux http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// serve sobe o HTTP e espera o sinal para desligar com ordem.
//
// Os timeouts são escolhidos, não copiados de um exemplo (ALE-157):
//
//   - `ReadHeaderTimeout` existe porque sem ele uma conexão que abre e nunca
//     manda o cabeçalho segura uma goroutine para sempre (slowloris);
//   - `IdleTimeout` recolhe conexões ociosas do keep-alive;
//   - `WriteTimeout` fica de FORA de propósito. Ele mataria o socket.io, que é
//     conexão longa por natureza, e o download do wasm de 780 KB numa rede
//     ruim. É o timeout que parece obrigatório e é justamente o errado aqui.
func serve(ctx context.Context, cfg api.Config, mux http.Handler) error {
	server := httpServerFor(cfg, mux)
	falhou := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

// buildMux wires the realtime gateway plus either the production single binary
// (SPA + /api/* + socket on one port) or the dev shape, where Vite serves the
// front and strips /api before proxying to us.
func buildMux(cfg api.Config, srv *api.Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/socket.io/", srv.SocketHandler())
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
func announce(cfg api.Config) {
	log.Printf("t20 %s server listening on :%s (db=%s)", cfg.AppEnv, cfg.Port, cfg.DatabasePath)
	if cfg.StaticDir == "" {
		return
	}
	for _, url := range lanURLs(cfg.Port) {
		log.Printf("  players can open %s", url)
	}
}

// lanURLs lists this host's non-loopback IPv4 addresses as URLs. The server
// binds every interface, so these already work — they are just not discoverable
// from the log line above.
func lanURLs(port string) []string {
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
		urls = append(urls, fmt.Sprintf("http://%s:%s", ipNet.IP, port))
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
	if ctype := mime.TypeByExtension(filepath.Ext(file)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	// Sem `Vary`, um proxy no meio serviria a resposta comprimida para quem não
	// aceita — e o contrário, que é pior.
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Set("Cache-Control", cacheControlFor(file))

	for _, variant := range []struct{ encoding, ext string }{{"br", ".br"}, {"gzip", ".gz"}} {
		if !api.AcceptsEncoding(r.Header.Get("Accept-Encoding"), variant.encoding) {
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
