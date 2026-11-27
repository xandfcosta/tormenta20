package master

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
)

// A aba Perigos (p317-321): o filtro por tipo e a ficha que o cartão escreve.

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

// hazardSheet são as linhas da ficha, na ordem em que a mesa as usa: o que a
// armadilha causa e em quem, o teste da vítima, e as CDs para achar e desarmar;
// na doença, como ela pega e o que cada falha seguida faz. O complexo tem cartão próprio, na ordem da ficha
// dele (`hazardCard`).
func hazardSheet(h book.Hazard) []string {
	var lines []string
	if t := h.Trap; t != nil {
		if damage := t.DamageWritten(); damage != "" {
			lines = append(lines, "Dano: "+damage)
		}
		if t.Area != "" {
			lines = append(lines, "Área: "+t.Area)
		}
		if save := t.SaveWritten(); save != "" {
			lines = append(lines, "Resistência: "+save)
		}
		lines = append(lines, fmt.Sprintf("Investigação CD %d · Ladinagem CD %d", t.Investigation, t.Thievery))
	}
	if d := h.Disease; d != nil {
		lines = append(lines,
			fmt.Sprintf("%s · Fortitude CD %d", firstUpper(d.Contagion), d.DC),
			"Efeitos: "+strings.Join(d.Effects, " / "),
		)
	}
	return lines
}
