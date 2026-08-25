package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t20engine/plataforma"
)

// O guarda do LIVRO servido (ALE-264).
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

// livroDeMentira grava um arquivo com conteúdo conhecido e devolve o caminho.
// O conteúdo importa em UM lugar só (a marca de linearizado); para a rota, o que
// importa é que ele exista e tenha bytes contáveis.
func livroDeMentira(t *testing.T, conteudo string) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "livro.pdf")
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("gravar o livro de mentira: %v", err)
	}
	return caminho
}

func servidorComLivro(t *testing.T, s *Server, conteudo string) *Server {
	t.Helper()
	s.livro = abreOLivro(plataforma.Config{LivroPDF: livroDeMentira(t, conteudo), LivroAbertura: 6})
	if s.livro.caminho == "" {
		t.Fatal("o livro de mentira não foi aceito — o resto do guarda mediria a ausência")
	}
	return s
}

// TestOBotaoAbreAPaginaDoArquivoENaoAImpressa: a soma da abertura.
//
// O controle está no segundo caso: com abertura zero o endereço traz o número
// IMPRESSO, então este guarda vê a diferença entre somar e não somar — sem ele,
// um `+ l.Abertura` apagado passaria verde contra um número que também existe.
func TestOBotaoAbreAPaginaDoArquivoENaoAImpressa(t *testing.T) {
	comAbertura := enderecoDoLivro{Base: "/piloto/livro?v=abc", Abertura: 6}
	if got := comAbertura.naPagina(289); got != "/piloto/livro?v=abc#page=295" {
		t.Errorf("a página impressa 289 abriu em %q — no arquivo ela é a 295", got)
	}
	semAbertura := enderecoDoLivro{Base: "/piloto/livro?v=abc"}
	if got := semAbertura.naPagina(289); got != "/piloto/livro?v=abc#page=289" {
		t.Errorf("sem abertura o endereço devia trazer a página impressa, e trouxe %q", got)
	}
}

// TestSemLivroConfiguradoNaoHaEndereco: o zero valor não produz link quebrado.
func TestSemLivroConfiguradoNaoHaEndereco(t *testing.T) {
	if got := (enderecoDoLivro{}).naPagina(289); got != "" {
		t.Errorf("sem livro o endereço devia ser vazio, e foi %q", got)
	}
	if got := (enderecoDoLivro{Base: "/piloto/livro"}).naPagina(0); got != "" {
		t.Errorf("criatura sem página no livro devia ficar sem endereço, e ficou %q", got)
	}
}

// TestOLivroAusenteNaoDerrubaOServidor: configurar errado degrada, não quebra.
func TestOLivroAusenteNaoDerrubaOServidor(t *testing.T) {
	l := abreOLivro(plataforma.Config{LivroPDF: filepath.Join(t.TempDir(), "nao-existe.pdf")})
	if l.caminho != "" || l.endereco.Base != "" {
		t.Errorf("um caminho inexistente virou livro servido: %+v", l)
	}
}

// TestALinearizacaoEReconhecidaNoComecoDoArquivo: os dois lados da marca.
//
// A marca vive no PRIMEIRO objeto do arquivo por definição da especificação —
// procurá-la no arquivo inteiro seria ler 89 MB no boot para responder um aviso.
func TestALinearizacaoEReconhecidaNoComecoDoArquivo(t *testing.T) {
	if !ehLinearizado([]byte("%PDF-1.6\n1 0 obj\n<< /Linearized 1 /L 78622788 >>")) {
		t.Error("um PDF linearizado foi lido como não linearizado — o aviso sairia sempre")
	}
	if ehLinearizado([]byte("%PDF-1.6\n1 0 obj\n<< /Type /Catalog >>")) {
		t.Error("um PDF cru passou por linearizado — o aviso nunca sairia")
	}
}

// TestARotaDoLivroEntregaFaixas: é o que faz o navegador pedir só a página.
//
// `Range` é a metade do arranjo que a linearização usa; sem ele o visualizador
// baixa o arquivo inteiro por mais linearizado que ele esteja. O controle é o
// pedido INTEIRO logo abaixo: ele prova que a rota serve e que o 206 do outro
// caso é recorte e não erro.
func TestARotaDoLivroEntregaFaixas(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "0123456789abcdef")
	eu := seedUser(t, s, "mestre@t20.local")

	inteiro := pedeOLivro(t, s, eu, "")
	if inteiro.Code != http.StatusOK || inteiro.Body.String() != "0123456789abcdef" {
		t.Fatalf("o livro inteiro deu %d com %q", inteiro.Code, inteiro.Body.String())
	}
	if inteiro.Header().Get("Accept-Ranges") != "bytes" {
		t.Error("a rota não anuncia faixas — o visualizador baixa o arquivo todo")
	}

	faixa := pedeOLivro(t, s, eu, "bytes=0-3")
	if faixa.Code != http.StatusPartialContent {
		t.Fatalf("um pedido de faixa deu %d, e não 206", faixa.Code)
	}
	if faixa.Body.String() != "0123" {
		t.Errorf("a faixa trouxe %q — o servidor mandou o arquivo inteiro", faixa.Body.String())
	}
}

// TestOLivroSaiComCachePRIVADO: ele passa pelo `requirePagina`.
//
// `public` autorizaria um cache compartilhado a guardar a resposta de quem
// entrou e reentregá-la a quem não entrou. A folha e as fontes são `public` de
// propósito — elas saem sem sessão.
func TestOLivroSaiComCachePrivado(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	rec := pedeOLivro(t, s, eu, "")
	cache := rec.Header().Get("Cache-Control")
	if !strings.Contains(cache, "private") || !strings.Contains(cache, "immutable") {
		t.Errorf("o livro versionado saiu com Cache-Control %q", cache)
	}
}

// TestOLivroNaoSaiSemSessao: a rota está atrás da mesma porta do resto.
func TestOLivroNaoSaiSemSessao(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "%PDF-1.6")

	req := httptest.NewRequest(http.MethodGet, "/piloto/livro?v="+s.livro.digito, nil)
	rec := httptest.NewRecorder()
	http.StripPrefix("/piloto", s.PilotoRouter()).ServeHTTP(rec, req)
	if rec.Code == http.StatusOK {
		t.Error("o livro saiu para quem não entrou na mesa")
	}
}

// TestSemConfiguracaoARotaDoLivroDa404: nada é servido por acidente.
func TestSemConfiguracaoARotaDoLivroDa404(t *testing.T) {
	s := newTestServer(t)
	eu := seedUser(t, s, "mestre@t20.local")

	if rec := pedeOLivro(t, s, eu, ""); rec.Code != http.StatusNotFound {
		t.Errorf("sem LIVRO_PDF a rota deu %d, e não 404", rec.Code)
	}
}

func pedeOLivro(t *testing.T, s *Server, userID int64, faixa string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(t.Context(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/piloto/livro?v="+s.livro.digito, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if faixa != "" {
		req.Header.Set("Range", faixa)
	}
	rec := httptest.NewRecorder()
	http.StripPrefix("/piloto", s.PilotoRouter()).ServeHTTP(rec, req)
	return rec
}

// TestACenaDoBestiarioAbreOLivroNaPaginaDaCriatura: o botão, ponta a ponta.
//
// É INTEGRAÇÃO e não asserção de componente porque o que se quer proteger é a
// composição — o endereço nasce na configuração, atravessa o `Server`, a view e
// duas telas até virar `href`. Cada um desses saltos já sumiu em silêncio uma
// vez nesta migração.
//
// O controle é o segundo caso: sem livro configurado a MESMA cena não traz
// `#page=` nenhum, então este guarda mede presença contra ausência e não contra
// uma string que estaria lá de qualquer jeito.
func TestACenaDoBestiarioAbreOLivroNaPaginaDaCriatura(t *testing.T) {
	s := servidorComLivro(t, newTestServer(t), "%PDF-1.6")
	eu := seedUser(t, s, "mestre@t20.local")

	corpo := pedeNoMestre(t, s, eu, "GET", "/piloto/mestre/bestiario?criatura=lobo", "").Body.String()
	// O Lobo diz p289, e a página impressa 289 é a 295 do arquivo.
	if !strings.Contains(corpo, "#page=295") {
		t.Error("a ficha do Lobo não linka a página 295 do arquivo")
	}

	semLivro := newTestServer(t)
	outro := seedUser(t, semLivro, "mestre@t20.local")
	sem := pedeNoMestre(t, semLivro, outro, "GET", "/piloto/mestre/bestiario?criatura=lobo", "").Body.String()
	if strings.Contains(sem, "#page=") {
		t.Error("sem LIVRO_PDF a cena desenhou um link para um livro que não é servido")
	}
	if !strings.Contains(sem, "p289") {
		t.Error("sem livro a página impressa devia continuar escrita na ficha")
	}
}
