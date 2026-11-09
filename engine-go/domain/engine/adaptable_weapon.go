package engine

import "strings"

// AdaptableTrait é a arma de uma mão que "pode ser usada com as duas mãos para aumentar seu dano em um
// passo" (p143).
const AdaptableTrait = "adaptavel"

// weaponInTheGrip é a arma como a MÃO a empunha. O catálogo escreve o dado da adaptável como
// "1d10/1d12", uma mão e duas, e nas duas ela é arma de duas mãos para os estilos (p128). Devolve a
// cópia: o verbete é de todo mundo.
//
// A ARMA DUPLA também escreve "1d6/1d6", uma ponta e a outra (p143). A carta rola uma ponta: a notação
// inteira não é um dado, e a mesa recusava o dano. As duas do livro têm as pontas iguais.
func weaponInTheGrip(w *WeaponStats, equipped string) *WeaponStats {
	first, second, split := strings.Cut(w.Damage, "/")
	if !split {
		return w
	}
	gripped := *w
	gripped.Damage = first
	if hasTrait(w.Traits, AdaptableTrait) && equipped == "wielded2" {
		gripped.Damage, gripped.Hand = second, "two"
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
