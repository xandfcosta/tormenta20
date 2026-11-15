package engine

// OS ESCOPOS DE ARMA: onde cai um bônus que vale só para algumas armas — uma arma do catálogo, um tipo (corpo a
// corpo, arremesso, disparo), as armas de fogo, a Fúria, a marca "feita por mim". A carta soma o que alcança os
// escopos dela (`cardBasis.card`).

// WeaponDamageScope é o escopo de dano de UMA arma do catálogo — onde cai o
// +2 da Especialização em Arma escolhida (p65).
//
//	WeaponDamageScope("espada-longa") // "weapon:espada-longa"
func WeaponDamageScope(catalogID string) string { return "weapon:" + catalogID }

// Os escopos de dano por TIPO de arma, com os nomes que o livro dá a eles.
const (
	// WieldedScope: "1 arma empunhada" (Arma Mágica). A arma natural e o desarmado ficam de
	// fora, como diz o quadro de Ataques Desarmados e Armas Naturais.
	WieldedScope = "wielded"
	// MeleeLightOrAgileScope: "uma arma corpo a corpo leve ou ágil"
	// (Esgrimista, p48).
	MeleeLightOrAgileScope = "melee-light-or-agile"
	// RangedScope: "uma arma de ataque à distância" (Arqueiro, p50), que o
	// livro divide em arremesso e disparo (p142) — as duas entram.
	RangedScope = "ranged"
	// MeleeScope: todo ataque corpo a corpo (Ataque Poderoso, p124).
	MeleeScope = "melee"
	// MeleeOneHandedScope e MeleeTwoHandedScope: a arma corpo a corpo em uma das mãos (Estilo de Uma Arma,
	// p128) e nas duas (Estilo de Duas Mãos, p128).
	MeleeOneHandedScope = "melee-one-handed"
	MeleeTwoHandedScope = "melee-two-handed"
	// ThrownScope e FiredScope: as duas metades de "à distância" (p142).
	ThrownScope = "thrown"
	FiredScope  = "fired"
	// FirearmScope: as armas de fogo, pela categoria do item (o Pistoleiro, p48).
	FirearmScope = "firearm"
	// MadeByOwnerMeleeScope e MadeByOwnerFiredScope: a arma corpo a corpo e a de disparo com a marca "feita por
	// mim" — o Ferreiro e o Cano Raiado (p69). Um escopo por propósito, porque a carta soma escopo a escopo e
	// "feita por você E corpo a corpo" não é a soma dos dois.
	MadeByOwnerMeleeScope = "made-by-owner-melee"
	MadeByOwnerFiredScope = "made-by-owner-fired"
	// RageScope é o alcance da Fúria: "ataque e […] dano corpo a corpo" (p40), e o arremesso também com a Fúria
	// da Savana (p42). É um escopo PRÓPRIO, e não o `melee`, porque a Savana estende a Fúria e não todo bônus
	// corpo a corpo; e os degraus pagos e a Fúria Titânica, que copiam e multiplicam a Fúria, vão junto dela.
	RageScope = "rage"
)

// savannaRageFlag é a bandeira da Fúria da Savana, que leva a Fúria ao arremesso (p42).
const savannaRageFlag = "furia-da-savana"

// OversizedWeaponScope é o escopo da arma de categoria maior, e é por ele que a
// Empunhadura Poderosa diminui a penalidade para –2 (p125).
const (
	OversizedWeaponScope   = "oversized-weapon"
	OversizedWeaponPenalty = -5
)

// weaponKindScopes são os escopos de dano por TIPO em que esta arma cai.
//
// O tipo é o PROPÓSITO deste uso, um só: a machadinha corpo a corpo recebe o
// Esgrimista, e arremessada recebe o Estilo de Arremesso. É a mesma leitura que
// decide se a Força entra no dano.
func weaponKindScopes(w *WeaponStats) []string {
	switch w.Purpose {
	case "ranged":
		return []string{RangedScope, FiredScope}
	case "thrown":
		return []string{RangedScope, ThrownScope}
	case "melee":
		scopes := []string{MeleeScope, MeleeOneHandedScope}
		if w.Hand == "two" {
			scopes[1] = MeleeTwoHandedScope
		}
		if w.Hand == "light" || hasTrait(w.Traits, "agil") {
			scopes = append(scopes, MeleeLightOrAgileScope)
		}
		return scopes
	}
	return nil
}
