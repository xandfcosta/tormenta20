package api

import (
	"context"
	"strings"
	"testing"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O GESTO QUE MEXE EM NÚMERO DE QUEM O USA aplica o número: o PM comprou o efeito, e a conta é do app (o que
// o efeito faz nos OUTROS fica com o mestre).

// paladinWithCharisma é um paladino de `level` com Carisma 3, com o PM cheio e na mesa.
func paladinWithCharisma(t *testing.T, level int64) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id, err := f.s.sceneCore().Queries().CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: f.player, Name: "Devota", Origin: "batedor", Level: level,
		Strength: 3, Dexterity: 1, Constitution: 2, Intelligence: 0, Wisdom: 1, Charisma: 3,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassChoices: "{}", CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("semear a paladina: %v", err)
	}
	seedClasse(t, f.s, id, "Paladino", level)
	arrangePools(t, f.s, id, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.HpCurrent, pools.MpCurrent = pools.HpMax, pools.MpMax
		return pools, nil
	})
	seedMember(t, f.s, f.campaignID, id)
	return f, id
}

// A AURA SAGRADA soma o Carisma nas resistências de quem está nela (p84), e a paladina está nela.
func TestTheHolyAuraAddsCharismaToThePaladinsResistances(t *testing.T) {
	f, id := paladinWithCharisma(t, 3)
	before := statOf(t, f, id, engine.ModifierTarget{K: "resistance"})
	if refusal := powerCommand(t, f, id, "usa/class.paladino.aura-sagrada", ""); refusal != "" {
		t.Fatalf("usar a Aura Sagrada foi recusado: %q", refusal)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "resistance"}) - before; got != 3 {
		t.Errorf("as resistências subiram %d com a aura, e o Carisma é 3 (p84)", got)
	}
}

// O VINGADOR SAGRADO dá voo 18m e RD 20 até o fim da cena (p84).
func TestTheHolyAvengerGivesFlightAndDamageReduction(t *testing.T) {
	f, id := paladinWithCharisma(t, 20)
	reduction := statOf(t, f, id, engine.ModifierTarget{K: "damageReduction"})
	flight := statOf(t, f, id, engine.ModifierTarget{K: "flySpeed"})
	if refusal := powerCommand(t, f, id, "usa/class.paladino.vingador-sagrado", ""); refusal != "" {
		t.Fatalf("usar o Vingador Sagrado foi recusado: %q", refusal)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "damageReduction"}) - reduction; got != 20 {
		t.Errorf("a RD subiu %d na forma de vingador, e o livro dá 20 (p84)", got)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "flySpeed"}) - flight; got != 18 {
		t.Errorf("o voo subiu %d na forma de vingador, e o livro dá 18m (p84)", got)
	}
}

// A ARMADURA DE ALLIHANNA não tem limite de usos: o livro diz só "gastar uma ação de movimento e 1 PM" (p21).
func TestAllihannasArmorCanBeUsedAgainInTheScene(t *testing.T) {
	f, id := barbaro(t, 5)
	seedRaca(t, f.s, id, "Dahllan")
	for use := 1; use <= 2; use++ {
		if refusal := powerCommand(t, f, id, "usa/race.dahllan.armadura-de-allihanna", ""); refusal != "" {
			t.Fatalf("o %dº uso na cena foi recusado: %q", use, refusal)
		}
	}
}

// O DEGRAU DA PALAVRAS AFIADAS diz o que compra: cada +1 PM é a linha seguinte da tabela do nobre, +2d6 (p80).
func TestTheSharpWordsStepSaysItsDice(t *testing.T) {
	spec := book.ActivationOf("class.nobre.palavras-afiadas-2d6", "")
	if spec == nil || spec.Scaling == nil {
		t.Fatal("a Palavras Afiadas não tem o degrau")
	}
	if !strings.Contains(spec.Scaling.StepLabel, "+2d6") {
		t.Errorf("o degrau da Palavras Afiadas diz %q; want o dado que ele compra, +2d6", spec.Scaling.StepLabel)
	}
}
