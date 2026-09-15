package forge

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
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
	// InsertCharacter fica no hospedeiro porque criar personagem é caminho
	// compartilhado: a forja é uma entrada dele, não a dona.
	//
	// Ela recebe CONTEXTO e não o `*http.Request`: pedir a requisição inteira
	// obrigaria quem chama a ter uma, e o gerador da seed não tem.
	InsertCharacter(
		ctx context.Context, ownerID int64, name string, body sheet.CreateBody,
		totalLevel int64, granted []string, trained map[string]bool,
	) (int64, error)
	// HealVitals enche PV e PM depois do nascimento.
	HealVitals(ctx context.Context, id int64, dto *sheet.CharacterDTO) error
	// ShiftVitalsToNewMax recompute os máximos e faz os ATUAIS acompanharem o
	// delta, em vez de encherem. É o que um passo de atributo faz com os poços
	// de um herói que já apanhou (ALE-309).
	ShiftVitalsToNewMax(ctx context.Context, id int64, dto *sheet.CharacterDTO) error
	// WritePage é a montagem da casca (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	CurrentUserID(r *http.Request) int64
}

// Scene é a forja montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
