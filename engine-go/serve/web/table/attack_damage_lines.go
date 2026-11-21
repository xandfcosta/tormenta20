package table

import "t20engine/domain/live"

// damageLineView é uma linha do passo do dano como a tela a escreve: a situação, a fórmula e o pedaço do caminho
// que a escolhe.
type damageLineView struct {
	Slug, Label, Source, ToRoll string
	Chosen                      bool
}

func damageLinesView(toRoll live.DamageToRoll) []damageLineView {
	out := make([]damageLineView, 0, len(toRoll.Lines))
	for _, line := range toRoll.Lines {
		slug := line.ID
		if slug == "" {
			slug = normalDamageLine
		}
		formula := toRollLine(live.DamageToRoll{Dice: line.Dice, Bonus: line.Bonus})
		out = append(out, damageLineView{Slug: slug, Label: line.Label, Source: line.Source, ToRoll: formula, Chosen: line.ID == toRoll.Line})
	}
	return out
}
