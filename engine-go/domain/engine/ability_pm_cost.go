package engine

import "strings"

// O CUSTO DE UMA HABILIDADE (ALE-647, decisão do dono, livro conferido).
//
// A ordem é a da magia (`SpellPmCostFor`): o custo, mais os aumentos, mais o MAIOR corte, e o piso de 1 PM por
// último. A p39 e a p45 põem a metade "após aplicar aprimoramentos e quaisquer outros efeitos que reduzam custo",
// e aprimoramento é aumento (p33); a p226 diz que "reduções no custo de PM não são cumulativas" e que "uma
// habilidade nunca pode ter seu custo reduzido para menos de 1 PM". O caminho não tomado — o piso nos cortes e o
// aumento por cima — fazia a habilidade de 1 PM com o –1 da Harmonizada custar 2 PM sob o Alquebrado; a decisão
// a deixa em 1.

// AbilityPmCost é o custo em PM de uma habilidade: `base` mais o `increase` (o Alquebrado) mais o maior dos
// `cuts` (cada um ≤ 0), com piso de 1 PM. O que custa zero continua de graça.
//
//	engine.AbilityPmCost(1, 1, -1) // 1: o +1 e o –1 se anulam antes do piso
func AbilityPmCost(base, increase int, cuts ...int) int {
	return AbilityPmCostWith(base, increase, Ratio{}, cuts...)
}

// AbilityPmCostWith é a conta única com um FATOR: a metade do Artista Completo (p45) vem "após aplicar
// aprimoramentos e quaisquer outros efeitos que reduzam custo", e o piso de 1 PM depois dela. Sem fator
// (`Ratio{}`), é o `AbilityPmCost`.
//
//	engine.AbilityPmCostWith(6, 1, engine.Ratio{Num: 1, Den: 2}) // 3: 6 + 1, e a metade
func AbilityPmCostWith(base, increase int, factor Ratio, cuts ...int) int {
	if base <= 0 {
		return 0
	}
	cut := 0
	for _, c := range cuts {
		cut = min(cut, c)
	}
	return max(1, factor.Applied(base+increase+cut))
}

// AbilityCostIncrease é o que sobe o custo de TODA habilidade da ficha, magias incluídas: o Alquebrado (p394).
func (e ItemEffects) AbilityCostIncrease() int {
	return StatFor(e, ModifierTarget{K: abilityPmCostTarget}).Total
}

// AbilityCostFactor é o fator no custo das habilidades da classe `class` (a metade do Artista Completo nas de
// bardo, sob a Inspiração, p45): o de toda habilidade e o da classe, e vale o que corta mais.
func (e ItemEffects) AbilityCostFactor(class string) Ratio {
	return severest(e.Factors[abilityPmCostTarget], e.Factors[abilityPmCostTarget+":"+class])
}

// AbilityCostRules são as regras do custo de habilidade de uma ficha: o aumento de toda habilidade (o
// Alquebrado, p394) e os fatores por classe (a metade do Artista Completo nas de bardo, p45; a chave vazia vale
// para toda habilidade). É o que a ficha leva para cada lugar que cobra ou mostra um custo.
type AbilityCostRules struct {
	Increase int
	Factors  map[string]Ratio
}

// AbilityCostRules lê as regras de custo dos efeitos da ficha.
func (e ItemEffects) AbilityCostRules() AbilityCostRules {
	out := AbilityCostRules{Increase: e.AbilityCostIncrease(), Factors: map[string]Ratio{}}
	for key, factor := range e.Factors {
		if key == abilityPmCostTarget {
			out.Factors[""] = factor
		} else if class, scoped := strings.CutPrefix(key, abilityPmCostTarget+":"); scoped {
			out.Factors[class] = factor
		}
	}
	return out
}

// Price é o custo de uma habilidade da classe `class` (vazia quando não é de classe), pela conta única.
//
//	rules.Price(2, "Bardo") // 1 sob a Inspiração do Artista Completo
func (c AbilityCostRules) Price(base int, class string, cuts ...int) int {
	return AbilityPmCostWith(base, c.Increase, severest(c.Factors[""], c.Factors[class]), cuts...)
}

// abilityPmCostTarget é o alvo do que mexe no custo de toda habilidade, e não só no das magias (`pmCost`).
const abilityPmCostTarget = "abilityPmCost"
