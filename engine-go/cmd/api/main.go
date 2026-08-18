// Command api is the app's HTTP server. It opens + migrates the SQLite database
// and serves the domain routes the frontend consumes — via the Vite proxy in
// dev, and directly alongside the built SPA in production (STATIC_DIR).
//
// The environment comes from `.env.<APP_ENV>` next to the package (ALE-119):
// `air` boots it as development, `pnpm start` as production.
package main

import (
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	announce(cfg) // last, so the address to open is the final line on the screen
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
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

	for _, variant := range []struct{ encoding, ext string }{{"br", ".br"}, {"gzip", ".gz"}} {
		if !acceptsEncoding(r.Header.Get("Accept-Encoding"), variant.encoding) {
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

// acceptsEncoding responde se o cabeçalho lista a codificação. Comparação por
// token e não por `strings.Contains`: "gzip;q=0" é uma RECUSA explícita, e
// aceitá-la mandaria conteúdo comprimido para quem disse que não quer.
func acceptsEncoding(header, encoding string) bool {
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(fields[0]), encoding) {
			continue
		}
		for _, param := range fields[1:] {
			if strings.EqualFold(strings.TrimSpace(param), "q=0") {
				return false
			}
		}
		return true
	}
	return false
}
