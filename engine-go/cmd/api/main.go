// Command api is the Go port of the NestJS backend HTTP API. It opens + migrates
// the SQLite database and serves the domain routes the frontend consumes via the
// Vite /api proxy. Runs alongside the Nest server until the big-bang cutover.
package main

import (
	"log"
	"net/http"
	"os"

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

	// Mount the socket.io realtime gateway (B.6) at /socket.io/ alongside the HTTP
	// API; the Vite proxy forwards both to this server at cutover.
	mux := http.NewServeMux()
	mux.Handle("/socket.io/", srv.SocketHandler())
	mux.Handle("/", srv.Router())

	log.Printf("t20 API listening on :%s (db=%s)", cfg.Port, cfg.DatabasePath)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
