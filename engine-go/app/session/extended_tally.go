package session

import (
	"context"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// OS GESTOS DO TESTE ESTENDIDO (ver GLOSSARY, **teste estendido**): o mestre abre o contador, marca cada resultado,
// desfaz a última marca e encerra. A regra — o que soma, quando chega ao fim, o que se recusa — é do `live`.

// OpenExtendedTest abre o teste estendido; `each` dá uma contagem a cada linha da fila.
func (st *Store) OpenExtendedTest(ctx context.Context, sessionID int64, name string, required int, each bool) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.SessionChanged{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { return live.OpenExtendedTest(s, name, required, each) })
}

// MarkExtendedTest põe a marca do mestre na contagem da linha, ou na do grupo com o id vazio.
func (st *Store) MarkExtendedTest(ctx context.Context, sessionID int64, entryID, kind string) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.SessionChanged{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { return live.MarkExtendedTest(s, entryID, kind) })
}

// UndoExtendedMark tira a última marca do teste estendido.
func (st *Store) UndoExtendedMark(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.SessionChanged{SessionID: sessionID}, live.UndoExtendedMark)
}

// CloseExtendedTest encerra o teste estendido.
func (st *Store) CloseExtendedTest(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.SessionChanged{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error { live.CloseExtendedTest(s); return nil })
}
