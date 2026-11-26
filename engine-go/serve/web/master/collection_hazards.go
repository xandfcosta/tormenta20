package master

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
)

// A aba Perigos (p317-319): o filtro por tipo e a ficha que o cartão escreve.

func hazardKindOptions() []filterOption {
	outside := make([]filterOption, 0, len(book.HazardKinds))
	for _, k := range book.HazardKinds {
		outside = append(outside, filterOption{k.ID, k.Name})
	}
	return outside
}

func hazardMatches(h book.Hazard, key, value string) bool {
	if key == "tipo" {
		return h.Kind == value
	}
	return true
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
	return badges
}

// hazardSheet são as linhas da ficha, na ordem em que a mesa as usa: o que a
// armadilha causa e em quem, o teste da vítima, e as CDs para achar e desarmar;
// na doença, como ela pega e o que cada falha seguida faz.
func hazardSheet(h book.Hazard) []string {
	var lines []string
	if t := h.Trap; t != nil {
		if len(t.Damage) > 0 {
			parts := make([]string, len(t.Damage))
			for i, d := range t.Damage {
				parts[i] = strings.TrimSpace(d.Dice + " " + d.Kind)
			}
			lines = append(lines, "Dano: "+strings.Join(parts, " + "))
		}
		if t.Area != "" {
			lines = append(lines, "Área: "+t.Area)
		}
		if t.Save != nil {
			lines = append(lines, fmt.Sprintf("Resistência: %s CD %d %s", t.Save.Test, t.Save.DC, t.Save.Effect))
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
