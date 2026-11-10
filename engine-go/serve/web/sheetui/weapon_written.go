package sheetui

import (
	"slices"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// A CARTA DA ARMA ESCRITA: o dano e o crítico como a mesa os lê ("1d8+8 +1d8 de trevas · 19-20/x2"). Três
// telas escrevem a mesma carta — a aba Combate, a linha de ataque das Ações e a tira do impulso —, e a
// escrita mora num lugar só para as três não divergirem.

// damageLabel é "1d8+4": o dado da arma mais o que se soma a ele, e os dados extras com o tipo — "1d8+8
// +1d8 de trevas" na espada Tumular (p336). Sem os extras, a linha escondia o dado que o encanto põe em todo
// golpe.
func damageLabel(card engine.WeaponCard) string {
	label := card.Damage
	if card.DamageBonus != 0 {
		label += book.WithSign(card.DamageBonus)
	}
	// O MESMO DADO DO MESMO TIPO SOMA: o d8 de trevas do encanto e o do gasto da Tumular são "+2d8", como
	// o livro escreve (p336). A ordem é a da primeira aparição.
	type group struct {
		faces, kind string
		count       int
	}
	groups := []group{}
	for _, extra := range card.ExtraDamage {
		count, faces, ok := strings.Cut(extra.Dice, "d")
		n, err := strconv.Atoi(count)
		if !ok || err != nil {
			groups = append(groups, group{faces: extra.Dice, kind: extra.Type})
			continue
		}
		at := slices.IndexFunc(groups, func(g group) bool { return g.count > 0 && g.faces == faces && g.kind == extra.Type })
		if at < 0 {
			groups = append(groups, group{faces: faces, kind: extra.Type, count: n})
			continue
		}
		groups[at].count += n
	}
	for _, g := range groups {
		dice := g.faces
		if g.count > 0 {
			dice = strconv.Itoa(g.count) + "d" + g.faces
		}
		label += " +" + dice
		if g.kind != "" {
			label += " de " + g.kind
		}
	}
	return label
}

// critLabel é "19-20/x3". Margem 20 se escreve sozinha, e não como "20-20".
func critLabel(card engine.WeaponCard) string {
	margin := "20"
	if card.CritRange < 20 {
		margin = strconv.Itoa(card.CritRange) + "-20"
	}
	return margin + "/x" + strconv.Itoa(card.CritMult)
}
