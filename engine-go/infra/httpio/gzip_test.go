package httpio

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Os guardas da COMPRESSÃO das cenas renderizadas.
//
// O que eles prendem não é "comprime" — é o conjunto de casos em que comprimir
// está ERRADO, e um deles não deixa erro para trás.

// scene é um handler que responde como as cenas respondem.
//
// Ela declara o `Content-Length`, como faz todo handler que serve conteúdo de
// tamanho conhecido. Sem essa linha não há o que apagar, e a asserção sobre o
// cabeçalho passa verde com o `Header().Del("Content-Length")` sabotado.
func scene(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = io.WriteString(w, body)
	})
}

func requestWith(t *testing.T, h http.Handler, accept string) *http.Response {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/personagens/1", nil)
	if accept != "" {
		req.Header.Set("Accept-Encoding", accept)
	}
	rec := httptest.NewRecorder()
	Gzip(h).ServeHTTP(rec, req)
	return rec.Result()
}

// A CENA VIAJA COMPRIMIDA, e o corpo continua sendo o mesmo depois de inflado.
func TestTheRenderedSceneTravelsCompressed(t *testing.T) {
	body := strings.Repeat("<div class=\"caixa\">Defesa 22</div>", 400)
	resp := requestWith(t, scene(body), "gzip")

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q, quer gzip: a cena saiu crua", got)
	}
	if !strings.Contains(resp.Header.Get("Vary"), "Accept-Encoding") {
		t.Error("sem `Vary: Accept-Encoding`: um cache serviria bytes comprimidos " +
			"para quem não aceita gzip")
	}
	if resp.Header.Get("Content-Length") != "" {
		t.Error("o `Content-Length` do texto CRU sobreviveu à compressão: o cliente " +
			"espera bytes que não vêm ou corta o corpo no meio")
	}

	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("o corpo não é gzip válido: %v", err)
	}
	inflated, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("inflar: %v", err)
	}
	if string(inflated) != body {
		t.Error("o corpo inflado não é igual ao original")
	}
}

// QUEM NÃO ACEITA GZIP RECEBE O TEXTO CRU, e `q=0` é uma RECUSA.
//
// O `q=0` é o caso que ninguém lembra de tratar, e um `strings.Contains` o leria
// como aceitação.
func TestWhoeverDoesNotAcceptGzipGetsItRaw(t *testing.T) {
	// GRANDE de propósito: com um corpo curto este caso passaria pelo corte de
	// TAMANHO em vez de pela negociação, e continuaria verde com a leitura do
	// `Accept-Encoding` quebrada. Um teste que pode passar por dois motivos não
	// prende nenhum dos dois.
	body := strings.Repeat("<p>o texto cru</p>", 200)
	for _, accept := range []string{"", "identity", "gzip;q=0", "br"} {
		t.Run(fmt.Sprintf("accept=%q", accept), func(t *testing.T) {
			resp := requestWith(t, scene(body), accept)
			if got := resp.Header.Get("Content-Encoding"); got != "" {
				t.Errorf("com Accept-Encoding %q a resposta saiu %q", accept, got)
			}
			read, _ := io.ReadAll(resp.Body)
			if string(read) != body {
				t.Errorf("o corpo cru não sobreviveu: %d bytes", len(read))
			}
		})
	}
}

// O QUE JÁ VEM COMPRIMIDO DO BUILD NÃO É RECOMPRIMIDO.
//
// Um estático que sai do build com irmão `.br`/`.gz` é servido com
// `Content-Encoding` próprio; reembrulhá-lo gastaria CPU para produzir bytes
// MAIORES, e o navegador desinflaria uma camada só.
//
// O tipo é `text/html`, e ISSO É O CASO: um `application/wasm` não é comprimível
// pelo TIPO, então um caso escrito com ele passaria pelo motivo errado e ficaria
// VERDE com a regra do `Content-Encoding` sabotada.
func TestWhatArrivesCompressedPassesThroughIntact(t *testing.T) {
	alreadyGzipped := "\x1f\x8b conteudo ja em gzip"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = io.WriteString(w, alreadyGzipped)
	})
	resp := requestWith(t, handler, "gzip")

	body, _ := io.ReadAll(resp.Body)
	if string(body) != alreadyGzipped {
		t.Errorf("o corpo mudou: o middleware embrulhou de novo o que já vinha "+
			"comprimido, e o navegador desinflaria uma camada só — corpo = %q", body)
	}
}

// O FLUXO AO VIVO ATRAVESSA O GZIP, e este é o guarda que justifica o arquivo.
//
// A Mesa é um SSE de conexão longa, e a resposta de todo comando do Datastar
// também é `text/event-stream` — ela usa o envelope de SSE para mandar UM
// remendo. Então "não comprimir SSE" pularia o que se quer comprimir, e
// comprimir SEM repassar o `Flush` prende o quadro no buffer interno do
// `gzip.Writer`: ele fica esperando encher, o navegador fica esperando o quadro,
// e a mesa simplesmente PARA de atualizar. Nada falha, nada loga, e o sintoma —
// "o tempo real quebrou" — não aponta para um middleware de compressão.
//
// O caso escreve UM quadro, esvazia, e exige que ele chegue inflado ANTES de o
// handler retornar. Sem o `Flush` atravessando, ele estoura no tempo.
func TestTheLiveStreamCrossesTheGzip(t *testing.T) {
	frameWritten := make(chan struct{})
	holds := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: patch\ndata: <p>a mesa mexeu</p>\n\n")
		w.(http.Flusher).Flush()
		close(frameWritten)
		<-holds // a conexão continua ABERTA, como um SSE de verdade
	})

	server := httptest.NewServer(Gzip(handler))
	defer server.Close()
	defer close(holds)

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	// `DisableCompression`: sem isto o próprio cliente do Go põe o
	// `Accept-Encoding` e infla sozinho, e o caso mediria o transporte em vez do
	// middleware.
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("abrir o fluxo: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q: o fluxo não foi comprimido, e o caso não "+
			"mediria a travessia do Flush", got)
	}
	<-frameWritten

	arrived := make(chan string, 1)
	go func() {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			arrived <- "ERRO ao abrir o gzip: " + err.Error()
			return
		}
		buf := make([]byte, 256)
		n, _ := zr.Read(buf)
		arrived <- string(buf[:n])
	}()

	select {
	case text := <-arrived:
		if !strings.Contains(text, "a mesa mexeu") {
			t.Errorf("o quadro chegou como %q", text)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("o quadro NÃO chegou com a conexão aberta: o `Flush` não atravessa o " +
			"gzip, e o tempo real da Mesa para sem erro nenhum")
	}
}

// O FLUSH ANTES DO PRIMEIRO WRITE COMPROMETE OS CABEÇALHOS.
//
// O `datastar-go` monta o fluxo assim, nesta ordem: escreve o `Content-Type`,
// chama `rc.Flush()` para MANDAR OS CABEÇALHOS, e só então escreve o primeiro
// remendo. Um envelope que decide comprimir apenas no `Write` chega tarde: os
// cabeçalhos já foram, sem `Content-Encoding`, e o corpo sai comprimido mesmo
// assim. O cliente recebe bytes de gzip rotulados como texto puro.
//
// O sintoma não aponta para lugar nenhum: nenhuma requisição falha, nenhum
// status muda, e o que se vê é que os remendos do Datastar simplesmente PARAM de
// ser aplicados — busca que não filtra, seta que não anda, diálogo que não abre.
// Um guarda que escreva o cabeçalho ANTES de esvaziar fica verde sobre isto.
func TestAFlushBeforeTheWriteAlreadyDecidesTheEnvelope(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A ORDEM É A DO DATASTAR, e ela é o caso inteiro.
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, "event: patch\ndata: <p>oi</p>\n\n")
		w.(http.Flusher).Flush()
	})

	req := httptest.NewRequest(http.MethodGet, "/personagens/1", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	Gzip(handler).ServeHTTP(rec, req)
	resp := rec.Result()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q: o corpo saiu comprimido e o cabeçalho não "+
			"disse — o cliente lê bytes de gzip como se fossem texto", got)
	}
	zr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("o corpo não é gzip válido: %v", err)
	}
	body, _ := io.ReadAll(zr)
	if !strings.Contains(string(body), "<p>oi</p>") {
		t.Errorf("o remendo não sobreviveu: %q", body)
	}
}

// A REGRA de quem comprime, exercitada direto.
func TestOnlyTheTypesWorthItAreCompressed(t *testing.T) {
	cases := []struct {
		kind string
		want bool
	}{
		{"text/html; charset=utf-8", true},
		{"text/event-stream", true},
		{"text/css", true},
		{"application/json", true},
		{"application/javascript", true},
		{"image/svg+xml", true},
		{"image/png", false},
		{"application/wasm", false},
		{"", false},
	}
	for _, tc := range cases {
		h := http.Header{}
		if tc.kind != "" {
			h.Set("Content-Type", tc.kind)
		}
		if got := willCompress(h, http.StatusOK); got != tc.want {
			t.Errorf("willCompress(%q) = %v, esperado %v", tc.kind, got, tc.want)
		}
	}
	// RESPOSTA PEQUENA sai crua: comprimir 19 bytes produziu 43 no contêiner.
	// O corte usa o `Content-Length` DECLARADO, e por isso um fluxo — que nunca
	// declara tamanho — continua passando.
	small := http.Header{}
	small.Set("Content-Type", "text/html")
	small.Set("Content-Length", "19")
	if willCompress(small, http.StatusOK) {
		t.Error("uma resposta de 19 bytes foi comprimida: o envelope do gzip a deixa MAIOR")
	}
	stream := http.Header{}
	stream.Set("Content-Type", "text/event-stream")
	if !willCompress(stream, http.StatusOK) {
		t.Error("o fluxo não passou pelo corte de tamanho: ele não declara " +
			"`Content-Length`, e tratá-lo como pequeno mataria a compressão do tempo real")
	}

	// SEM CORPO não leva envelope: um gzip vazio sobre um 304 é lixo que alguns
	// clientes recusam.
	noBody := http.Header{}
	noBody.Set("Content-Type", "text/html")
	for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
		if willCompress(noBody, status) {
			t.Errorf("o status %d não tem corpo e mesmo assim levou envelope gzip", status)
		}
	}
}
