package character

import (
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// scaledPrice é o PM de pagar os degraus pedidos, ou a recusa que diz por que
// não dá: acima do teto que o nível NA CLASSE abre (p40), ou sem PM para o total.
// Poder que não vende degrau tem teto zero.
func scaledPrice(dto sheet.CharacterDTO, spec book.Activation, steps int, onAlly bool) (int, error) {
	if onAlly && !spec.AllyTarget {
		return 0, fmt.Errorf("%s não tem um aliado como alvo", spec.Name)
	}
	max := PowerMaxSteps(dto, spec)
	// O CUSTO PELA CONTA ÚNICA (o Alquebrado, p394; o Artista Completo, p45; a Caridade com o aliado como alvo,
	// p83), e a falta de PM se confere contra o preço que se cobra.
	rules := sheet.AbilityCostRules(dto)
	class := book.ActivationClass(spec)
	cut := allyCutOf(rules, spec, onAlly)
	if can, reason := book.ScaledDecision(spec, steps, max, int(dto.ManaAvailable()), rules, cut); !can {
		return 0, fmt.Errorf("%s: %s", spec.Name, reason)
	}
	return rules.Price(book.ScaledCost(spec, steps), class, cut), nil
}

// allyCutOf é o corte deste uso quando o gesto diz que o alvo é um aliado (a Caridade, p83), ≤ 0; zero no uso
// comum.
func allyCutOf(rules engine.AbilityCostRules, spec book.Activation, onAlly bool) int {
	if !onAlly {
		return 0
	}
	return book.AllyTargetCut(spec, rules)
}
