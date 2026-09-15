package master

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/serve/web/bookui"
	"t20engine/serve/web/ui"
)

// A PORTA do trilho do mestre: DOIS métodos, e o tamanho é o que a cena É. As
// treze unidades do `/mestre/*` desenham o LIVRO, que é igual para todo mundo e
// chega por `go:embed` — não há banco, campanha nem personagem.
type Deps interface {
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	// BookAddress é onde o PDF está, para o selo "p289" saber para onde apontar.
	// Vem por aqui e não por `book` porque não é dado do livro: é CONFIGURAÇÃO
	// (`LIVRO_PDF`), e sem ela o selo não é desenhado.
	BookAddress() bookui.BookAddress
}

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
