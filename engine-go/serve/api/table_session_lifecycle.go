package api

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O CICLO DE VIDA DA SESSÃO — iniciar, encerrar e reiniciar o combate.
//
// A regra fica FORA dos handlers HTTP, longe de `intParam`, `WriteJSON` e
// códigos de status: duas telas decidindo por conta própria o que "iniciar"
// significa é como nasce a divergência que ninguém nota — uma reabriria a sessão
// encerrada e a outra recusaria.
//
// O que mora aqui é a DECISÃO, não a resposta: quem traduz o resultado em JSON
// ou em remendo é cada tela.

// StartSession — "iniciar" significa TRÊS coisas conforme o estado, e é por
// isso que ela merece função própria:
//
//   - já ativa → não faz nada, e não é erro. Clicar duas vezes é o gesto de
//     quem não viu a tela mudar, e recusar seria punir a dúvida.
//   - encerrada → REABRE. A noite continuou, e obrigar a criar uma sessão nova
//     perderia a fila e o tabuleiro dela.
//   - planejada → começa do zero, carimbando o início.
func (tr tableRules) StartSession(ctx context.Context, sess sqlcgen.Session) (sqlcgen.Session, error) {
	if sess.Status == "active" {
		return sess, nil
	}
	agora := dbvalue.NowISO()
	if sess.Status == "ended" {
		return tr.queries.ReopenSession(ctx, sqlcgen.ReopenSessionParams{UpdatedAt: agora, ID: sess.ID})
	}
	return tr.queries.StartSessionFresh(ctx, sqlcgen.StartSessionFreshParams{
		StartedAt: sql.NullString{String: agora, Valid: true}, UpdatedAt: agora, ID: sess.ID,
	})
}

// EndSession recusa o que nunca começou.
//
// A recusa é deliberada e diferente do "já ativa" acima: encerrar uma sessão
// planejada não é um clique repetido, é um gesto sobre a coisa errada — e
// carimbar um fim numa noite que não teve início deixaria o histórico dizendo
// que ela aconteceu.
func (tr tableRules) EndSession(ctx context.Context, sess sqlcgen.Session) (sqlcgen.Session, error) {
	switch sess.Status {
	case "planned":
		return sess, fmt.Errorf("a sessão %d nunca foi iniciada; não há o que encerrar", sess.ID)
	case "ended":
		return sess, nil
	}
	agora := dbvalue.NowISO()
	return tr.queries.EndSession(ctx, sqlcgen.EndSessionParams{
		EndedAt: sql.NullString{String: agora, Valid: true}, UpdatedAt: agora, ID: sess.ID,
	})
}

// RestartCombat devolve a fila ao estado de quem acabou de abrir a sessão.
//
// É o gesto do combate que acabou e da mesa que vai começar outro na mesma
// noite — a sessão continua ao vivo, o que some é a ordem e os turnos. Não
// confundir com ENCERRAR, que tira a partida do ar.
func (tr tableRules) RestartCombat(ctx context.Context, sessionID int64) error {
	if err := tr.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: defaultRuntimeState, UpdatedAt: dbvalue.NowISO(), ID: sessionID,
	}); err != nil {
		return err
	}
	// ESQUECER O CACHE é metade do gesto, e a metade que não aparece: a fila ao
	// vivo mora em MEMÓRIA e o banco só é lido na primeira carga, então escrever
	// a linha limpa sem derrubar o cache deixa a sessão servindo a fila velha — o
	// reinício "funciona" e nada muda na tela.
	//
	// Um guarda que mede o BANCO não pega isto: ele já está vazio antes do
	// reinício, porque a fila nunca chegou lá.
	tr.sessions.Forget(sessionID)
	return nil
}
