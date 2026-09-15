package api

import (
	"context"
	"database/sql"
	"strings"
	"t20engine/infra/db/dbvalue"
)

// setBuilder acumula um UPDATE parcial: as colunas que o corpo do PATCH de fato
// trouxe, com os valores delas, em ordem.
//
// Ele existe para o `//nolint:gosec` ser afirmado UMA vez. A supressão promete
// que a cláusula SET é uma lista fechada de literais e não entrada do usuário, e
// essa promessa repetida em cada handler é uma promessa que o próximo handler
// quebra em silêncio.
//
// Nome de coluna vem de LITERAL no sítio de chamada, nunca do pedido — só os
// valores são ligados.
//
// @example
//
//	var set setBuilder
//	set.Add("name = ?", name)
//	if err := set.exec(ctx, s.db, "UPDATE campaigns", id); err != nil { … }
type setBuilder struct {
	columns []string
	args    []any
}

// Add registra a atribuição de uma coluna. `clause` tem de ser um literal como
// "name = ?".
func (b *setBuilder) Add(clause string, value any) {
	b.columns = append(b.columns, clause)
	b.args = append(b.args, value)
}

// empty diz que o PATCH não trouxe campo atualizável nenhum — o chamador
// responde 400 em vez de rodar um UPDATE que só mexe no `updatedAt`.
func (b *setBuilder) empty() bool { return len(b.columns) == 0 }

// exec roda `<prefix> SET <clauses> WHERE id = ?` como foi registrado, sem tocar
// em carimbo nenhum — `character_items` não tem coluna `updatedAt`.
func (b *setBuilder) exec(ctx context.Context, db *sql.DB, prefix string, id int64) error {
	//nolint:gosec // A cláusula SET é montada de literais de coluna nos sítios de
	// chamada — lista fechada. Só os valores são ligados.
	_, err := db.ExecContext(ctx,
		prefix+" SET "+strings.Join(b.columns, ", ")+" WHERE id = ?",
		append(append([]any{}, b.args...), id)...)
	return err
}

// execTouched é o `exec` mais o carimbo de `updatedAt` — a forma de toda tabela
// que TEM a coluna. É uma chamada separada para a diferença ficar visível onde
// ela importa, em vez de escondida atrás de um booleano.
func (b *setBuilder) execTouched(ctx context.Context, db *sql.DB, prefix string, id int64) error {
	stamped := setBuilder{
		columns: append(append([]string{}, b.columns...), "updatedAt = ?"),
		args:    append(append([]any{}, b.args...), dbvalue.NowISO()),
	}
	return stamped.exec(ctx, db, prefix, id)
}
