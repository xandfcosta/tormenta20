package master

import "t20engine/domain/book"

// A aba Perigos (p317-321): o filtro por tipo e os crachás do cartão. A ficha de cada tipo é do `hazardCard`.

func hazardKindOptions() []filterOption {
	outside := make([]filterOption, 0, len(book.HazardKinds))
	for _, k := range book.HazardKinds {
		outside = append(outside, filterOption{k.ID, k.Name})
	}
	return outside
}

// hazardBadges são o tipo e, na armadilha, o ND e a marca de mágica — o nome em
// itálico do livro, que diz que Dissipar Magia a anula (p317).
func hazardBadges(h book.Hazard) []string {
	badges := []string{book.HazardKindName(h.Kind)}
	if h.Trap != nil {
		badges = append(badges, "ND "+book.CRWritten(h.Trap.ND))
		if h.Trap.Magical {
			badges = append(badges, "Mágica")
		}
	}
	if h.Complex != nil {
		badges = append(badges, "ND "+book.CRWritten(h.Complex.ND))
	}
	return badges
}
