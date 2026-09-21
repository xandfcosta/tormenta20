package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t20engine/infra/config"
)

// O guarda do LIVRO servido.
//
// O defeito que ele prende é o que MAIS parece certo: `#page=N` conta páginas do
// ARQUIVO e o catálogo grava a página IMPRESSA. Sem somar a abertura, o botão
// abre seis páginas antes — no MESMO capítulo, com a diagramação parecida, e
// ninguém desconfia. MEDIDO pelo número no RODAPÉ do PDF da casa: a página 295
// do arquivo imprime "289", a 297 imprime "291", a 203 imprime "197".
//
// O que este guarda NÃO afirma: que a criatura comece naquela página. Medido
// contra o livro, algumas entradas do `bestiary.json` erram a página por uma —
// o bloco do Lobo abre na impressa 290 e o catálogo diz 289. É imprecisão do
// DADO, anterior a isto, e a mesma que a linha da lista já mostra ("p289"); o
// botão só a torna visível.

// fakeBookFile grava um arquivo com conteúdo conhecido e devolve o caminho.
// O conteúdo importa em UM lugar só (a marca de linearizado); para a rota, o que
// importa é que ele exista e tenha bytes contáveis.
func serverWithBook(t *testing.T, s *Server, content string) *Server {
	t.Helper()
	s.book = openServedBook(config.Config{BookPDF: fakeBookFile(t, content), BookPageOffset: 6})
	if s.book.path == "" {
		t.Fatal("o livro de mentira não foi aceito — o resto do guarda mediria a ausência")
	}
	return s
}

func fakeBookFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "livro.pdf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("gravar o livro de mentira: %v", err)
	}
	return path
}

func TestAMissingBookDoesNotBringTheServerDown(t *testing.T) {
	l := openServedBook(config.Config{BookPDF: filepath.Join(t.TempDir(), "nao-existe.pdf")})
	if l.path != "" || l.address.Base != "" {
		t.Errorf("um caminho inexistente virou livro servido: %+v", l)
	}
}

// Os dois lados da marca de linearizado.
//
// A marca vive no PRIMEIRO objeto do arquivo por definição da especificação —
// procurá-la no arquivo inteiro seria ler 89 MB no boot para responder um aviso.
func TestLinearizationIsRecognizedAtTheStartOfTheFile(t *testing.T) {
	if !isLinearized([]byte("%PDF-1.6\n1 0 obj\n<< /Linearized 1 /L 78622788 >>")) {
		t.Error("um PDF linearizado foi lido como não linearizado — o aviso sairia sempre")
	}
	if isLinearized([]byte("%PDF-1.6\n1 0 obj\n<< /Type /Catalog >>")) {
		t.Error("um PDF cru passou por linearizado — o aviso nunca sairia")
	}
}

// O `Range` é o que faz o navegador pedir só a página.
//
// `Range` é a metade do arranjo que a linearização usa; sem ele o visualizador
// baixa o arquivo inteiro por mais linearizado que ele esteja. O controle é o
// pedido INTEIRO logo abaixo: ele prova que a rota serve e que o 206 do outro
// caso é recorte e não erro.
func TestTheBookRouteServesRanges(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "0123456789abcdef")
	eu := seedUser(t, s, "mestre@t20.local")

	whole := askForTheBook(t, s, eu, "")
	if whole.Code != http.StatusOK || whole.Body.String() != "0123456789abcdef" {
		t.Fatalf("o livro inteiro deu %d com %q", whole.Code, whole.Body.String())
	}
	if whole.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("a rota não anuncia faixas — o visualizador baixa o arquivo todo")
	}

	httpRange := askForTheBook(t, s, eu, "bytes=0-3")
	if httpRange.Code != http.StatusPartialContent {
		t.Fatalf("um pedido de faixa deu %d, e não 206", httpRange.Code)
	}
	if httpRange.Body.String() != "0123" {
		t.Errorf("a faixa trouxe %q — o servidor mandou o arquivo inteiro", httpRange.Body.String())
	}
}

// O livro sai com cache PRIVADO, porque ele passa pelo `requirePage`.
//
// `public` autorizaria um cache compartilhado a guardar a resposta de quem
// entrou e reentregá-la a quem não entrou. A folha e as fontes são `public` de
// propósito — elas saem sem sessão.
func TestTheBookLeavesWithAPrivateCache(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	rec := askForTheBook(t, s, eu, "")
	cache := rec.Header().Get("Cache-Control")
	if !strings.Contains(cache, "private") || !strings.Contains(cache, "immutable") {
		t.Errorf("o livro versionado saiu com Cache-Control %q", cache)
	}
}

// A rota está atrás da mesma porta do resto.
func TestTheBookDoesNotLeaveWithoutASession(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")

	req := httptest.NewRequest(http.MethodGet, "/livro?v="+s.book.digest, nil)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Error("o livro saiu para quem não entrou na mesa")
	}
}

// Nada é servido por acidente.
func TestWithoutConfigurationTheBookRouteGives404(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	if rec := askForTheBook(t, s, eu, ""); rec.Code != http.StatusNotFound {
		t.Errorf("sem LIVRO_PDF a rota deu %d, e não 404", rec.Code)
	}
}

func askForTheBook(t *testing.T, s *Server, userID int64, httpRange string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.accountGate().SignSession(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/livro?v="+s.book.digest, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if httpRange != "" {
		req.Header.Set("Range", httpRange)
	}
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// O botão, ponta a ponta.
//
// É INTEGRAÇÃO e não asserção de componente porque o que se quer proteger é a
// composição — o endereço nasce na configuração, atravessa o `Server`, a view e
// duas telas até virar `href`. Cada um desses saltos já sumiu em silêncio uma
// vez nesta migração.
//
// O controle é o segundo caso: sem livro configurado a MESMA cena não traz
// `#page=` nenhum, então este guarda mede presença contra ausência e não contra
// uma string que estaria lá de qualquer jeito.
func TestTheBestiarySceneOpensTheBookAtTheCreaturePage(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/mestre/bestiario?criatura=lobo", "").Body.String()
	// O endereço leva ao LEITOR, na página impressa e com o nome a destacar.
	//
	// 290 e não 289: o bloco do Lobo abre na impressa 290, e o catálogo dizia
	// 289 porque a p289 tem "lobos-das-cavernas" no texto corrido — a
	// conferência por substring aprovava a página que CITA em vez da que ABRE.
	// Corrigido pela assinatura "<nome> nd <valor>", que é como o livro imprime
	// o começo de todo bloco de criatura.
	if !strings.Contains(body, "/livro/ler?p=290&amp;t=Lobo") {
		t.Error("a ficha do Lobo não abre o leitor na página dele")
	}

	withoutBook := newTestServer(t)
	other := seedUser(t, withoutBook, "mestre@t20.local")
	without := pedeNoMestre(t, withoutBook, other, "GET", "/mestre/bestiario?criatura=lobo", "").Body.String()
	if strings.Contains(without, "/livro/ler") {
		t.Error("sem LIVRO_PDF a cena desenhou um link para um livro que não é servido")
	}
	if !strings.Contains(without, "p289") {
		t.Error("sem livro a página impressa devia continuar escrita na ficha")
	}
}

// O contrato entre a cena e o leitor.
//
// O leitor é JavaScript: o pdf.js lê o PDF, desenha o canvas e posiciona as
// marcas. Nada disso cabe num teste de handler, e é o `e2e/tests/book-reader.spec.ts`
// que mede. O que cabe AQUI é o contrato entre os dois — os cinco dados que o
// servidor escreve no `<div id="reader">`. Errar um deles quebra o leitor em
// silêncio: sem `data-abertura` ele abre seis páginas antes, sem `data-worker` o
// pdf.js cai no modo sem worker e trava a aba.
func TestTheBookReaderLoadsWhatTheSceneNeeds(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	body := pedeNoMestre(t, s, eu, "GET", "/livro/ler?p=290&t=Lobo", "").Body.String()
	for _, data := range []string{
		`data-pagina="290"`,
		`data-termo="Lobo"`,
		`data-abertura="6"`,
		`data-worker="/static/pdf.worker.js`,
		`data-livro="/livro?v=`,
	} {
		if !strings.Contains(body, data) {
			t.Errorf("a cena do leitor não escreveu %s", data)
		}
	}
	// O módulo do leitor só entra NESTA cena: são 540 KB de pdf.js.
	if !strings.Contains(body, "reader.js") {
		t.Error("a cena não carrega o módulo do leitor")
	}
	if bestiary := pedeNoMestre(t, s, eu, "GET", "/mestre/bestiario", "").Body.String(); strings.Contains(bestiary, "reader.js") {
		t.Error("o bestiário carregou o pdf.js — 540 KB no caminho de quem só quer a ficha")
	}
}

// A cena não existe sem o arquivo, como a rota do PDF.
func TestWithoutABookThereIsNoReader(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	if rec := pedeNoMestre(t, s, eu, "GET", "/livro/ler?p=290", ""); rec.Code != http.StatusNotFound {
		t.Errorf("sem LIVRO_PDF o leitor respondeu %d, e não 404", rec.Code)
	}
}

// O endereço é compartilhável e se digita à mão.
func TestTheReaderPageRefusesGarbage(t *testing.T) {
	s := serverWithBook(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	for _, target := range []string{"/livro/ler?p=abacaxi", "/livro/ler?p=-3", "/livro/ler"} {
		body := pedeNoMestre(t, s, eu, "GET", target, "").Body.String()
		if !strings.Contains(body, `data-pagina="1"`) {
			t.Errorf("%s não caiu na primeira página — a cena aceitou lixo", target)
		}
	}
}
