package plataforma

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

// Os guardas da COMPRESSÃO das cenas renderizadas (ALE-273).
//
// O que eles prendem não é "comprime" — é o conjunto de casos em que comprimir
// está ERRADO, e um deles não deixa erro para trás.

// aCena é um handler que responde como as cenas do piloto respondem.
//
// Ela declara o `Content-Length`, como faz todo handler que serve conteúdo de
// tamanho conhecido. Sem essa linha não há o que apagar, e a asserção sobre o
// cabeçalho passaria sobre o vazio — provado sabotando: a primeira versão deste
// arquivo ficou VERDE com o `Header().Del("Content-Length")` apagado.
func aCena(corpo string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Length", strconv.Itoa(len(corpo)))
		_, _ = io.WriteString(w, corpo)
	})
}

func pedeCom(t *testing.T, h http.Handler, accept string) *http.Response {
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
func TestACenaRenderizadaViajaComprimida(t *testing.T) {
	corpo := strings.Repeat("<div class=\"caixa\">Defesa 22</div>", 400)
	resp := pedeCom(t, aCena(corpo), "gzip")

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
	inflado, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("inflar: %v", err)
	}
	if string(inflado) != corpo {
		t.Error("o corpo inflado não é igual ao original")
	}
}

// QUEM NÃO ACEITA GZIP RECEBE O TEXTO CRU, e `q=0` é uma RECUSA.
//
// O `q=0` é o caso que ninguém lembra de tratar, e um `strings.Contains` o leria
// como aceitação — a mesma armadilha que a ALE-159 registrou do outro lado.
func TestQuemNaoAceitaGzipRecebeCru(t *testing.T) {
	// GRANDE de propósito: com um corpo curto este caso passaria pelo corte de
	// TAMANHO em vez de pela negociação, e continuaria verde no dia em que a
	// leitura do `Accept-Encoding` quebrasse. Um teste que pode passar por dois
	// motivos não prende nenhum dos dois.
	corpo := strings.Repeat("<p>o texto cru</p>", 200)
	for _, accept := range []string{"", "identity", "gzip;q=0", "br"} {
		t.Run(fmt.Sprintf("accept=%q", accept), func(t *testing.T) {
			resp := pedeCom(t, aCena(corpo), accept)
			if got := resp.Header.Get("Content-Encoding"); got != "" {
				t.Errorf("com Accept-Encoding %q a resposta saiu %q", accept, got)
			}
			lido, _ := io.ReadAll(resp.Body)
			if string(lido) != corpo {
				t.Errorf("o corpo cru não sobreviveu: %d bytes", len(lido))
			}
		})
	}
}

// O QUE JÁ VEM COMPRIMIDO DO BUILD NÃO É RECOMPRIMIDO.
//
// Os assets da SPA saem do build com irmão `.br`/`.gz` e o handler serve a
// variante com `Content-Encoding` próprio (ALE-153). Reembrulhar aquilo gastaria
// CPU para produzir bytes MAIORES, e o navegador desinflaria uma camada só.
//
// O tipo é `text/html`, e ISSO É O CASO: um `application/wasm` não é
// comprimível pelo TIPO, então um caso escrito com ele passaria pelo motivo
// errado — provado sabotando, a primeira versão deste teste ficou VERDE com a
// regra do `Content-Encoding` apagada. O `index.html.gz` da SPA é exatamente
// isto: comprimível pelo tipo, e já comprimido.
func TestOQueJaVemComprimidoPassaIntacto(t *testing.T) {
	jaComprimido := "\x1f\x8b conteudo ja em gzip"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = io.WriteString(w, jaComprimido)
	})
	resp := pedeCom(t, handler, "gzip")

	corpo, _ := io.ReadAll(resp.Body)
	if string(corpo) != jaComprimido {
		t.Errorf("o corpo mudou: o middleware embrulhou de novo o que já vinha "+
			"comprimido, e o navegador desinflaria uma camada só — corpo = %q", corpo)
	}
}

// O FLUXO AO VIVO ATRAVESSA O GZIP, e este é o guarda que justifica o arquivo.
//
// # O defeito que ele existe para não deixar acontecer
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
func TestOFluxoAoVivoAtravessaOGzip(t *testing.T) {
	quadroEscrito := make(chan struct{})
	segura := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: patch\ndata: <p>a mesa mexeu</p>\n\n")
		w.(http.Flusher).Flush()
		close(quadroEscrito)
		<-segura // a conexão continua ABERTA, como um SSE de verdade
	})

	servidor := httptest.NewServer(Gzip(handler))
	defer servidor.Close()
	defer close(segura)

	req, _ := http.NewRequest(http.MethodGet, servidor.URL, nil)
	req.Header.Set("Accept-Encoding", "gzip")
	// `DisableCompression`: sem isto o próprio cliente do Go põe o
	// `Accept-Encoding` e infla sozinho, e o caso mediria o transporte em vez do
	// middleware.
	cliente := &http.Client{Transport: &http.Transport{DisableCompression: true}}
	resp, err := cliente.Do(req)
	if err != nil {
		t.Fatalf("abrir o fluxo: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Encoding"); got != "gzip" {
		t.Fatalf("Content-Encoding = %q: o fluxo não foi comprimido, e o caso não "+
			"mediria a travessia do Flush", got)
	}
	<-quadroEscrito

	chegou := make(chan string, 1)
	go func() {
		zr, err := gzip.NewReader(resp.Body)
		if err != nil {
			chegou <- "ERRO ao abrir o gzip: " + err.Error()
			return
		}
		buf := make([]byte, 256)
		n, _ := zr.Read(buf)
		chegou <- string(buf[:n])
	}()

	select {
	case texto := <-chegou:
		if !strings.Contains(texto, "a mesa mexeu") {
			t.Errorf("o quadro chegou como %q", texto)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("o quadro NÃO chegou com a conexão aberta: o `Flush` não atravessa o " +
			"gzip, e o tempo real da Mesa para sem erro nenhum")
	}
}

// O FLUSH ANTES DO PRIMEIRO WRITE COMPROMETE OS CABEÇALHOS.
//
// # O defeito que este caso existe para não deixar voltar
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
// Custou 27 casos vermelhos no e2e, e os guardas unitários daqui estavam TODOS
// verdes porque escreviam o cabeçalho antes de esvaziar.
func TestOFlushAntesDoWriteJaDecideOEnvelope(t *testing.T) {
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
	corpo, _ := io.ReadAll(zr)
	if !strings.Contains(string(corpo), "<p>oi</p>") {
		t.Errorf("o remendo não sobreviveu: %q", corpo)
	}
}

// A REGRA de quem comprime, exercitada direto.
func TestSoOsTiposQueValemSaoComprimidos(t *testing.T) {
	casos := []struct {
		tipo string
		quer bool
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
	for _, caso := range casos {
		h := http.Header{}
		if caso.tipo != "" {
			h.Set("Content-Type", caso.tipo)
		}
		if got := vaiComprimir(h, http.StatusOK); got != caso.quer {
			t.Errorf("vaiComprimir(%q) = %v, quer %v", caso.tipo, got, caso.quer)
		}
	}
	// RESPOSTA PEQUENA sai crua: comprimir 19 bytes produziu 43 no contêiner.
	// O corte usa o `Content-Length` DECLARADO, e por isso um fluxo — que nunca
	// declara tamanho — continua passando.
	pequena := http.Header{}
	pequena.Set("Content-Type", "text/html")
	pequena.Set("Content-Length", "19")
	if vaiComprimir(pequena, http.StatusOK) {
		t.Error("uma resposta de 19 bytes foi comprimida: o envelope do gzip a deixa MAIOR")
	}
	fluxo := http.Header{}
	fluxo.Set("Content-Type", "text/event-stream")
	if !vaiComprimir(fluxo, http.StatusOK) {
		t.Error("o fluxo não passou pelo corte de tamanho: ele não declara " +
			"`Content-Length`, e tratá-lo como pequeno mataria a compressão do tempo real")
	}

	// SEM CORPO não leva envelope: um gzip vazio sobre um 304 é lixo que alguns
	// clientes recusam.
	semCorpo := http.Header{}
	semCorpo.Set("Content-Type", "text/html")
	for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
		if vaiComprimir(semCorpo, status) {
			t.Errorf("o status %d não tem corpo e mesmo assim levou envelope gzip", status)
		}
	}
}
