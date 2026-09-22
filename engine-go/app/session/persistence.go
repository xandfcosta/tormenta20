package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"t20engine/domain/live"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// A PONTE ENTRE A MEMÓRIA E O BANCO: ler a sessão na primeira vez, gravá-la a
// cada mutação e esquecê-la. Mora fora do `store.go` porque muda por outra
// razão — o resto do store é o que cada gesto faz com o estado.

// stateLocked devolve o estado da sessão, lendo-o do banco na PRIMEIRA vez.
// Chamada sempre com `st.Mu` seguro.
//
// É o ÚNICO caminho para o estado, e a razão é a ALE-369: aqui morava um
// `getOrCreateLocked` que CRIAVA um rastreador vazio para a sessão não
// carregada, e só o `Load` lia o banco. Depois de um reinício, o primeiro
// comando que chegasse antes de alguém abrir a Mesa — a aba do mestre que
// sobreviveu ao deploy — mutava uma fila vazia, e o `Load` seguinte devolvia a
// vazia para sempre; o `Persist` a gravava por cima do combate.
//
// A leitura que FALHA não deixa nada em memória: o erro sobe para quem muta, e
// um comando recusado é melhor que uma sessão apagada. Sem banco (`q` nulo, o
// store montado à mão em teste de regime) não há o que ler, e a sessão nasce
// vazia como sempre nasceu.
func (st *Store) stateLocked(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	if s := st.States[sessionID]; s != nil {
		return s, nil
	}
	if st.q == nil {
		s := live.EmptyRuntimeState()
		st.States[sessionID] = s
		return s, nil
	}
	sess, err := st.q.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ler do banco a mesa da sessão %d: %w", sessionID, err)
	}
	s := parseRuntimeBlob(sess.Runtimestate)
	st.States[sessionID] = s
	return s, nil
}

// Load hidrata a sessão de `Session.runtimeState` no primeiro acesso e depois
// serve a cópia em memória. Ao contrário do `GetState`, ele devolve o erro.
func (st *Store) Load(ctx context.Context, sessionID int64) (*live.SessionRuntimeState, error) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	s, err := st.stateLocked(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return live.CloneState(s), nil
}

// parseRuntimeBlob lê um blob gravado e cai para um rastreador vazio quando ele
// vem malformado. Os blobs são sempre completos (o nosso Marshal e o default da
// coluna carregam rodada e turno), então não há campo parcial a preencher.
func parseRuntimeBlob(blob string) *live.SessionRuntimeState {
	if blob == "" {
		return live.EmptyRuntimeState()
	}
	var parsed live.SessionRuntimeState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		return live.EmptyRuntimeState()
	}
	if parsed.Initiative == nil {
		parsed.Initiative = []live.InitiativeEntry{}
	}
	return &parsed
}

// SaveFailed diz se a última gravação do estado desta sessão falhou.
//
// ESTADO e não notícia, e a diferença é o que faz o aviso servir: ele vale
// enquanto durar, então quem abre a aba dez minutos depois da primeira falha
// merece vê-lo. Um evento perdido é um evento que não existiu. O irmão dele é o
// `BoardStore.SaveFailed`.
//
// Sob a trava porque o `Dirty` é escrito pelo `Persist`, que roda em goroutine.
func (st *Store) SaveFailed(sessionID int64) bool {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	return st.Dirty[sessionID]
}

// Persist serializa o estado atual em `Session.runtimeState`. Dispara e esquece:
// nunca devolve erro — devolve (Dirty, changed), com `changed` verdadeiro só
// quando a saúde da gravação VIROU desde o último `Persist`, para quem chama
// avisar a mesa exatamente nas transições. O store é o dono único da marca.
//
// Serializado para que gravações sobrepostas da mesma sessão cheguem em ordem: a
// última a rodar retrata o estado mais novo, e o banco converge para ele em vez
// de para uma captura velha.
func (st *Store) Persist(ctx context.Context, sessionID int64) (Dirty, changed bool) {
	pm := st.persistLock(sessionID)
	pm.Lock()
	defer pm.Unlock()

	st.Mu.Lock()
	s := st.States[sessionID]
	if s == nil {
		st.Mu.Unlock()
		return false, false
	}
	blob, _ := json.Marshal(live.CloneState(s))
	st.Mu.Unlock()

	err := st.q.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: string(blob), UpdatedAt: dbvalue.NowISO(), ID: sessionID,
	})

	st.Mu.Lock()
	defer st.Mu.Unlock()
	prev := st.Dirty[sessionID] // absent ⇒ false (healthy)
	Dirty = err != nil
	changed = prev != Dirty
	if Dirty {
		st.Dirty[sessionID] = true
		log.Printf("session %d: Persist failed (%v); marked Dirty for retry", sessionID, err)
	} else {
		delete(st.Dirty, sessionID)
	}
	return Dirty, changed
}

// Forget descarta o rastreador em memória de uma sessão. Ele NÃO limpa o
// `Dirty`: isso engoliria a recuperação suja→saudável — uma sessão deixada suja
// ainda precisa avisar `persistence-warning{Dirty:false}` no próximo `Persist`
// bem-sucedido, e o mapa se poda sozinho nesse sucesso.
func (st *Store) Forget(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.States, sessionID)
}

// SessionDeleted é o `Forget` de uma sessão que deixou de EXISTIR, e a diferença
// entre os dois é uma linha: esta apaga o `Dirty` também.
//
// O argumento do `Forget` acima — *"não limpa o Dirty: isso engoliria a
// recuperação suja→saudável"* — depende de haver um próximo `Persist` que avise
// que voltou a gravar. Com a sessão apagada não há: a marca ficaria acesa até o
// processo reiniciar, sobre uma mesa que ninguém quer gravar.
func (st *Store) SessionDeleted(sessionID int64) {
	st.Mu.Lock()
	defer st.Mu.Unlock()
	delete(st.States, sessionID)
	delete(st.Dirty, sessionID)
}
