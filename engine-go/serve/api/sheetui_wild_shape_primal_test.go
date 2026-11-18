package api

import (
	"strings"
	"testing"

	"t20engine/domain/engine"
)

// A FORMA PRIMAL (p62, druida 18): os benefícios de DOIS animais na Forma Selvagem. "Bônus iguais não se
// acumulam": cada número usa o maior das duas formas (decisão do dono). A arma natural e o tamanho o
// jogador escolhe, com o padrão na primeira forma (decisão do dono, mockup aprovado).

func primalDruid(t *testing.T, withPrimal bool) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Druida primal", "Druida", 18, 0, 0)
	powers := `["class.druida.forma-selvagem"]`
	if withPrimal {
		powers = `["class.druida.forma-selvagem","class.druida.forma-primal"]`
	}
	choiceCom(t, f, id, powers, `[]`)
	return f, id
}

func TestTheWildShapeWithPrimalFormTakesTheBestOfTwoAnimals(t *testing.T) {
	f, id := primalDruid(t, true)
	if page := powerScreen(t, f, id); !strings.Contains(page, "Segunda forma") {
		t.Error("quem tem Forma Primal não vê a escolha da segunda forma")
	}
	defense := statOf(t, f, id, engine.ModifierTarget{K: "defense"})
	strength := statOf(t, f, id, engine.ModifierTarget{K: "attribute", Name: "strength"})
	shape := `{"wild_form":"feroz","wild_second":"resistente","wild_tier":"normal","wild_damage":"corte","wild_move":"","wild_weapon_from":"resistente","wild_size_from":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", shape); refusal != "" {
		t.Fatalf("a transformação com duas formas foi recusada: %q", refusal)
	}
	// Feroz normal: Força +3, Defesa +2, 1d8. Resistente normal: Defesa +5, RD 5, 1d6 (p63).
	if got := statOf(t, f, id, engine.ModifierTarget{K: "defense"}) - defense; got != 5 {
		t.Errorf("a Defesa subiu %d; das duas formas vale a maior, +5 da Resistente, e não a soma", got)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "attribute", Name: "strength"}) - strength; got != 3 {
		t.Errorf("a Força subiu %d; a Feroz dá +3", got)
	}
	if got := statOf(t, f, id, engine.ModifierTarget{K: "damageReduction"}); got != 5 {
		t.Errorf("a RD ficou %d; a Resistente dá 5", got)
	}
	if combat := combatScreen(t, f, id); !strings.Contains(combat, "Forma Resistente") || !strings.Contains(combat, "1d6") {
		t.Error("a arma natural escolhida foi a da Resistente, e a carta não é ela")
	}
}

func TestTheSecondFormIsRefusedWithoutPrimalFormOrRepeated(t *testing.T) {
	f, id := primalDruid(t, false)
	two := `{"wild_form":"feroz","wild_second":"resistente","wild_tier":"normal","wild_damage":"corte","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", two); !strings.Contains(refusal, "Forma Primal") {
		t.Errorf("sem a Forma Primal, a segunda forma é recusada nomeando-a: %q", refusal)
	}
	if strings.Contains(powerScreen(t, f, id), "Segunda forma") {
		t.Error("quem não tem Forma Primal vê a escolha da segunda forma")
	}
	g, primal := primalDruid(t, true)
	same := `{"wild_form":"feroz","wild_second":"feroz","wild_tier":"normal","wild_damage":"corte","wild_move":""}`
	if refusal := powerCommand(t, g, primal, "forma-selvagem", same); !strings.Contains(refusal, "dois tipos") {
		t.Errorf("a mesma forma duas vezes é recusada: são dois tipos de animais (p62): %q", refusal)
	}
}
