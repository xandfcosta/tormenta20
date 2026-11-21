package character

import (
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// scaledPrice é o PREÇO DO USO dos degraus pedidos (ver `book.UsePrice`), ou a recusa que diz por que não dá:
// acima do teto que o nível NA CLASSE abre (p40), ou sem PM para o preço. Poder que não vende degrau tem teto zero.
func scaledPrice(dto sheet.CharacterDTO, spec book.Activation, use book.Use) (int, error) {
	rules := sheet.AbilityCostRules(dto)
	if can, reason := book.ScaledDecision(spec, use, PowerMaxSteps(dto, spec), int(dto.ManaAvailable()), rules); !can {
		return 0, fmt.Errorf("%s: %s", spec.Name, reason)
	}
	return book.UsePrice(spec, rules, use), nil
}
