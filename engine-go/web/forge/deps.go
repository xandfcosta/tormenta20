package forge

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/db/sqlcgen"
	"t20engine/engine"
	"t20engine/sheet"
	"t20engine/web/ui"
)

// A PORTA da forja (ALE-278).
//
// Ela é a primeira cena a sair do `api`, e a razão de ter sido possível é que
// três camadas saíram antes: a forma da ficha (`sheet`), o catálogo tipado
// (`book`) e o kit de apresentação (`web/ui`). Enquanto elas eram do `api`,
// qualquer cena que se mudasse levava o `api` junto — e o `api` importa a cena de
// volta para montar rota, que é ciclo.
//
// # O que ela pede, e o que ela deliberadamente NÃO pede
//
// Medido antes de escrever: a forja tocava UM campo do `Server` (`queries`) e
// cinco métodos. Dois deles — `ComputeSheet` e `LoadCharacter` — viraram funções
// do `sheet` na terceira camada, então a forja as chama direto e eles saíram
// desta lista. Sobraram quatro.
//
// A porta é declarada AQUI, no consumidor, e não no `api`: é o que a torna uma
// porta e não um segundo nome para o objeto-deus. O `*api.Server` a cumpre por
// acidente feliz de já ter esses métodos — e no dia em que ele deixar de ter, o
// compilador acusa do lado de quem monta, que é onde a decisão mora.
type Deps interface {
	// Queries é o banco. A forja lê o personagem que está sendo forjado e grava
	// os atributos distribuídos.
	Queries() *sqlcgen.Queries
	// Catalogs é o motor primado, para computar a ficha do herói recém-nascido.
	Catalogs() *engine.Catalogs
	// InsertCharacter faz nascer o herói. Ela fica no `api` porque a criação de
	// personagem é um caminho compartilhado — a forja é uma porta de entrada
	// dela, não a dona.
	// As duas recebem CONTEXTO e não o `*http.Request` (ALE-287): elas só liam
	// o `r.Context()` dele, e pedir a requisição inteira obrigava quem chama a
	// ter uma — o que o gerador da seed não tem, e foi ele que denunciou.
	InsertCharacter(
		ctx context.Context, ownerID int64, name string, body sheet.CreateBody,
		totalLevel int64, granted []string, trained map[string]bool,
	) (int64, error)
	// HealVitals enche PV e PM depois do nascimento.
	HealVitals(ctx context.Context, id int64, dto *sheet.CharacterDTO) error
	// WritePage é a montagem da casca: ela injeta os estáticos e as
	// sobreposições que a cena não pode conhecer (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro.
	//
	// Duas razões, e a segunda é a que decide. A primeira: é só disto que a forja
	// precisa — ela confere a posse do herói e nada mais. A segunda: o tipo do
	// usuário é do `api`, e pedi-lo aqui faria esta cena IMPORTAR o `api`, que a
	// importa de volta para montar rota. Uma porta que devolve tipo do
	// hospedeiro não é porta.
	CurrentUserID(r *http.Request) int64
}

// Scene é a forja montada com as dependências dela.
//
// Uma struct e não a interface direta porque o Go não aceita interface como
// RECEPTOR — e um invólucro de um campo é o preço de os handlers continuarem
// métodos, que é o que mantém as rotas legíveis.
type Scene struct{ deps Deps }

// New monta a cena. Quem chama é quem tem as dependências na mão: hoje o `api`,
// no `WebRouter`.
func New(d Deps) Scene { return Scene{deps: d} }
