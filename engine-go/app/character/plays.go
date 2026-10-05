package character

import (
	"database/sql"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// Plays são os GESTOS de um herói que já existe: conjurar, beber uma dose,
// subir uma classe de nível e ligar um efeito.
//
// Cada um deles faz a mesma sequência, e é ela que define a camada: pergunta a
// DECISÃO à regra — do `domain/sheet` ou do catálogo do livro —, GRAVA, e
// devolve o que mudou. A cena recebe o resultado e redesenha; ela não sabe em
// que ordem nada disso acontece, nem quais tabelas foram tocadas.
//
// O `*sql.DB` está aqui por UM gesto só: beber uma dose escreve em três
// lugares e é uma transação. Os outros gravam pelo `queries` direto — e isso é
// visível de propósito, porque uma transação que não precisa existir é um
// bloqueio que ninguém pediu.
type Plays struct {
	db       *sql.DB
	queries  *sqlcgen.Queries
	catalogs *engine.Catalogs
	// rollDie é o DADO, injetado e não escolhido aqui — a mesma razão do
	// `combat.NewStrike`: é o que torna a regra testável, e em produção ele é o
	// `engine.RollDie`, com aleatoriedade criptográfica.
	//
	// Ele entrou quando a ficha passou a ROLAR (ALE-423). Antes dela a ficha só
	// recebia o que a mesa tinha rolado, e o comentário dos sinais ainda dizia
	// "a ficha não rola por ninguém" — deixou de ser verdade.
	rollDie func(faces int) (int, error)
}

func NewPlays(
	db *sql.DB, q *sqlcgen.Queries, catalogs *engine.Catalogs,
	rollDie func(faces int) (int, error),
) Plays {
	return Plays{db: db, queries: q, catalogs: catalogs, rollDie: rollDie}
}
