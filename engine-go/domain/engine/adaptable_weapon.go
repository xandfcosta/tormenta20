package engine

import "strings"

// AdaptableTrait é a arma de uma mão que "pode ser usada com as duas mãos para aumentar seu dano em um
// passo" (p143).
const AdaptableTrait = "adaptavel"

// weaponInTheGrip é a arma como a MÃO a empunha. O catálogo escreve o dado da adaptável como
// "1d10/1d12", uma mão e duas, e nas duas ela é arma de duas mãos para os estilos (p128). Devolve a
// cópia: o verbete é de todo mundo.
func weaponInTheGrip(w *WeaponStats, equipped string) *WeaponStats {
	if !hasTrait(w.Traits, AdaptableTrait) {
		return w
	}
	gripped := *w
	oneHand, twoHands, split := strings.Cut(w.Damage, "/")
	if !split {
		return w
	}
	gripped.Damage = oneHand
	if equipped == "wielded2" {
		gripped.Damage, gripped.Hand = twoHands, "two"
	}
	return &gripped
}

// HoldsProficiently diz se a pessoa sabe usar o item como ele está na mão. A exótica adaptável "pode ser
// usada como uma arma marcial de duas mãos" (p147), então em duas mãos a marcial também serve.
func HoldsProficiently(item *CatalogItem, equipped *string, proficiencies map[string]bool) bool {
	required := requiredProficiency(item)
	if required == "" || proficiencies[required] {
		return true
	}
	return required == "armas-exoticas" && proficiencies["armas-marciais"] &&
		item.Weapon != nil && hasTrait(item.Weapon.Traits, AdaptableTrait) &&
		equipped != nil && *equipped == "wielded2"
}
