package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"t20engine/plataforma"
	"testing"
)

// O que a SPA entrega no FIO (ALE-153).
//
// O defeito que originou estes testes: o `net/http` não comprime nada sozinho, e
// o servidor mandava os 3,7 MB crus do `t20.wasm` para um navegador que pedia
// `gzip, br` — 4,9 MB de carga fria que comprimidos são 1,1 MB. Ninguém percebeu
// porque o app funcionava perfeitamente; só era lento.

// dist monta um build de mentira com o irmão comprimido de um dos arquivos.
func dist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	escrever := func(nome, conteudo string) {
		if err := os.WriteFile(filepath.Join(dir, nome), []byte(conteudo), 0o644); err != nil {
			t.Fatalf("escrever %s: %v", nome, err)
		}
	}
	escrever("index.html", "<!doctype html><title>t20</title>")
	escrever("t20.wasm", "os bytes crus do motor")
	escrever("t20.wasm.br", "brotli")
	escrever("t20.wasm.gz", "gzip")
	// Sem irmão comprimido: build que não rodou o passo de compressão.
	escrever("app.css", "body{}")
	escrever("manifest.webmanifest", `{"name":"Tormenta 20"}`)
	return dir
}

func pedir(t *testing.T, dir, caminho, accept string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, caminho, nil)
	if accept != "" {
		req.Header.Set("Accept-Encoding", accept)
	}
	rec := httptest.NewRecorder()
	spaHandler(dir).ServeHTTP(rec, req)
	return rec.Result()
}

func TestBrotliWinsWhenTheBrowserTakesIt(t *testing.T) {
	res := pedir(t, dist(t), "/t20.wasm", "gzip, deflate, br")

	if got := res.Header.Get("Content-Encoding"); got != "br" {
		t.Errorf("Content-Encoding %q, esperava br — o navegador aceita e o irmão existe", got)
	}
	// O tipo sai do nome ORIGINAL: adivinhado pela extensão do irmão, o wasm
	// viraria octet-stream e o `instantiateStreaming` recusaria.
	if got := res.Header.Get("Content-Type"); got != "application/wasm" {
		t.Errorf("Content-Type %q, esperava application/wasm", got)
	}
	// Sem `Vary`, um proxy no meio serviria isto para quem não aceita.
	if got := res.Header.Get("Vary"); got != "Accept-Encoding" {
		t.Errorf("Vary %q, esperava Accept-Encoding", got)
	}
}

func TestGzipWhenBrotliIsNotOffered(t *testing.T) {
	res := pedir(t, dist(t), "/t20.wasm", "gzip, deflate")

	if got := res.Header.Get("Content-Encoding"); got != "gzip" {
		t.Errorf("Content-Encoding %q, esperava gzip", got)
	}
}

// Quem não aceita nada recebe o arquivo cru — e íntegro.
func TestRawForAClientThatAcceptsNothing(t *testing.T) {
	res := pedir(t, dist(t), "/t20.wasm", "")

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding %q para quem não pediu compressão", got)
	}
}

// `gzip;q=0` é uma RECUSA explícita, e um `strings.Contains` a leria como
// aceitação — mandando conteúdo comprimido para quem disse que não quer.
func TestQualityZeroIsARefusal(t *testing.T) {
	res := pedir(t, dist(t), "/t20.wasm", "gzip;q=0")

	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding %q apesar de gzip;q=0", got)
	}
}

// Build sem o passo de compressão continua servindo o app: ausência de irmão é
// caminho normal, não erro.
func TestFileWithoutASiblingIsStillServed(t *testing.T) {
	res := pedir(t, dist(t), "/app.css", "gzip, br")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d para arquivo sem irmão comprimido", res.StatusCode)
	}
	if got := res.Header.Get("Content-Encoding"); got != "" {
		t.Errorf("Content-Encoding %q para arquivo que não tem irmão", got)
	}
}

// A rota do cliente cai no index — e o index também viaja comprimido, que é o
// primeiro byte que o jogador espera.
func TestClientRouteFallsBackToTheIndex(t *testing.T) {
	res := pedir(t, dist(t), "/campaigns/1/sessions/4", "br")

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d numa rota do cliente", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got == "" {
		t.Error("o index saiu sem Content-Type")
	}
}

// O manifest sai com tipo de JSON (ALE-118).
//
// A tabela do `mime` do Go NÃO conhece `.webmanifest`, então sem a tabela da
// casa o arquivo sai adivinhado pelo CONTEÚDO: `text/plain`. O Chromium engole
// isso — foi medido, ele parseia o manifest mesmo assim —, e é justamente por
// isso que só um teste segura este byte: nenhum navegador da nossa mesa
// reclamaria, e a especificação pede um tipo de JSON.
func TestTheManifestGoesOutAsJSON(t *testing.T) {
	res := pedir(t, dist(t), "/manifest.webmanifest", "")

	if got := res.Header.Get("Content-Type"); got != "application/manifest+json" {
		t.Errorf("Content-Type %q, esperava application/manifest+json — a tabela do mime do Go não conhece .webmanifest e adivinha text/plain", got)
	}
}

// Quanto tempo o navegador pode guardar cada coisa (ALE-157).
func TestHashedAssetsAreImmutableAndTheRestRevalidates(t *testing.T) {
	dir := dist(t)
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatalf("criar assets: %v", err)
	}
	// O Vite carimba o hash NO NOME: este arquivo nunca muda de conteúdo.
	if err := os.WriteFile(filepath.Join(dir, "assets", "admin-BvY0grFM.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatalf("escrever asset: %v", err)
	}

	asset := pedir(t, dir, "/assets/admin-BvY0grFM.js", "")
	if got := asset.Header.Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("asset hasheado com Cache-Control %q — o hash no nome é o que autoriza guardar para sempre", got)
	}

	// O index aponta para os assets novos: guardá-lo congelaria o app inteiro
	// numa versão antiga, com os arquivos novos já publicados ao lado.
	if got := pedir(t, dir, "/", "").Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("index com Cache-Control %q, esperava no-cache", got)
	}
	// O wasm NÃO é hasheado — revalidar custa um 304 vazio.
	if got := pedir(t, dir, "/t20.wasm", "").Header.Get("Cache-Control"); got != "no-cache" {
		t.Errorf("wasm com Cache-Control %q, esperava no-cache", got)
	}
}

// Os timeouts são escolhidos: o que falta é tão importante quanto o que está.
func TestServerTimeoutsProtectWithoutKillingLongResponses(t *testing.T) {
	server := httpServerFor(plataforma.Config{Port: "0"}, http.NewServeMux())

	if server.ReadHeaderTimeout == 0 {
		t.Error("sem ReadHeaderTimeout: uma conexão que nunca manda cabeçalho segura uma goroutine para sempre")
	}
	if server.IdleTimeout == 0 {
		t.Error("sem IdleTimeout: conexão ociosa de keep-alive nunca é recolhida")
	}
	// O que NÃO pode existir: `WriteTimeout` mataria o fluxo SSE, que é conexão
	// longa por natureza, e o download do wasm de 780 KB numa rede ruim.
	if server.WriteTimeout != 0 {
		t.Errorf("WriteTimeout=%s derruba o fluxo SSE ao vivo e o download do wasm", server.WriteTimeout)
	}
}

// HTTPS na LAN: o que o processo faz quando o par de certificados existe
// (ALE-118). O TLS termina AQUI, no mesmo processo que serve a SPA, a API e o
// fluxo de eventos — é a decisão registrada no `engine-go/CLAUDE.md`, e um proxy na
// frente a contrariaria.

// O log É o endereço que o mestre lê e repassa para a mesa. Com TLS ligado e
// `http://` impresso, os quatro telefones batem num 400 do próprio net/http e o
// sintoma parece defeito do app.
func TestTheAnnouncedURLFollowsTheScheme(t *testing.T) {
	semTLS := lanURLs(plataforma.Config{Port: "3001"})
	comTLS := lanURLs(plataforma.Config{Port: "3001", TLSCertFile: "/tmp/c.pem", TLSKeyFile: "/tmp/k.pem"})

	if len(semTLS) == 0 || len(comTLS) == 0 {
		t.Skip("máquina sem endereço IPv4 não-loopback: não há URL de LAN para anunciar")
	}
	for _, url := range semTLS {
		if !strings.HasPrefix(url, "http://") {
			t.Errorf("sem TLS o endereço saiu %q", url)
		}
	}
	for _, url := range comTLS {
		if !strings.HasPrefix(url, "https://") {
			t.Errorf("com TLS o endereço saiu %q — a mesa inteira digitaria o esquema errado", url)
		}
	}
}

// O piso do TLS é dito com todas as letras: o padrão do Go pode mudar de versão,
// e um aparelho antigo negociando TLS 1.0 seria uma queda silenciosa.
func TestTheTLSFloorIsDeclared(t *testing.T) {
	server := httpServerFor(plataforma.Config{Port: "0"}, http.NewServeMux())

	if server.TLSConfig == nil {
		t.Fatal("sem TLSConfig: o piso do TLS passa a ser o do Go da vez, e ele muda de versão")
	}
	if got := server.TLSConfig.MinVersion; got != tls.VersionTLS12 {
		t.Errorf("MinVersion %#04x, esperava TLS 1.2 (%#04x)", got, tls.VersionTLS12)
	}
}
