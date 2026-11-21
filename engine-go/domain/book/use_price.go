package book

import "t20engine/domain/engine"

// Use é o que o gesto diz de UM uso: quantos degraus se pagam e se o alvo é um aliado.
type Use struct {
	Steps  int
	OnAlly bool
}

// UsePrice é o PREÇO DO USO: o PM que usar `spec` desta forma cobra da ficha. É o custo do catálogo com os
// degraus, pelas regras de custo da ficha (o Alquebrado, p394; a metade do Artista Completo, p45), com o corte da
// Caridade quando o alvo é um aliado e a habilidade pode tê-lo (p83), e o piso de 1 PM (ver `AbilityPmCostWith`).
//
// É o número que a decisão confere, que o gesto cobra e que a tela escreve. Refazê-lo por fora foi o que deixou a
// prévia dos diálogos dizendo um preço e o gesto cobrando outro.
//
//	book.UsePrice(coragem, regras, book.Use{OnAlly: true}) // 1 com a Caridade, e não os 2 do catálogo
func UsePrice(spec Activation, rules engine.AbilityCostRules, use Use) int {
	cut := 0
	if use.OnAlly {
		cut = allyTargetCut(spec, rules)
	}
	return rules.Price(ScaledCost(spec, use.Steps), activationClass(spec), cut)
}

// UsePrices é o preço do uso de zero a `maxSteps` degraus, na ordem: a prévia do diálogo lê o da contagem
// escolhida, e não refaz a conta no navegador.
//
//	book.UsePrices(cura, regras, 2, false) // [1 2 3]
func UsePrices(spec Activation, rules engine.AbilityCostRules, maxSteps int, onAlly bool) []int {
	prices := make([]int, 0, maxSteps+1)
	for steps := 0; steps <= maxSteps; steps++ {
		prices = append(prices, UsePrice(spec, rules, Use{Steps: steps, OnAlly: onAlly}))
	}
	return prices
}

// OffersAllyCut diz se a ficha corta o custo de `spec` quando o alvo é um aliado — o que faz a tela oferecer a
// escolha do alvo. Sem a Caridade, ou numa habilidade que não pode ter um aliado como alvo, não há o que escolher.
func OffersAllyCut(spec Activation, rules engine.AbilityCostRules) bool {
	return allyTargetCut(spec, rules) < 0
}

// allyTargetCut é o corte da ficha no uso de `spec` com um aliado como alvo, ≤ 0; zero na habilidade que não pode
// ter um aliado como alvo.
func allyTargetCut(spec Activation, rules engine.AbilityCostRules) int {
	if !spec.AllyTarget {
		return 0
	}
	return rules.AllyCut(activationClass(spec))
}
