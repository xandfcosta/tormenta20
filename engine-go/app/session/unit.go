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
	// InTx é o caderno de consultas LIGADO a esta transação, para o gesto que
	// precisa de uma porta que a unidade não carrega pronta.
	//
	// Ele existe por causa do TABULEIRO (ALE-376): o store dele guarda o
	// retrato no construtor, então pôr uma `BoardSnapshots` aqui não faria o
	// store usá-la — quem tem de receber a porta da transação é o STORE, e quem
	// a monta é o gesto. Entregar o caderno é o que permite isso sem esta
	// unidade conhecer o tabuleiro.
	//
	// O QUE ELE NÃO AFROUXA: tudo que se monta a partir dele continua preso à
	// transação, então a garantia do `Unit` — "não dá para escrever por fora
	// sem querer" — vale igual. O que ele permite é escrever por DENTRO de mais
	// lugares, que é justamente o pedido.
	InTx *sqlcgen.Queries
	// AfterCommit guarda o que só vale DEPOIS de o disco aceitar.
	//
	// Os dois stores mantêm uma cópia em MEMÓRIA do que gravaram, e ela não
	// participa da transação: o `rollback` desfaz o disco e deixa a memória
	// adiantada. Medido na ALE-376 — com a gravação da fila recusada por um
	// gatilho, a transação desfez tudo no banco e a peça continuou na casa nova
	// no mapa que a mesa vê. **Um gesto que desfaz metade é o que a unidade
	// existe para não ter.**
	//
	// Quem chama registra a instalação em memória aqui em vez de fazê-la na
	// hora; o `Do` roda as registradas depois do `Commit`, e não roda nenhuma se
	// ele falhar.
	AfterCommit func(install func())
}

// UnitFrom devolve a unidade que o contexto carrega, se houver.
//
// Ela existe para o store do TABULEIRO, que mora noutro pacote e precisa saber
// duas coisas da unidade aberta: por onde gravar e quando instalar na memória.
func UnitFrom(ctx context.Context) (Unit, bool) {
	open, ok := ctx.Value(unitKey{}).(Unit)
	return open, ok
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
// REENTRANTE quando o contexto carrega a unidade aberta, e desde a ALE-374 ele
// CARREGA: o `turnGesture` embrulha o contexto da requisição com `WithUnit`
// antes de descer, então um `Do` aninhado reusa em vez de pedir a segunda
// conexão do pool. Quem prova é o `TestANestedUnitReusesTheOpenTransaction`, e
// ele falha LENTO de propósito — sem a rede a espera é o `busy_timeout`
// inteiro, que é o sintoma que a pessoa vai ver.
//
// O `TestAGestureOpensExactlyOneUnit` prende a outra metade, e as duas são
// diferentes: um gesto abre UMA transação por desenho, e a rede é o que salva
// quem escrever o gesto que chama outro. Contar só com a rede seria trocar uma
// garantia por um resgate.
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
	var pending []func()
	unit := Unit{
		Snapshots: NewSnapshots(inTx), Sheet: sheetPort, TurnEffects: turnPort, InTx: inTx,
		AfterCommit: func(install func()) { pending = append(pending, install) },
	}
	if err := work(unit); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("fechar a transação do gesto: %w", err)
	}
	// DEPOIS do commit, e nunca antes: é o que faz a memória dos stores contar a
	// mesma história que o disco.
	for _, install := range pending {
		install()
	}
	return nil
}

// WithUnit devolve um contexto que carrega a unidade aberta, para quem for
// chamado lá de dentro reusá-la em vez de abrir a segunda.
func WithUnit(ctx context.Context, unit Unit) context.Context {
	return context.WithValue(ctx, unitKey{}, unit)
}
