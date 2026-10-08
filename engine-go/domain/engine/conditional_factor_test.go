package engine

import "testing"

// O FATOR DE UM CONDICIONAL LIGADO MULTIPLICA (ALE-423).
//
// O `harvestFactors` pula todo termo ADIADO, e todo condicional é adiado: um
// fator sob `flagOn` nunca chegava ao mapa de fatores. O
// `ApplyActiveConditionals` então o perdia de vez, com um comentário dizendo
// que *"nenhum condicional traz fator hoje"* — a Torre Inabalável do cavaleiro
// (p55) trouxe, e o "você não pode se deslocar" dela não descia.
//
// O defeito é CALADO nos dois sentidos: a postura sobe, o interruptor aparece
// ligado na aba Efeitos, e o número não se mexe.
//
// UNITÁRIO porque isto é a REGRA de uma conversão — o que sobrevive à ida e
// volta entre a pilha e o opt-in —, e ela não precisa de banco nem de tela.

// stoppedByTheTower é o modificador da postura: deslocamento vezes zero.
func stoppedByTheTower() []Modifier {
	return []Modifier{{
		Target:    ModifierTarget{K: "displacement"},
		Factor:    &Ratio{Num: 0, Den: 1},
		BonusType: "untyped",
		Condition: &ModifierCondition{C: "flagOn", Flag: "postura-torre"},
	}}
}

func wornItemWith(mods []Modifier) []ActiveItem {
	wielded := "wielded"
	return []ActiveItem{{SourceID: "torre", Source: "Torre", Equipped: &wielded, Modifiers: mods}}
}

func TestTheFactorOfASwitchedOnConditionalMultiplies(t *testing.T) {
	base := ComputeItemEffects(wornItemWith(stoppedByTheTower()))
	// DESLIGADO não multiplica nada, e ele é o controle: sem esta metade, um
	// fator aplicado SEMPRE passaria no caso de baixo.
	if _, has := base.Factors[targetKey(ModifierTarget{K: "displacement"})]; has {
		t.Fatal("o fator desceu com a postura DESLIGADA")
	}
	if len(base.Conditional) != 1 {
		t.Fatalf("o condicional não saiu para o opt-in: %d", len(base.Conditional))
	}

	ligado := ApplyActiveConditionals(base, map[string]bool{FlagGroupID("postura-torre"): true})
	factor, has := ligado.Factors[targetKey(ModifierTarget{K: "displacement"})]
	if !has {
		t.Fatal("a postura ligada não trouxe o fator: o deslocamento não é zerado")
	}
	if got := factor.Applied(9); got != 0 {
		t.Errorf("o fator ligado tinha de zerar 9 quadrados, e deu %d", got)
	}
}

// E O FATOR INCONDICIONAL CONTINUA ATRAVESSANDO.
//
// Ele é o controle do conserto: a cópia do mapa de fatores que o opt-in agora
// faz poderia ter perdido os que já estavam lá — e aí a ficha com um
// condicional ligado andaria MAIS que a sem, que é o que o comentário antigo
// temia.
func TestTheUnconditionalFactorSurvivesTheOptIn(t *testing.T) {
	wielded := "wielded"
	base := ComputeItemEffects([]ActiveItem{{
		SourceID: "espada", Source: "Espada longa", Equipped: &wielded,
		Modifiers: []Modifier{
			{Target: ModifierTarget{K: "critRange"}, Factor: &Ratio{Num: 2, Den: 1},
				Condition: &ModifierCondition{C: "wielded"}},
			{Target: ModifierTarget{K: "attack"}, Amount: 1, BonusType: "untyped",
				Condition: &ModifierCondition{C: "flagOn", Flag: "qualquer"}},
		},
	}})
	// O `wielded` é INCONDICIONAL para o motor (ver `isUnconditional`), então
	// este fator é colhido na passada principal — o encanto Ameaçadora (p336)
	// é o caso real, e ele sempre funcionou.
	if _, has := base.Factors[targetKey(ModifierTarget{K: "critRange"})]; !has {
		t.Fatal("o fator de `wielded` não foi colhido: a bancada não mede o que vem medir")
	}
	ligado := ApplyActiveConditionals(base, map[string]bool{FlagGroupID("qualquer"): true})
	if _, has := ligado.Factors[targetKey(ModifierTarget{K: "critRange"})]; !has {
		t.Error("ligar um condicional perdeu o fator que já estava lá")
	}
}
