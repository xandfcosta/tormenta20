package sheetui

import (
	"slices"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/serve/web/ds"
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

// critLabel é o crítico da carta: "19-20/x3".
func critLabel(card engine.WeaponCard) string {
	return critWritten(card.CritRange, card.CritMult)
}

// critWritten é a escrita do crítico em toda a ficha, a carta e a referência do livro:
// "19-20/x3". Margem 20 se escreve sozinha, e não como "20-20".
func critWritten(margin, multiplier int) string {
	from := "20"
	if margin < 20 {
		from = strconv.Itoa(margin) + "-20"
	}
	return from + "/x" + strconv.Itoa(multiplier)
}

// damageLinesOf são as linhas do dano da carta numa situação que só a mesa conhece — a Arma Improvisada e a Chave
// (p76) —, cada uma com o dano inteiro já um passo acima e o poder que a dá. São leitura: a linha que vale se
// escolhe no passo do dano da mesa (ALE-655).
func damageLinesOf(card engine.WeaponCard) []ds.SumTerm {
	var lines []ds.SumTerm
	for _, variant := range card.DamageVariants {
		varied, err := engine.WithDamageVariant(card, variant.ID)
		if err != nil {
			continue
		}
		note := variant.Source
		if variant.BookPage != 0 {
			note += " (p" + strconv.Itoa(variant.BookPage) + ")"
		}
		lines = append(lines, ds.SumTerm{Label: variant.Label, Note: note, Value: damageLabel(varied)})
	}
	return lines
}
