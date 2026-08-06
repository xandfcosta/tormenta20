// Command server exposes the Go compute engine over stdlib net/http, mirroring
// the Node counterpart (bench/server-node/server.mjs) so the Go-vs-Node
// benchmark compares runtime + engine, not framework vs framework.
//
//	POST /sheet   body = CharacterInput JSON → 200 ComputedSheet JSON
//	GET  /health  → 200 "ok"
//
// Run: go run ./cmd/server [port]   (default 3002)
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"t20engine/engine"
)

func main() {
	port := "3002"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/sheet", handleSheet)

	addr := ":" + port
	log.Printf("go engine server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("content-type", "text/plain")
	_, _ = io.WriteString(w, "ok")
}

func handleSheet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, err)
		return
	}
	var input engine.CharacterInput
	if err := json.Unmarshal(body, &input); err != nil {
		writeError(w, err)
		return
	}
	sheet := engine.ComputeCharacterSheet(&input)
	out, err := json.Marshal(sheet)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("content-type", "application/json")
	_, _ = w.Write(out)
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
