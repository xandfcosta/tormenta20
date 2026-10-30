package character

import (
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// UseChoice é o que o diálogo de usar escolheu: os degraus pagos e a caixa do
// Inimigo — a presa é do tipo escolhido, e os acréscimos dobram (p51).
type UseChoice struct {
	Steps   int
	Doubled bool
}

// RidersOf são os acréscimos que a ficha TEM para o uso de `spec` — o
// Espreitar, o Ponto Fraco e o Inimigo na Marca da Presa (ALE-509).
func RidersOf(dto sheet.CharacterDTO, spec book.Activation) []book.Activation {
	out, owned := []book.Activation{}, sheet.OwnedIndexOf(dto)
	for _, a := range book.Activations() {
		if a.Rider != nil && a.Rider.On == spec.ID && owned.Has(a) {
			out = append(out, a)
		}
	}
	return out
}
