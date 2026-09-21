package characters

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/serve/web/ui"
)

// A PORTA da cena de PERSONAGENS, que lista os heróis de quem está olhando.
type Deps interface {
	CurrentUserID(r *http.Request) int64
	// CharacterList: a cena faz a PERGUNTA, o hospedeiro sabe de qual tabela ela
	// sai.
	CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error)
	// Catalogs pode vir NULO, e não é descuido: sem ele o cartão mostra
	// travessão no lugar da Defesa e a lista continua de pé (ver `HeroCardOf`).
	Catalogs() *engine.Catalogs
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, body templ.Component)
}

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
