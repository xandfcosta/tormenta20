package master

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/a-h/templ"

	"t20engine/web/bookui"
	"t20engine/web/routes"
	"t20engine/web/ui"
)

// semLivro é a porta cumprida por um dublê NOMEADO, e não por um stub inline
// (CLAUDE.md, "Testes").
//
// Ele responde "não há PDF configurado", que é o caso REAL de quem não apontou
// `LIVRO_PDF` — e não um valor inventado para o teste passar. A consequência de
// verdade é que o selo "p289" não é desenhado, e é isso que os casos daqui
// medem quando medem o desenho.
//
// `WritePage` não faz nada porque nenhum caso deste pacote monta página: montar
// página é composição, e a composição ficou no `api` com o `Server` de verdade.
// Se algum caso vier a chamá-lo, o vazio aqui é o sinal de que ele está na
// camada errada.
type semLivro struct{}

func (semLivro) BookAddress() bookui.BookAddress { return bookui.BookAddress{} }

// WritePage do dublê escreve o CORPO e não a casca.
//
// A casca — o cabeçalho, os estáticos, as sobreposições — é do `api`, e é lá
// que ela tem guarda. O que os casos deste pacote afirmam mora no corpo: o
// trilho, os rótulos das paradas, os cartões do acervo.
//
// Escrever NADA aqui seria pior que não ter o dublê: a resposta sairia 200 com
// corpo vazio, e todo `strings.Contains` passaria a medir a ausência da casca
// em vez do conteúdo da cena. Foi o que aconteceu na primeira versão, e onze
// subcasos reprovaram de uma vez dizendo "a cena não desenha o próprio nome"
// quando a cena estava inteira.
func (semLivro) WritePage(
	w http.ResponseWriter, r *http.Request, status int, _ ui.Page, corpo templ.Component,
) {
	w.WriteHeader(status)
	_ = corpo.Render(r.Context(), w)
}

// cenaSemLivro é a cena montada com o dublê, para os casos que só precisam do
// desenho.
func cenaSemLivro() Scene { return New(semLivro{}) }

// pedeNaCena dirige as rotas DESTA cena, sem hospedeiro.
//
// A composição que ela prova é a do pacote — `Routes` registrou, o handler leu
// a URL, o componente desenhou — e ela não precisa de sessão porque a cena não
// tem autorização própria: o bestiário e os catálogos são o LIVRO, igual para
// todo mundo, e o `requirePage` que exige sessão é do `api`.
//
// O que fica no hospedeiro é a outra pergunta: a cena está MONTADA e atrás do
// login? Uma camada afirma a fronteira; as outras afirmam presença e ligação.
func pedeNaCena(t *testing.T, alvo string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	Routes(r, cenaSemLivro())
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, alvo, nil))
	return rec
}

// comLivro é a porta de quem CONFIGUROU o PDF, para os casos que medem o selo
// "p289" e o botão que abre o leitor.
type comLivro struct{ semLivro }

func (comLivro) BookAddress() bookui.BookAddress {
	return bookui.BookAddress{Base: routes.Book, Abertura: 6}
}

func pedeNaCenaComLivro(t *testing.T, alvo string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	Routes(r, New(comLivro{}))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, alvo, nil))
	return rec
}
