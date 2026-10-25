package combat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"t20engine/app"
	"t20engine/domain/engine"
)

// O IMPULSO DE PM NO GOLPE (ALE-487). O que o motor soma é do
// `BoostedWeaponCards` e tem os casos do livro lá; aqui se prende o que ESTA
// camada decide: a carta do golpe é a impulsionada, o provisório leva o PM para
// a confirmação cobrar, e o PM que a ficha não tem recusa a proposta (decisão do
// dono: recusa na proposta, cobra no confirmar).

// boostedSword devolve a espada com +3 no ataque por 2 PM, para qualquer pedido.
func boostedSword(choice engine.BoostChoice) ([]engine.WeaponCard, int, error) {
	card := aLongsword()
	card.Attack += 3
	return []engine.WeaponCard{card}, 2, nil
}

func TestTheBoostedCardRollsAndTheProvisionalCarriesTheCost(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()},
			Boost: boostedSword, ManaAvailable: 5},
		"ogro": {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	out, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
	})
	if err != nil {
		t.Fatalf("propor com impulso: %v", err)
	}
	if out.Total != 10+5+3 {
		t.Errorf("total %d, want 18 — o golpe rola com a carta IMPULSIONADA (+3)", out.Total)
	}
	if out.BoostPM != 2 || out.BoostPowerID != "class.paladino.golpe-divino-1d8" {
		t.Errorf("o provisório saiu com impulso %q por %d PM, want o Golpe Divino por 2 PM", out.BoostPowerID, out.BoostPM)
	}
}

func TestABoostTheSheetCannotPayIsRefused(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()},
			Boost: boostedSword, ManaAvailable: 1},
		"ogro": {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
	})
	if !errors.Is(err, app.ErrRefused) || !strings.Contains(err.Error(), "2 PM") || !strings.Contains(err.Error(), "1") {
		t.Errorf("2 PM com 1 na ficha: err = %v, want recusa que diz os dois números", err)
	}
	if table.stored != nil {
		t.Errorf("a proposta recusada foi guardada na mesa")
	}
}

func TestACombatantWithoutBoostsCannotAskForOne(t *testing.T) {
	table := aTable(t)
	strike := NewStrike(sheetDouble{
		"heroi": {EntryID: "heroi", Label: "Arwen", Weapons: []engine.WeaponCard{aLongsword()}},
		"ogro":  {EntryID: "ogro", Label: "Ogro", Defense: 10},
	}, table, semTabuleiro{}, fixedDie(4))

	d20 := 10
	_, err := strike.Propose(context.Background(), app.Caller{ID: 7}, "player", Request{
		SessionID: 1, AttackerEntryID: "heroi", TargetEntryID: "ogro", D20: &d20, OwnsAttacker: true,
		Boost: engine.BoostChoice{PowerID: "class.paladino.golpe-divino-1d8"},
	})
	if !errors.Is(err, app.ErrRefused) {
		t.Errorf("impulso pedido por quem não tem impulso nenhum: err = %v, want recusa", err)
	}
}
