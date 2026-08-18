package api

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// O catálogo é o único payload GRANDE e IMUTÁVEL da API (ALE-159).
//
// O front o busca por HTTP de propósito — a decisão da ALE-107 de não embutir
// catálogo no bundle —, então esses bytes entram em toda carga fria, logo
// depois do wasm. `spells` sozinho são ~179 KB crus.

func pedirCatalogo(t *testing.T, s *Server, recurso, accept string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/catalog/"+recurso, nil)
	if accept != "" {
		req.Header.Set("Accept-Encoding", accept)
	}
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	return rec.Result()
}

func TestCatalogTravelsCompressed(t *testing.T) {
	s := newTestServer(t)

	comprimido := pedirCatalogo(t, s, "spells", "gzip")
	cru := pedirCatalogo(t, s, "spells", "")

	if got := comprimido.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding %q para quem aceita gzip", got)
	}
	// Sem `Vary`, um proxy no meio guardaria isto e devolveria a quem não aceita.
	if got := comprimido.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary %q, esperava Accept-Encoding", got)
	}
	if got := cru.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding %q para quem não pediu compressão", got)
	}

	bytesComprimidos, _ := io.ReadAll(comprimido.Body)
	bytesCrus, _ := io.ReadAll(cru.Body)
	if len(bytesComprimidos) >= len(bytesCrus) {
		t.Fatalf("comprimido (%d) não ficou menor que cru (%d)", len(bytesComprimidos), len(bytesCrus))
	}
	t.Logf("spells: %d bytes crus → %d comprimidos (%.0f%% a menos)",
		len(bytesCrus), len(bytesComprimidos),
		100*(1-float64(len(bytesComprimidos))/float64(len(bytesCrus))))

	// E o que chega descomprimido é IGUAL ao original: compressão que corrompe
	// o catálogo derrubaria a ficha inteira, e o front não teria como acusar.
	leitor, err := gzip.NewReader(bytes.NewReader(bytesComprimidos))
	if err != nil {
		t.Fatalf("o corpo não é gzip válido: %v", err)
	}
	descomprimido, err := io.ReadAll(leitor)
	if err != nil {
		t.Fatalf("descomprimir: %v", err)
	}
	if !bytes.Equal(descomprimido, bytesCrus) {
		t.Error("o catálogo descomprimido difere do cru")
	}
}

// `gzip;q=0` é RECUSA explícita — a mesma leitura por token que a SPA usa, e
// agora numa implementação só para os dois (ALE-159).
func TestCatalogRespectsQualityZero(t *testing.T) {
	s := newTestServer(t)

	res := pedirCatalogo(t, s, "spells", "gzip;q=0")

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding %q apesar de gzip;q=0", got)
	}
}

// A segunda requisição usa o corpo já comprimido: o conteúdo vem de `go:embed`
// e não muda enquanto o binário for o mesmo.
func TestCatalogCompressesOnlyOnce(t *testing.T) {
	s := newTestServer(t)
	gzipCache.Delete("spells")

	primeira, _ := io.ReadAll(pedirCatalogo(t, s, "spells", "gzip").Body)
	if _, guardado := gzipCache.Load("spells"); !guardado {
		t.Fatal("o corpo comprimido não ficou guardado — cada jogador pagaria a compressão de novo")
	}
	segunda, _ := io.ReadAll(pedirCatalogo(t, s, "spells", "gzip").Body)

	if !bytes.Equal(primeira, segunda) {
		t.Error("a segunda resposta difere da primeira")
	}
}

// Recurso desconhecido continua 404 com a lista do que existe — o caminho de
// erro não pode virar um gzip de mensagem de erro.
func TestUnknownCatalogStillExplains(t *testing.T) {
	s := newTestServer(t)

	res := pedirCatalogo(t, s, "inventado", "gzip")

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("recurso inventado respondeu %d", res.StatusCode)
	}
	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("a resposta de erro saiu comprimida (%q)", got)
	}
}
