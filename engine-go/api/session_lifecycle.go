package api

import (
	"context"
	"database/sql"
	"fmt"

	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
)

// O CICLO DE VIDA DA SESSÃO — iniciar, encerrar e reiniciar o combate.
//
// A regra vivia DENTRO dos handlers HTTP, misturada com `intParam`,
// `WriteJSON` e códigos de status. Ela saiu daqui para que a Mesa em Datastar a
// use sem reescrevê-la (ALE-269): duas telas decidindo por conta própria o que
// "iniciar" significa é como nasce a divergência que ninguém nota — a SPA
// reabriria uma sessão encerrada e a Mesa recusaria, ou o contrário.
//
// O que se extraiu é a DECISÃO, não a resposta: quem traduz o resultado em JSON
// ou em remendo continua sendo cada tela.

// StartSession — "iniciar" significa TRÊS coisas conforme o estado, e é por
// isso que ela merece função própria:
//
//   - já ativa → não faz nada, e não é erro. Clicar duas vezes é o gesto de
//     quem não viu a tela mudar, e recusar seria punir a dúvida.
//   - encerrada → REABRE. A noite continuou, e obrigar a criar uma sessão nova
//     perderia a fila e o tabuleiro dela.
//   - planejada → começa do zero, carimbando o início.
func (s *Server) StartSession(ctx context.Context, sess sqlcgen.Session) (sqlcgen.Session, error) {
	if sess.Status == "active" {
		return sess, nil
	}
	agora := plataforma.NowISO()
	if sess.Status == "ended" {
		return s.queries.ReopenSession(ctx, sqlcgen.ReopenSessionParams{UpdatedAt: agora, ID: sess.ID})
	}
	return s.queries.StartSessionFresh(ctx, sqlcgen.StartSessionFreshParams{
		StartedAt: sql.NullString{String: agora, Valid: true}, UpdatedAt: agora, ID: sess.ID,
	})
}

// EndSession recusa o que nunca começou.
//
// A recusa é deliberada e diferente do "já ativa" acima: encerrar uma sessão
// planejada não é um clique repetido, é um gesto sobre a coisa errada — e
// carimbar um fim numa noite que não teve início deixaria o histórico dizendo
// que ela aconteceu.
func (s *Server) EndSession(ctx context.Context, sess sqlcgen.Session) (sqlcgen.Session, error) {
	switch sess.Status {
	case "planned":
		return sess, fmt.Errorf("a sessão %d nunca foi iniciada; não há o que encerrar", sess.ID)
	case "ended":
		return sess, nil
	}
	agora := plataforma.NowISO()
	return s.queries.EndSession(ctx, sqlcgen.EndSessionParams{
		EndedAt: sql.NullString{String: agora, Valid: true}, UpdatedAt: agora, ID: sess.ID,
	})
}

// RestartCombat devolve a fila ao estado de quem acabou de abrir a sessão.
//
// É o gesto do combate que acabou e da mesa que vai começar outro na mesma
// noite — a sessão continua ao vivo, o que some é a ordem e os turnos. Não
// confundir com ENCERRAR, que tira a partida do ar.
func (s *Server) RestartCombat(ctx context.Context, sessionID int64) error {
	if err := s.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: defaultRuntimeState, UpdatedAt: plataforma.NowISO(), ID: sessionID,
	}); err != nil {
		return err
	}
	// ESQUECER O CACHE é metade do gesto, e a metade que não aparece: a fila ao
	// vivo mora em MEMÓRIA e o banco só é lido na primeira carga, então escrever
	// a linha limpa sem derrubar o cache deixa a sessão servindo a fila velha —
	// o reinício "funciona" e nada muda na tela.
	//
	// Esta linha estava no `handleClearTracker` e eu a PERDI ao extrair a regra.
	// O guarda que eu tinha escrito não pegou porque media o BANCO, que já
	// estava vazio antes do reset: a fila nunca tinha chegado lá. Foi a
	// sabotagem que denunciou os dois.
	s.sessions.Forget(sessionID)
	return nil
}
