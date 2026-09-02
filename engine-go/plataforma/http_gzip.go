package plataforma

import (
	"compress/gzip"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// COMPRIMIR O QUE O SERVIDOR RENDERIZA (ALE-273).
//
// A SPA já saía comprimida — mas do BUILD, com `.br` e `.gz` gerados ao lado de
// cada asset (ALE-153). O que o servidor RENDERIZA na hora não passava por nada:
// medido, a aba de Combate da ficha viaja 44,7 KB crus e 5,6 KB em gzip, e o
// mesmo HTML vai de novo a cada toque no PV, porque todo comando responde
// redesenhando a cena inteira.
//
// # Por que não um proxy na frente
//
// Porque a decisão do ALE-101 é um processo só, e comprimir aqui custa este
// arquivo. Um nginx compraria exatamente esta função e traria um segundo
// runtime, um segundo lugar para configurar TLS e um segundo lugar onde o SSE
// pode ser bufferizado por engano.
//
// # A ARMADILHA, e ela é a razão de este arquivo ter cuidado
//
// A resposta de um comando do Datastar é `text/event-stream` — ela usa o
// envelope de SSE para mandar UM remendo e fechar. A regra ingênua "não
// comprimir SSE" pularia justamente o que se quer comprimir; e a regra ingênua
// oposta, comprimir sem repassar o `Flush`, MATA o fluxo AO VIVO da Mesa: o
// `gzip.Writer` acumula num buffer interno, o quadro não sai, e o cliente fica
// esperando para sempre. Sem erro, sem log, sem nada na tela além de uma mesa
// que parou de atualizar.
//
// Por isso o `Flush` daqui esvazia o gzip ANTES de esvaziar quem está embaixo, e
// há guarda medindo o fluxo ao vivo com `Accept-Encoding: gzip`.

// tiposComprimiveis são os `Content-Type` que valem a pena.
//
// A lista é de PREFIXOS porque o cabeçalho carrega charset (`text/html;
// charset=utf-8`), e comparar a string inteira acertaria zero vezes.
var tiposComprimiveis = []string{
	"text/",                  // html, css, plain, event-stream
	"application/json",       // a API
	"application/javascript", // o `cena.js` do piloto
	"image/svg+xml",
}

// Gzip devolve um middleware que comprime o que o servidor gerou na hora.
//
// Ele NÃO toca em resposta que já chega com `Content-Encoding`: é o caso dos
// assets pré-comprimidos da SPA, e recomprimi-los gastaria CPU para produzir
// bytes maiores.
func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !AcceptsEncoding(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		// `Vary` sai SEMPRE que a negociação aconteceu, inclusive quando ela
		// decide não comprimir: sem ele, um cache guardaria a resposta
		// comprimida e a serviria a quem não aceita gzip.
		w.Header().Add("Vary", "Accept-Encoding")
		envelope := &gzipEnvelope{ResponseWriter: w}
		defer envelope.Close()
		next.ServeHTTP(envelope, r)
	})
}

// gzipEnvelope decide no PRIMEIRO write se vale comprimir, porque é só aí
// que o handler já escreveu o `Content-Type`.
type gzipEnvelope struct {
	http.ResponseWriter
	gz       *gzip.Writer
	decidido bool
	uma      sync.Once
}

// poolDeGzip reaproveita os escritores: um `gzip.NewWriter` aloca ~260 KB de
// janela, e a ficha faz um por toque no PV.
var poolDeGzip = sync.Pool{
	New: func() any { return gzip.NewWriter(nil) },
}

func (e *gzipEnvelope) WriteHeader(status int) {
	e.decide(status)
	e.ResponseWriter.WriteHeader(status)
}

// decide resolve, uma vez, se esta resposta vai comprimida.
func (e *gzipEnvelope) decide(status int) {
	if e.decidido {
		return
	}
	e.decidido = true
	if !vaiComprimir(e.Header(), status) {
		return
	}
	e.Header().Set("Content-Encoding", "gzip")
	// O TAMANHO deixa de valer no instante em que o corpo muda: um
	// `Content-Length` do texto cru faria o cliente esperar bytes que nunca
	// chegam, ou cortar o corpo no meio.
	e.Header().Del("Content-Length")
	e.gz = poolDeGzip.Get().(*gzip.Writer)
	e.gz.Reset(e.ResponseWriter)
}

// vaiComprimir é a REGRA, separada para o guarda poder exercitá-la direto.
func vaiComprimir(h http.Header, status int) bool {
	// Já vem comprimido do build (`.br`/`.gz` da SPA) — não recomprimir.
	if h.Get("Content-Encoding") != "" {
		return false
	}
	// 204 e 304 não têm corpo; um envelope gzip vazio sobre eles é lixo que
	// alguns clientes recusam.
	if status == http.StatusNoContent || status == http.StatusNotModified {
		return false
	}
	// RESPOSTA PEQUENA SAI CRUA: o envelope do gzip custa ~24 bytes fixos, e
	// medido no contêiner uma resposta de 19 bytes virou 43. Abaixo de um MTU a
	// compressão não economiza um pacote sequer.
	//
	// O corte olha o `Content-Length` que o handler JÁ declarou, e nunca o
	// tamanho do corpo — essa é a diferença que mantém o fluxo ao vivo vivo. Um
	// limiar de verdade precisaria BUFERIZAR até saber o tamanho, e bufferizar é
	// exatamente o que mata o SSE; um fluxo nunca declara `Content-Length`, então
	// ele não passa por aqui.
	// RESPOSTA PEQUENA SAI CRUA: o envelope do gzip custa ~24 bytes fixos, e
	// medido no contêiner uma resposta de 19 bytes virou 43. Abaixo de um MTU a
	// compressão não economiza um pacote sequer.
	//
	// O corte olha o `Content-Length` que o handler JÁ declarou, e nunca o
	// tamanho do corpo — essa é a diferença que mantém o fluxo ao vivo vivo. Um
	// limiar de verdade precisaria BUFERIZAR até saber o tamanho, e bufferizar é
	// exatamente o que mata o SSE; um fluxo nunca declara `Content-Length`, então
	// ele não passa por aqui.
	if n, err := strconv.Atoi(h.Get("Content-Length")); err == nil && n < umMTU {
		return false
	}
	tipo := h.Get("Content-Type")
	for _, prefixo := range tiposComprimiveis {
		if strings.HasPrefix(tipo, prefixo) {
			return true
		}
	}
	return false
}

// umMTU é o piso abaixo do qual comprimir só acrescenta bytes.
const umMTU = 1400

func (e *gzipEnvelope) Write(b []byte) (int, error) {
	if !e.decidido {
		// Handler que escreve sem chamar `WriteHeader`: o `net/http` assume 200,
		// e a decisão tem de acontecer aqui, ANTES do primeiro byte.
		e.decide(http.StatusOK)
	}
	if e.gz == nil {
		return e.ResponseWriter.Write(b)
	}
	return e.gz.Write(b)
}

// Flush esvazia o GZIP antes de quem está embaixo, e essa ordem é o arquivo
// inteiro.
//
// Invertida — ou ausente — o quadro do SSE fica preso no buffer do gzip e a Mesa
// para de atualizar sem nada acusar. É a mesma família das armadilhas do
// Datastar: nada falha, alguém só espera para sempre.
func (e *gzipEnvelope) Flush() {
	// UM FLUSH COMPROMETE OS CABEÇALHOS, então a decisão tem de estar tomada
	// aqui — e não só no `Write`.
	//
	// O `datastar-go` escreve o `Content-Type`, chama `Flush()` para mandar os
	// cabeçalhos e SÓ ENTÃO escreve o primeiro remendo. Decidindo apenas no
	// `Write`, o `Content-Encoding` chega depois de a resposta já ter saído sem
	// ele, e o corpo vai comprimido mesmo assim: o cliente lê bytes de gzip como
	// texto puro, os remendos param de ser aplicados, e nada falha em lugar
	// nenhum. Foram 27 casos vermelhos no e2e com os unitários todos verdes,
	// porque eles escreviam o cabeçalho antes de esvaziar.
	if !e.decidido {
		e.decide(http.StatusOK)
	}
	if e.gz != nil {
		_ = e.gz.Flush()
	}
	if quemEsvazia, ok := e.ResponseWriter.(http.Flusher); ok {
		quemEsvazia.Flush()
	}
}

// Unwrap devolve quem está embaixo, que é o contrato do `http.ResponseController`
// desde o Go 1.20.
//
// Sem ele, um `rc.SetWriteDeadline` ou `rc.Hijack` chamado por qualquer camada
// acima deste envelope responde `ErrNotSupported` — e a biblioteca que o chamou
// decide que o ambiente não suporta fluxo, sem que nada aqui tenha errado.
func (e *gzipEnvelope) Unwrap() http.ResponseWriter {
	return e.ResponseWriter
}

func (e *gzipEnvelope) Close() {
	e.uma.Do(func() {
		if e.gz == nil {
			return
		}
		_ = e.gz.Close()
		poolDeGzip.Put(e.gz)
		e.gz = nil
	})
}
