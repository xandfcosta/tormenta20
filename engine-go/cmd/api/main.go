// Command api is the Go port of the NestJS backend HTTP API. It opens + migrates
// the SQLite database and serves the domain routes the frontend consumes via the
// Vite /api proxy. Runs alongside the Nest server until the big-bang cutover.
package main

import (
	"log"
	"net/http"

	"t20engine/api"
	"t20engine/db"
)

func main() {
	cfg := api.LoadConfig()

	database, err := db.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer func() { _ = database.Close() }()

	srv := api.NewServer(cfg, database)
	log.Printf("t20 API listening on :%s (db=%s)", cfg.Port, cfg.DatabasePath)
	if err := http.ListenAndServe(":"+cfg.Port, srv.Router()); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
