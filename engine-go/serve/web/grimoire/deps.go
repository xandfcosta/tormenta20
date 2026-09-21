package grimoire

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/serve/web/ui"
)

// A PORTA do grimório, que é a FOLHA DE ESPECIFICAÇÃO: cada peça do sistema de
// desenho, lado a lado, para ser conferida com o olho.
//
// É a menor porta do projeto, e não por acidente — uma folha de especificação
// não lê banco, não computa regra e não sabe de personagem. Se ela crescer, a
// pergunta é o que está sendo desenhado aqui que não é peça.
type Deps interface {
	// Asset monta o endereço versionado de um estático: a folha carrega a ilha de
	// JS dela (`grimorio.js`).
	Asset(file string) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
}

// Scene é o grimório montado com as dependências dele.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
