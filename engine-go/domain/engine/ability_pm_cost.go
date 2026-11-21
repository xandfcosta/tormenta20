package engine

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
	if base <= 0 {
		return 0
	}
	cut := 0
	for _, c := range cuts {
		cut = min(cut, c)
	}
	return max(1, base+increase+cut)
}

// AbilityCostIncrease é o que sobe o custo de TODA habilidade da ficha, magias incluídas: o Alquebrado (p394).
func (e ItemEffects) AbilityCostIncrease() int {
	return StatFor(e, ModifierTarget{K: abilityPmCostTarget}).Total
}

// abilityPmCostTarget é o alvo do que mexe no custo de toda habilidade, e não só no das magias (`pmCost`).
const abilityPmCostTarget = "abilityPmCost"
