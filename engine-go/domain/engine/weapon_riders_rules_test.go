package engine

import "testing"

// O ENCANTO ATRAVESSA DO CATÁLOGO ATÉ A CARTA DA ARMA.
//
// Os casos do `attack_rules_test.go` prendem a ARITMÉTICA: dado extra não
// multiplica, bônus de crítico não entra no acerto normal. Este prende a
// COMPOSIÇÃO — que um modificador escrito no catálogo vira parcela na carta, e
// não some no caminho.
//
// É a metade que o oráculo não testemunha: nenhuma das 18 fichas empunha arma
// encantada, então o diff delas sai vazio por mais certo ou errado que o
// trajeto esteja.
func TestTheEnchantReachesTheWeaponCard(t *testing.T) {
	flamejante := Modifier{
		Target: ModifierTarget{K: "damage", DamageType: "fogo"},
		Dice:   "1d6", BonusType: "untyped",
	}
	dilacerante := Modifier{
		Target: ModifierTarget{K: "damage"}, Amount: 10, BonusType: "untyped",
		Condition: &ModifierCondition{C: "onCritical"},
	}
	empunhada := "wielded"
	efeitos := ComputeItemEffects([]ActiveItem{{
		SourceID: "espada", Source: "Espada longa", Equipped: &empunhada,
		Modifiers: []Modifier{flamejante, dilacerante},
	}})

	if len(efeitos.ExtraDamage) != 1 {
		t.Fatalf("esperava uma parcela extra e vieram %d", len(efeitos.ExtraDamage))
	}
	if got := efeitos.ExtraDamage[0]; got.Dice != "1d6" || got.Type != "fogo" {
		t.Errorf("a parcela veio %+v, e o catálogo escreveu 1d6 de fogo", got)
	}
	if got := efeitos.CriticalBonus[targetKey(ModifierTarget{K: "damage"})]; got != 10 {
		t.Errorf("o bônus de crítico veio %d e o catálogo escreveu 10", got)
	}

	// NENHUM DOS DOIS PODE TER VIRADO PARCELA DA PILHA. Se o dado extra somasse
	// como `amount`, ele entraria no dano de TODO ataque valendo zero — um
	// modificador que existe e não faz nada, que é a forma mais silenciosa de
	// defeito neste motor.
	if agg, achou := efeitos.ByTarget[targetKey(ModifierTarget{K: "damage"})]; achou {
		t.Errorf("o alvo `damage` recebeu %d na pilha, e as duas parcelas são da "+
			"CARTA: %+v", agg.Total, agg.Contributions)
	}

	// E NENHUM DOS DOIS PODE SER OFERECIDO COMO INTERRUPTOR: crítico não se
	// liga, e dado extra não é circunstância.
	if len(efeitos.Conditional) != 0 {
		t.Errorf("foram oferecidos %d interruptores, e nenhum dos dois é opt-in: %+v",
			len(efeitos.Conditional), efeitos.Conditional)
	}
}
