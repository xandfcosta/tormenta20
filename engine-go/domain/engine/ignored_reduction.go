package engine

import (
	"fmt"
	"maps"
	"slices"
)

// IGNORAR A REDUÇÃO DE DANO (ALE-652, fatia B): o Punhos de Adamante (lutador, p77) ignora 10 pontos de RD no
// desarmado, e o Romper Resistências (guerreiro, p66) vende o mesmo no Ataque Especial, por 1 PM a mais.
//
// "Ignorar N pontos" é a RD do alvo valer N a menos ANTES de encontrar o dano: o alvo de RD 12 fica com RD 2, e
// absorve 2 de um golpe de 7. Tirar os N do que a RD JÁ absorveu erra justamente aí — o golpe de 7 seria todo
// absorvido pela RD 12, e 7 menos 10 não deixa nada para a RD 2 que sobrou.

// withoutIgnoredReduction devolve a RD do alvo com os pontos ignorados já tirados. Eles saem da GERAL primeiro, e
// o que sobrar sai das RDs por tipo, em ordem alfabética do tipo. A ordem é decisão desta casa: o livro fala da
// redução do alvo como uma só, e a geral é a que alcança qualquer golpe; a ordem alfabética só existe para o mesmo
// golpe dar sempre o mesmo dano.
//
//	geral, porTipo := withoutIgnoredReduction(AttackTarget{DamageReduction: 12}, 10) // 2, map[]
func withoutIgnoredReduction(target AttackTarget, ignored int) (int, map[string]int) {
	if ignored <= 0 {
		return target.DamageReduction, target.DamageReductionByType
	}
	general := max(0, target.DamageReduction-ignored)
	left := ignored - (target.DamageReduction - general)
	byType := make(map[string]int, len(target.DamageReductionByType))
	for _, tipo := range slices.Sorted(maps.Keys(target.DamageReductionByType)) {
		rd := target.DamageReductionByType[tipo]
		byType[tipo] = max(0, rd-left)
		left -= rd - byType[tipo]
	}
	return general, byType
}

// ignoredReductionOf é quanto de RD a carta ignora: o que vale em toda arma e o que vale nos escopos dela (o
// desarmado do Punhos de Adamante).
func ignoredReductionOf(effects ItemEffects, scopes []string) int {
	total := StatFor(effects, ModifierTarget{K: "ignoresDamageReduction", Scope: "all"}).Total
	for _, scope := range scopes {
		total += StatFor(effects, ModifierTarget{K: "ignoresDamageReduction", Scope: scope}).Total
	}
	return total
}

// ClassPowerBoostAddOn é o ACRÉSCIMO PAGO num impulso que outro poder vende: o Romper Resistências põe +1 PM
// no Ataque Especial (`Boost`) para ignorar 10 de RD. Ele só existe pago junto do impulso, e é por isso que mora
// no verbete do acréscimo e aponta para o impulso, e não o contrário: quem não tem o Romper não vê nada.
type ClassPowerBoostAddOn struct {
	Boost     string     `json:"boost"`
	PM        int        `json:"pm"`
	Modifiers []Modifier `json:"modifiers"`
}

// BoostAddOnOffer é o acréscimo que a tela oferece junto de um impulso: o poder, o nome, o PM a mais e o que ele
// compra, em frase ("ignora 10 de RD").
type BoostAddOnOffer struct {
	PowerID string
	Name    string
	PM      int
	Buys    string
}

// addOnBuys é o que o acréscimo compra, em frase. Só o ignorar a RD existe hoje; um acréscimo de outro alvo sai
// sem frase, e a tela fica com o nome e o preço.
func addOnBuys(addOn *ClassPowerBoostAddOn) string {
	for _, m := range addOn.Modifiers {
		if m.Target.K == "ignoresDamageReduction" {
			return fmt.Sprintf("ignora %d de RD", m.Amount)
		}
	}
	return ""
}

// ownedBoostAddOn é o acréscimo `id` que a ficha tem para o impulso `boost`, ou nil.
func (r *Ruleset) ownedBoostAddOn(ch Character, id, boost string) *ClassPower {
	for _, power := range r.ownedBoostAddOns(ch, boost) {
		if power.ID == id {
			return power
		}
	}
	return nil
}

// ownedBoostAddOns são os acréscimos que a ficha tem para o impulso `boost`, na ordem das classes.
func (r *Ruleset) ownedBoostAddOns(ch Character, boost string) []*ClassPower {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	var out []*ClassPower
	for _, entry := range ch.Classes {
		for _, power := range r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName]) {
			if power.BoostAddOn != nil && power.BoostAddOn.Boost == boost {
				out = append(out, power)
			}
		}
	}
	return out
}
