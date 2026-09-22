package session

import (
	"context"
	"errors"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// snapshotsDouble é o retrato de mentira: ele guarda uma mesa e pode recusar a
// gravação, que é o caso que importa aqui.
type snapshotsDouble struct {
	stored     *live.SessionRuntimeState
	refuse     error
	mutations  int
	readsAsked int
}

func (d *snapshotsDouble) Read(context.Context, int64) (*live.SessionRuntimeState, error) {
	d.readsAsked++
	return live.CloneState(d.stored), nil
}

func (d *snapshotsDouble) Mutate(_ context.Context, _ int64,
	change func(*live.SessionRuntimeState) error) (*live.SessionRuntimeState, error) {
	d.mutations++
	// A MUDANÇA RODA SOBRE UMA CÓPIA, como no retrato de verdade: o que ela
	// escreve só vira mesa depois de gravado.
	draft := live.CloneState(d.stored)
	if err := change(draft); err != nil {
		return nil, err
	}
	if d.refuse != nil {
		return nil, d.refuse
	}
	d.stored = draft
	return live.CloneState(draft), nil
}

// stateOf lê a mesa e FALHA ALTO quando a leitura é recusada.
func stateOf(t *testing.T, store *Store, sessionID int64) *live.SessionRuntimeState {
	t.Helper()
	state, err := store.State(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("ler a mesa da sessão %d: %v", sessionID, err)
	}
	return state
}

func storeWithDouble(d *snapshotsDouble) *Store {
	return NewStore(d, nil, func() string { return "id" }, nil, nil, &events.Bus{})
}

// A GRAVAÇÃO QUE FALHA RECUSA O COMANDO E NÃO DEIXA RASTRO NA MESA.
//
// O caso é de DECISÃO: o que se prende é o ramo em volta da gravação. Com a
// gravação saindo depois — como era antes da ALE-371 —, a linha entrava na fila
// que a mesa lê e o disco a recusava em silêncio.
func TestAWriteThatFailsRefusesTheCommandAndLeavesTheTableAsItWas(t *testing.T) {
	double := &snapshotsDouble{stored: live.EmptyRuntimeState()}
	store := storeWithDouble(double)

	// O CONTROLE: com o retrato aceitando, a mesma chamada entra na fila.
	if _, err := store.AddInitiativeEntry(context.Background(), 7, live.InitiativeEntry{Label: "Ogro", Type: "npc"}); err != nil {
		t.Fatalf("o controle falhou: %v", err)
	}
	if len(stateOf(t, store, 7).Initiative) != 1 {
		t.Fatalf("o controle falhou: o ogro não entrou na fila")
	}

	double.refuse = errors.New("o disco encheu")
	_, err := store.AddInitiativeEntry(context.Background(), 7, live.InitiativeEntry{Label: "Goblin", Type: "npc"})
	if err == nil {
		t.Fatal("a gravação falhou e o comando passou")
	}
	if !errors.Is(err, double.refuse) {
		t.Errorf("a recusa perdeu a causa pelo caminho: %v", err)
	}
	queue := stateOf(t, store, 7).Initiative
	if len(queue) != 1 || queue[0].Label != "Ogro" {
		t.Errorf("a fila que a mesa lê ficou %+v, e o goblin nunca foi gravado", queue)
	}
}

// A MUTAÇÃO PARTE DO QUE ESTÁ GRAVADO, e não do cache: quem responde "como está
// a mesa agora" para uma mudança é o retrato.
func TestEveryMutationStartsFromTheStoredTable(t *testing.T) {
	double := &snapshotsDouble{stored: live.EmptyRuntimeState()}
	store := storeWithDouble(double)

	for range 3 {
		if _, err := store.AddInitiativeEntry(context.Background(), 7, live.InitiativeEntry{Label: "NPC", Type: "npc"}); err != nil {
			t.Fatalf("pôr na fila: %v", err)
		}
	}
	if double.mutations != 3 {
		t.Errorf("três comandos deram %d gravações", double.mutations)
	}
	if len(double.stored.Initiative) != 3 {
		t.Errorf("o retrato gravado tem %d linhas, e os três comandos entraram nele",
			len(double.stored.Initiative))
	}
}
