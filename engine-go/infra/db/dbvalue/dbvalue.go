// Package dbvalue traduz entre o valor Go e a forma que a COLUNA guarda.
//
// Ele mora ao lado do `sqlcgen` e não dentro do `db` de propósito: o `db` abre o
// banco e por isso importa o goose e o driver do SQLite, e o `domain` chama o
// `NowISO` em dezenas de lugares. Um pacote-folha aqui é o que deixa a regra
// carimbar a hora sem arrastar um driver para dentro dela.
package dbvalue

import (
	"database/sql"
	"time"
)

// NullToPtr traz uma coluna anulável para um ponteiro (nulo vira `nil`).
func NullToPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullString e NullBool são o caminho de VOLTA do `NullToPtr`: um ponteiro que
// pode ser nulo virando a coluna anulável que o sqlc espera.
//
// A ida e a volta moram juntas porque separá-las é a forma que as faz
// divergirem.
//
// NullBool grava BOOLEANO em coluna INTEIRA, que é como o SQLite guarda um
// `trained`.
func NullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func NullBool(p *bool) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	var n int64
	if *p {
		n = 1
	}
	return sql.NullInt64{Int64: n, Valid: true}
}

// IsoLayout é o carimbo que as colunas de tempo guardam: texto, em UTC, com
// milissegundos. O SQLite não tem tipo de data, então o formato É o contrato —
// e ele ordena por comparação de string justamente por ser fixo.
const IsoLayout = "2006-01-02T15:04:05.000Z"

func NowISO() string { return IsoAt(time.Now()) }

func IsoAt(t time.Time) string { return t.UTC().Format(IsoLayout) }
