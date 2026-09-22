package session

import (
	"context"

	"t20engine/domain/live"
)

// A MESA VIVA: a cópia em memória do retrato gravado.
//
// Ela é CACHE, e isso passou a ser verdade na ALE-371 — antes era o contrário,
// a memória mandava e o banco corria atrás. Quem grava é a `SessionSnapshots`,
// dentro da mutação; esta cópia existe para a leitura não ir ao disco a cada
// pergunta, e jogá-la fora não perde nada.

// cachedLocked devolve a mesa em memória, lendo o retrato na primeira vez.
// Chamada sempre com `st.Mu` seguro.
//
// A leitura que FALHA não deixa nada em memória: o erro sobe, e a próxima
// pergunta tenta o banco de novo. Inventar uma mesa vazia aqui foi o defeito da
// ALE-369 — ela vira a resposta para sempre, e some com o combate gravado.
func (st *Store) cachedLocked(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	if s := st.States[sessionID]; s != nil {
		return s, nil
	}
	s, err := st.snapshots.Read(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	st.States[sessionID] = s
	return s, nil
}

// Load hidrata a mesa e devolve uma cópia. Ao contrário do `GetState`, ele
// devolve o erro.
func (st *Store) Load(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	s, err := st.cachedLocked(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return live.CloneState(s), nil
}

// Forget descarta a cópia em memória. É inofensivo por construção: o que vale
// está gravado, e a próxima pergunta relê.
func (st *Store) Forget(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.States, sessionID)
}

// SessionDeleted é o `Forget` de uma sessão que deixou de EXISTIR. Hoje os dois
// fazem a mesma coisa; os nomes ficam porque os CHAMADORES são diferentes — um
// é faxina, o outro é parte do gesto de apagar, e a ordem dele tem teste.
func (st *Store) SessionDeleted(sessionID int64) {
	st.Forget(sessionID)
}
