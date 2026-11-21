package engine

import "slices"

// O ATRIBUTO DA ARMA: a Força corpo a corpo e a Destreza à distância (p142), e quem pode trocá-los — a arma e a
// Acuidade com Arma, pela Destreza (p145); a licença `weaponAttribute` do catálogo, pela Inteligência (o Armeiro e
// a Balística, p68, p69). Todos dizem "pode", e a carta fica com o maior.

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

// licensedWeaponAttribute é o atributo de uma conta da arma — o ataque ou o dano (`account`) — depois das
// licenças `weaponAttribute` do catálogo: o Armeiro põe a Inteligência no ataque e no dano corpo a corpo, a
// Balística no ataque à distância (p68, p69). Os dois "podem", e a carta fica com o maior, pelo mesmo
// `licensedAttribute` das outras licenças. Só na arma EMPUNHADA ("quando empunha uma arma"): o desarmado e a arma
// natural ficam com o atributo deles. Devolve também a parcela da licença que trocou, vazia quando nada trocou —
// é a frase dela que a carta mostra.
func (b cardBasis) licensedWeaponAttribute(ch Character, w *WeaponStats, account, attribute string, scopes []string) (string, Contribution) {
	if !slices.Contains(scopes, WieldedScope) {
		return attribute, Contribution{}
	}
	// O arremesso é ataque À DISTÂNCIA (p142): a Balística o alcança.
	reach := "ranged"
	if w.Purpose == "melee" {
		reach = "melee"
	}
	return licensedAttribute(ch, attribute, b.effects, func(attribute string) ModifierTarget {
		return ModifierTarget{K: "weaponAttribute", Name: reach, Scope: account, Attribute: attribute}
	})
}
