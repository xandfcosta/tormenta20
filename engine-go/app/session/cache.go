package session

import (
	"context"

	"t20engine/domain/live"
)

// A MESA SE LÊ DO BANCO, sempre (decisão do dono).
//
// Aqui morava uma cópia em memória do retrato gravado, um mapa por sessão, para a leitura não ir ao
// disco a cada pergunta. Ela saiu por duas razões medidas: o gancho de depois do commit a escrevia
// FORA da trava quando um gesto abria a unidade direto, e uma leitura ao mesmo tempo derrubava o
// processo (`concurrent map read and map write`, no e2e); e quem perguntava "esta linha tem ficha
// atrás?" a uma sessão que não estava em memória (servidor recém-subido) ouvia "não", e o dano de um
// personagem caía no rastreador como se ele fosse NPC.
//
// O preço é uma leitura do banco por pergunta. Ela foi medida contra a cópia, e o desenho da mesa
// passou a ler UMA vez em vez de três.

// State devolve a mesa desta sessão, lida do retrato gravado.
//
// ELA DEVOLVE ERRO: sem conseguir ler o banco não sabemos nada, e "não há ninguém na fila" é pior que
// recusar (ALE-373, decisão do dono).
func (st *Store) State(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	return st.snapshotsFor(ctx).Read(ctx, sessionID)
}

// SessionDeleted esquece a numeração de uma sessão que deixou de existir. É a única memória que
// sobrou: a `seq` dos quadros do hub, que não mora no retrato.
func (st *Store) SessionDeleted(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.seqs, sessionID)
}
