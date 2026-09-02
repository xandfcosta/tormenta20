package grimoire

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/web/ui"
)

// A PORTA do grimório (ALE-278, terceira cena).
//
// O grimório é a FOLHA DE ESPECIFICAÇÃO: ele desenha cada peça do sistema de
// desenho lado a lado com a contraparte da SPA, para as duas serem comparadas
// com o olho. Ele foi o instrumento da migração — quem descobriu que o
// `secondary` do templ era o `outline` da SPA com o nome errado (ALE-250) foi
// esta tela.
//
// A porta dele é a menor do projeto e isso não é acidente: uma folha de
// especificação não lê banco, não computa regra e não sabe de personagem. Se um
// dia ela crescer, a pergunta é o que está sendo desenhado aqui que não é peça.
type Deps interface {
	// Asset monta o endereço versionado de um estático. A folha carrega o bundle
	// dos componentes da SPA para desenhar a coluna de comparação.
	Asset(arquivo string) string
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Scene é o grimório montado com as dependências dele.
type Scene struct{ deps Deps }

// New monta a cena.
func New(d Deps) Scene { return Scene{deps: d} }
