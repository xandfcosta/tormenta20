package forge

import (
	"net/http"
	"t20engine/app/character"

	"github.com/a-h/templ"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/ui"
)

// A PORTA da forja — a cena que faz um herói nascer.
type Deps interface {
	// Queries é o banco. A forja lê o personagem que está sendo forjado e grava
	// os atributos distribuídos.
	Queries() *sqlcgen.Queries
	// Catalogs é o motor primado, para computar a ficha do herói recém-nascido.
	Catalogs() *engine.Catalogs
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
	CurrentUserID(r *http.Request) int64
}

// Scene é a forja montada com as dependências dela.
type Scene struct {
	deps Deps
	// births é o CASO DE USO do nascimento, e chega por parâmetro e não pela
	// porta: o `app/character` está ABAIXO desta cena, então ela o importa
	// direto. A porta dizia, antes desta camada existir, que criar personagem é
	// "caminho compartilhado" e que "a forja é uma entrada dele, não a dona" —
	// era a descrição de um caso de uso sem endereço.
	births character.Births
}

func New(d Deps, birth character.Births) Scene {
	return Scene{deps: d, births: birth}
}
