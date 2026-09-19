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
}

func NewPlays(db *sql.DB, q *sqlcgen.Queries, catalogs *engine.Catalogs) Plays {
	return Plays{db: db, queries: q, catalogs: catalogs}
}
