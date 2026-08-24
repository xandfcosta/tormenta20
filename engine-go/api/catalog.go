package api

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"t20engine/catalog"
)

// handleCatalogIndex serves GET /catalog — the accepted resource names.
func (s *Server) handleCatalogIndex(w http.ResponseWriter, _ *http.Request) {
	plataforma.WriteJSON(w, http.StatusOK, catalog.Resources())
}

// handleCatalogResource serves GET /catalog/:resource from the embedded JSON.
// Unknown name → 404 with the accepted set.
func (s *Server) handleCatalogResource(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "resource")
	body, ok := catalog.Resource(name)
	if !ok {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf(
			"unknown catalog resource: %q; expected one of %s", name, strings.Join(catalog.Resources(), ", ")))
		return
	}
	writeCatalogJSON(w, r, name, body)
}

// handleCharacterOptions serves GET /characters/options (public creation lists).
func (s *Server) handleCharacterOptions(w http.ResponseWriter, r *http.Request) {
	body, err := catalog.Options()
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not load options")
		return
	}
	writeCatalogJSON(w, r, "characters/options", body)
}

// writeRawJSON writes pre-serialized JSON bytes verbatim (no re-encode).
func writeRawJSON(w http.ResponseWriter, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// writeCatalogJSON entrega o catálogo comprimido quando o navegador aceita
// (ALE-159).
//
// O catálogo é o único payload GRANDE e IMUTÁVEL da API: `spells` sozinho são
// 179 KB crus, ~40 KB em gzip, e o front os busca por HTTP de propósito — a
// decisão da ALE-107 de não embutir catálogo nenhum no bundle —, então esses
// bytes entram em TODA carga fria, logo depois do wasm.
//
// Comprimido UMA vez e guardado, não a cada requisição: o conteúdo vem de
// `go:embed` e não muda enquanto o binário for o mesmo. É a mesma escolha da
// ALE-153 para a SPA, pelo mesmo motivo — a máquina que serve a mesa é a do
// mestre, e ela não deve gastar CPU repetindo trabalho idêntico por jogador.
func writeCatalogJSON(w http.ResponseWriter, r *http.Request, name string, body []byte) {
	// `Vary` sempre, mesmo servindo cru: sem ele um proxy no meio guardaria a
	// resposta comprimida e a devolveria a quem não aceita.
	w.Header().Set("Vary", "Accept-Encoding")
	if !plataforma.AcceptsEncoding(r.Header.Get("Accept-Encoding"), "gzip") {
		writeRawJSON(w, body)
		return
	}
	packed, err := gzippedCatalog(name, body)
	if err != nil {
		// Falha ao comprimir não pode custar a resposta: o cru serve.
		log.Printf("catálogo %q: gzip falhou (%v); servindo cru", name, err)
		writeRawJSON(w, body)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(packed)
}

// gzipCache guarda o corpo comprimido por nome de recurso. Preguiçoso em vez de
// no boot: quem nunca pede um catálogo não paga por ele, e o primeiro pedido
// custa alguns milissegundos uma única vez.
var gzipCache sync.Map // map[string][]byte

func gzippedCatalog(name string, body []byte) ([]byte, error) {
	if cached, ok := gzipCache.Load(name); ok {
		return cached.([]byte), nil
	}
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(body); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	packed := buf.Bytes()
	gzipCache.Store(name, packed)
	return packed, nil
}
