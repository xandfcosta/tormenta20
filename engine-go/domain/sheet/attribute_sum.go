package sheet

import (
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// AttributeSum escreve de onde vem o total de um atributo: o valor livre, e cada fonte com as
// parcelas iguais juntas, na ordem em que apareceram.
//
//	AttributeSum(1, []engine.SourceAmount{{Source: "Aumento de Atributo", Amount: 2}})
//	// "+1 livre +2 Aumento de Atributo"
func AttributeSum(base int, contributions []engine.SourceAmount) string {
	order := []string{}
	amounts := map[string]int{}
	for _, part := range contributions {
		if _, seen := amounts[part.Source]; !seen {
			order = append(order, part.Source)
		}
		amounts[part.Source] += part.Amount
	}
	parts := []string{book.WithSign(base) + " livre"}
	for _, source := range order {
		parts = append(parts, book.WithSign(amounts[source])+" "+source)
	}
	return strings.Join(parts, " ")
}
