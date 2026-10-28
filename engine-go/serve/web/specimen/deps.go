package specimen

import (
	"net/http"

	"github.com/a-h/templ"

	"t20engine/serve/web/ui"
)

// A PORTA da folha viva: cada peça do kit Tempestade, nos dois modos, para o
// olho conferir e o guarda de aparência medir.
//
// Ela é tão estreita quanto a do grimório, e pelo mesmo motivo: uma folha de
// peças não lê banco, não computa regra e não sabe o que é um personagem. Os
// números que ela mostra são de exemplo, escritos aqui.
type Deps interface {
	// WritePage é a montagem da casca.
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
}

// Scene é a folha viva montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
