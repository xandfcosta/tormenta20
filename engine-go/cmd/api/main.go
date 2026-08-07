// Command api is the Go port of the NestJS backend HTTP API. It opens + migrates
// the SQLite database and serves the domain routes the frontend consumes via the
// Vite /api proxy. Runs alongside the Nest server until the big-bang cutover.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"t20engine/api"
	"t20engine/db"
	"t20engine/engine"
)

func main() {
	cfg := api.LoadConfig()

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer func() { _ = database.Close() }()

	// Best-effort: prime the rules catalogs for the mutation validators. Auth +
	// read + vitals work without them; item/creation writes need them.
	var catalogs *engine.Catalogs
	if raw, err := os.ReadFile(cfg.CatalogPath); err != nil {
		log.Printf("catalogs: %v — mutation validators disabled", err)
	} else if catalogs, err = engine.PrimeEngineCatalogs(raw); err != nil {
		log.Printf("catalogs: prime failed: %v", err)
		catalogs = nil
	} else {
		log.Printf("catalogs primed from %s", cfg.CatalogPath)
	}

	srv := api.NewServer(cfg, database, catalogs)

	// The socket.io realtime gateway (B.6) lives at /socket.io/ in both modes.
	mux := http.NewServeMux()
	mux.Handle("/socket.io/", srv.SocketHandler())
	if cfg.StaticDir != "" {
		// Production single binary: serve the built front here + route /api/* to the
		// domain (no Vite to strip the prefix), with an SPA fallback for client routes.
		mux.Handle("/api/", http.StripPrefix("/api", srv.Router()))
		mux.Handle("/", spaHandler(cfg.StaticDir))
		log.Printf("serving built frontend from %s", cfg.StaticDir)
	} else {
		// Dev: Vite serves the front and strips /api before proxying to us.
		mux.Handle("/", srv.Router())
	}

	log.Printf("t20 API listening on :%s (db=%s)", cfg.Port, cfg.DatabasePath)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// spaHandler serves the built SPA from dir: an existing file (JS/CSS/wasm assets) is
// served directly, anything else falls back to index.html so client-side (TanStack)
// routes resolve on a hard refresh. Mirrors what the Vite dev server does implicitly.
func spaHandler(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rooted Clean so "../" can't escape dir; serve the file when it exists.
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}
