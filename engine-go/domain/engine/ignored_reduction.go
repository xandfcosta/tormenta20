package engine

import (
	"maps"
	"slices"
)

// IGNORAR A REDUÇÃO DE DANO (ALE-652, fatia B): o Punhos de Adamante (lutador, p77) ignora 10 pontos de RD no
// desarmado.
//
// "Ignorar N pontos" é a RD do alvo valer N a menos ANTES de encontrar o dano: o alvo de RD 12 fica com RD 2, e
// absorve 2 de um golpe de 7. Tirar os N do que a RD JÁ absorveu erra justamente aí — o golpe de 7 seria todo
// absorvido pela RD 12, e 7 menos 10 não deixa nada para a RD 2 que sobrou.

// withoutIgnoredReduction devolve a RD do alvo com os pontos ignorados já tirados. Eles saem da GERAL primeiro, e
// o que sobrar sai das RDs por tipo, em ordem alfabética do tipo. A ordem é decisão desta casa: o livro fala da
// redução do alvo como uma só, e a geral é a que alcança qualquer golpe; a ordem alfabética só existe para o mesmo
// golpe dar sempre o mesmo dano.
//
//	geral, porTipo := withoutIgnoredReduction(AttackTarget{DamageReduction: 12}, 10) // 2, map[]
func withoutIgnoredReduction(target AttackTarget, ignored int) (int, map[string]int) {
	if ignored <= 0 {
		return target.DamageReduction, target.DamageReductionByType
	}
	general := max(0, target.DamageReduction-ignored)
	left := ignored - (target.DamageReduction - general)
	byType := make(map[string]int, len(target.DamageReductionByType))
	for _, tipo := range slices.Sorted(maps.Keys(target.DamageReductionByType)) {
		rd := target.DamageReductionByType[tipo]
		byType[tipo] = max(0, rd-left)
		left -= rd - byType[tipo]
	}
	return general, byType
}

// ignoredReductionOf é quanto de RD a carta ignora: o que vale em toda arma e o que vale nos escopos dela (o
// desarmado do Punhos de Adamante).
func ignoredReductionOf(effects ItemEffects, scopes []string) int {
	total := StatFor(effects, ModifierTarget{K: "ignoresDamageReduction", Scope: "all"}).Total
	for _, scope := range scopes {
		total += StatFor(effects, ModifierTarget{K: "ignoresDamageReduction", Scope: scope}).Total
	}
	return total
}
