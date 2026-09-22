package session

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/domain/live"
	"t20engine/infra/db/sqlcgen"
)

// A UNIDADE DE TRABALHO: um gesto da mesa, uma transação (ALE-373).
//
// Passar a vez é UMA mudança lógica que atravessa dois donos de dado — a FILA
// (o retrato da sessão) e a FICHA (os efeitos que duravam a vez, o mana da
// manutenção). Cada um gravado por conta dá um gesto pela metade: a vez passa e
// o efeito fica ligado na ficha de alguém, em silêncio.
//
// # Por que a unidade CARREGA as portas em vez de o chamador abrir uma transação
//
// Porque a transação não é um pedaço de estado que se pendura no ar: ela é uma
// CONEXÃO, e quem escrever por outra bate na trava desta e espera o
// `busy_timeout` inteiro — medido na ALE-371, `database is locked (5)
// (SQLITE_BUSY)`. As portas que a unidade entrega já estão ligadas à transação
// dela, e é isso que torna impossível escrever por fora sem querer.

// Unit são as portas de um gesto, todas na mesma transação.
type Unit struct {
	Snapshots   SessionSnapshots
	Sheet       live.SheetVitals
	TurnEffects live.SheetTurnEffects
}

// Units abre a unidade de trabalho. Quem implementa é o hospedeiro, que é quem
// sabe montar os adaptadores ligados a uma transação.
//
// REENTRANTE por contrato: um `Do` dentro de outro reusa a unidade aberta em
// vez de abrir a segunda. Abrir a segunda é o impasse da ALE-371 com outro
// nome — ela pega outra conexão e espera a primeira, que espera o trabalho
// dela terminar.
//
// O erro do trabalho DESFAZ tudo: é isso que faz o gesto ser tudo-ou-nada.
type Units interface {
	Do(ctx context.Context, work func(Unit) error) error
}

// SheetPorts é como o HOSPEDEIRO monta as portas da ficha a partir de um
// caderno de consultas — o dele, ou o ligado a uma transação.
//
// Ela existe porque as duas metades desta fatia moram em lados diferentes: o
// ADAPTADOR da ficha é do hospedeiro, que conhece o `sheet` e o catálogo; o
// CONTORNO do gesto é daqui, e o `TestNoPresentationLayerOpensATransaction`
// cobra exatamente isso — quem decide o que é um gesto não é quem sabe o que é
// um `http.ResponseWriter`.
type SheetPorts func(*sqlcgen.Queries) (live.SheetVitals, live.SheetTurnEffects)

// NewUnits monta a unidade de trabalho sobre o banco deste app.
func NewUnits(database *sql.DB, q *sqlcgen.Queries, ports SheetPorts) Units {
	return storedUnits{db: database, q: q, ports: ports}
}

type storedUnits struct {
	db    *sql.DB
	q     *sqlcgen.Queries
	ports SheetPorts
}

// unitKey marca no contexto a unidade JÁ ABERTA. Chave de tipo próprio e não
// string: é o jeito de o valor não colidir com o de outro pacote.
type unitKey struct{}

// Do abre a transação, monta as portas ligadas a ela e desfaz tudo se o
// trabalho devolver erro.
//
// REENTRANTE quando o contexto carrega a unidade aberta. HOJE ESSA REDE NÃO É
// ACIONADA, e é honesto dizer: os gestos chamam com `context.Background()`,
// então nenhum contexto carrega unidade. O que protege agora é ESTRUTURAL — um
// gesto chama o `Do` uma vez e não chama método de store lá de dentro, e o
// `TestAGestureOpensExactlyOneUnit` prende isso.
func (u storedUnits) Do(ctx context.Context, work func(Unit) error) error {
	if open, ok := ctx.Value(unitKey{}).(Unit); ok {
		return work(open)
	}
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("abrir a transação do gesto: %w", err)
	}
	// O rollback é NO-OP depois do commit, e é o que garante que toda saída por
	// erro — inclusive um `panic` — feche a transação.
	defer func() { _ = tx.Rollback() }()

	inTx := u.q.WithTx(tx)
	sheetPort, turnPort := u.ports(inTx)
	unit := Unit{Snapshots: NewSnapshots(inTx), Sheet: sheetPort, TurnEffects: turnPort}
	if err := work(unit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fechar a transação do gesto: %w", err)
	}
	return nil
}

// WithUnit devolve um contexto que carrega a unidade aberta, para quem for
// chamado lá de dentro reusá-la em vez de abrir a segunda.
func WithUnit(ctx context.Context, unit Unit) context.Context {
	return context.WithValue(ctx, unitKey{}, unit)
}
