package session

import (
	"context"
	"errors"
	"testing"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// turnEffectsDouble é a ficha vista pelo giro da vez: ela pode RECUSAR uma
// escrita, que é o caso que esta fatia prende.
type turnEffectsDouble struct {
	refuseExpire error
	expired      []int64
}

func (d *turnEffectsDouble) SustainedOf(context.Context, int64) ([]live.SustainedEffect, error) {
	return nil, nil
}

func (d *turnEffectsDouble) EndSustained(context.Context, int64, string) error { return nil }

func (d *turnEffectsDouble) ExpireTurnEffects(_ context.Context, charID int64) error {
	if d.refuseExpire != nil {
		return d.refuseExpire
	}
	d.expired = append(d.expired, charID)
	return nil
}

func (d *turnEffectsDouble) ConditionsOf(context.Context, int64) ([]string, error) {
	return nil, nil
}

func (d *turnEffectsDouble) ConstitutionOf(context.Context, int64) (int, error) { return 0, nil }

func (d *turnEffectsDouble) StabilizeBleeding(context.Context, int64) error { return nil }

// unitsDouble é a unidade de trabalho de mentira, e o que ela imita é o que
// importa: DESFAZER tudo quando o trabalho devolve erro.
type unitsDouble struct {
	snapshots *snapshotsDouble
	effects   *turnEffectsDouble
	opened    int
}

func (d *unitsDouble) Do(ctx context.Context, work func(Unit) error) error {
	d.opened++
	before := live.CloneState(d.snapshots.stored)
	if err := work(Unit{Snapshots: d.snapshots, TurnEffects: d.effects}); err != nil {
		d.snapshots.stored = before // o rollback da transação de verdade
		return err
	}
	return nil
}

func storeWithUnit(d *unitsDouble) *Store {
	return NewStore(d.snapshots, d, func() string { return "id" }, nil, d.effects, &events.Bus{})
}

// A VEZ NÃO PASSA QUANDO A FICHA NÃO PODE SER GRAVADA (ALE-373).
//
// Aqui o giro engolia o erro: expirava o que pudesse, passava a vez e gravava a
// fila. Quando a escrita da ficha falhava, a mesa avançava com o efeito ainda
// ligado na ficha de alguém — e ninguém ficava sabendo, porque o erro ia para o
// chão (ALE-372).
func TestTheTurnDoesNotPassWhenTheSheetWriteIsRefused(t *testing.T) {
	stored := live.EmptyRuntimeState()
	stored.Initiative = []live.InitiativeEntry{
		{ID: "a", Label: "Arcanista", Type: "character", CharacterID: live.PtrInt64(14)},
		{ID: "b", Label: "Goblin", Type: "npc"},
	}
	live.StartScene(stored, live.SceneAction)
	double := &unitsDouble{
		snapshots: &snapshotsDouble{stored: stored},
		effects:   &turnEffectsDouble{},
	}
	store := storeWithUnit(double)

	// O CONTROLE: com a ficha aceitando, a vez passa e o efeito expira.
	if _, err := store.NextTurn(7); err != nil {
		t.Fatalf("o controle falhou: %v", err)
	}
	if got := double.snapshots.stored.TurnIndex; got != 0 {
		t.Fatalf("o controle falhou: a vez tinha de ser a primeira, e veio %d", got)
	}
	if len(double.effects.expired) == 0 {
		t.Fatal("o controle falhou: nenhum efeito foi expirado")
	}

	double.effects.refuseExpire = errors.New("o disco encheu")
	_, err := store.NextTurn(7)
	if err == nil {
		t.Fatal("a ficha recusou a escrita e a vez passou mesmo assim")
	}
	if got := double.snapshots.stored.TurnIndex; got != 0 {
		t.Errorf("a vez avançou para %d sobre uma gravação que falhou", got)
	}
	if double.snapshots.stored.Round != 1 {
		t.Errorf("a rodada virou %d, e nada podia ter acontecido", double.snapshots.stored.Round)
	}
}

// UMA UNIDADE DENTRO DE OUTRA É O IMPASSE DA ALE-371 de volta: a segunda
// transação pega outra conexão e espera a primeira, que espera ela. O `Do` tem
// de REUSAR a unidade que já está aberta.
func TestAGestureOpensExactlyOneUnit(t *testing.T) {
	stored := live.EmptyRuntimeState()
	stored.Initiative = []live.InitiativeEntry{
		{ID: "a", Label: "Arcanista", Type: "character", CharacterID: live.PtrInt64(14)},
	}
	live.StartScene(stored, live.SceneAction)
	double := &unitsDouble{
		snapshots: &snapshotsDouble{stored: stored},
		effects:   &turnEffectsDouble{},
	}
	store := storeWithUnit(double)

	if _, err := store.NextTurn(7); err != nil {
		t.Fatalf("passar a vez: %v", err)
	}
	if double.opened != 1 {
		t.Errorf("o giro abriu %d unidades, e um gesto é UMA transação", double.opened)
	}
}
