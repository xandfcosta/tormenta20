package engine

import "slices"

// O ATRIBUTO DA ARMA: a Força corpo a corpo e a Destreza à distância (p142), e quem pode trocá-los — a arma e a
// Acuidade com Arma, pela Destreza (p145); o Armeiro e a Balística, pela Inteligência (p68, p69). Todos dizem
// "pode", e a carta fica com o maior.

// Quem pôs a Destreza no ataque: a arma (`WeaponStats.Finesse`) ou o poder.
const (
	DexByWeapon  = "weapon"
	DexByFinesse = "acuidade-com-arma"
)

// weaponDexUse mirrors weapon-cards.ts weaponDexUse: whether a wielded weapon may
// use Destreza instead of Força on attack/damage (T20 p145). Only when DES beats
// FOR (the rule is optional, so the sheet takes the better). Attack finesse = the
// weapon's inherent flag (Adaga) OR the Acuidade power on a light-melee or ágil
// weapon; damage finesse is Acuidade-only. Ranged never applies (ALE-31), and the
// thrown weapon has its own branch in the card, because it already attacks with Destreza.
func weaponDexUse(w *WeaponStats, hasFinesse bool, forTotal, dexTotal int) (attack, damage bool) {
	if w.Purpose == "ranged" || dexTotal <= forTotal {
		return false, false
	}
	finesse := hasFinesse && ((w.Hand == "light" && w.Purpose == "melee") || hasTrait(w.Traits, "agil"))
	return w.Finesse || finesse, finesse
}

// Quem põe a Inteligência no lugar do atributo da arma, pelo nome que a ficha escreve.
const (
	armorerName    = "Armeiro"
	ballisticsName = "Balística"
	armorerFlag    = "armeiro"
	ballisticsFlag = "balistica"
)

// intelligenceInstead é o atributo do ataque depois do Armeiro, corpo a corpo, e da Balística, à distância:
// os dois "podem usar a Inteligência", e a carta fica com ela quando ela é maior que o atributo em uso
// (p68, p69). Só na arma EMPUNHADA ("quando empunha uma arma"): o desarmado e a arma natural ficam com o
// atributo deles. Devolve também quem a pôs, vazio quando ninguém pôs.
func (b cardBasis) intelligenceInstead(w *WeaponStats, attribute string, scopes []string) (string, string) {
	if !slices.Contains(scopes, WieldedScope) {
		return attribute, ""
	}
	current := b.forTotal
	if attribute == "dexterity" {
		current = b.dexTotal
	}
	by := ""
	switch {
	case w.Purpose == "melee" && b.effects.Flags[armorerFlag]:
		by = armorerName
	case (w.Purpose == "ranged" || w.Purpose == "thrown") && b.effects.Flags[ballisticsFlag]:
		by = ballisticsName
	}
	if by == "" || b.intTotal <= current {
		return attribute, ""
	}
	return "intelligence", by
}
