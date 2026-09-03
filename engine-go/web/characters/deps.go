package characters

import (
	"context"
	"net/http"

	"github.com/a-h/templ"

	"t20engine/engine"
	"t20engine/sheet"
	"t20engine/web/ui"
)

// A PORTA da cena de PERSONAGENS (ALE-278).
//
// Ela lista os heróis de quem está olhando: `/personagens`. Três métodos, e os
// três são a mesma pergunta vista de ângulos diferentes — quem é você, o que é
// seu, e como isso vira página.
//
// # O que ela deliberadamente NÃO pede
//
// O `sheetFromDTO` do hospedeiro esteve nesta lista e saiu depois de medido:
// ele é um invólucro de UMA linha sobre `sheet.Compute(catalogs, dto)`, e a
// cena já tem o `Catalogs()`. Pedir os dois seria pedir a mesma coisa duas
// vezes — a regra da menor pergunta, com a nuance de que aqui o menor não é o
// mais estreito, é o que não se repete.
//
// E o usuário INTEIRO também não: `Load` recebia um `AuthUser`, e
// o que ele lia dele era o `ID`. O tipo é do `api`, então pedi-lo aqui faria
// esta cena importar o hospedeiro, que a importa de volta para montar rota. É a
// mesma linha que a forja e a porta de entrar já tinham escrito.
type Deps interface {
	// CurrentUserID é quem está pedindo, pelo ID e não pelo usuário inteiro.
	CurrentUserID(r *http.Request) int64
	// CharacterList são os personagens de uma pessoa. A cena não monta consulta:
	// ela faz a pergunta, e o hospedeiro sabe de qual tabela ela sai.
	CharacterList(ctx context.Context, ownerID int64) ([]sheet.CharacterDTO, error)
	// Catalogs é o motor primado, para o cartão saber a Defesa.
	//
	// Ele pode vir NULO, e isso não é descuido: sem catálogo primado o cartão
	// mostra travessão no lugar da Defesa e a cena continua de pé. Derrubar a
	// lista inteira por causa de um número seria pior — ver `HeroCardOf`.
	Catalogs() *engine.Catalogs
	// WritePage é a montagem da casca: ela injeta os estáticos e as
	// sobreposições que a cena não pode conhecer (ver `web/ui`).
	WritePage(w http.ResponseWriter, r *http.Request, status int, p ui.Page, corpo templ.Component)
}

// Scene é a cena montada com as dependências dela.
type Scene struct{ deps Deps }

func New(d Deps) Scene { return Scene{deps: d} }
