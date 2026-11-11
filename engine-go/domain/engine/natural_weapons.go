package engine

import (
	"slices"
	"strings"
)

// AS ARMAS NATURAIS (quadro de Ataques Desarmados e Armas Naturais): "armas leves corpo a
// corpo", que não precisam de mão e não são afetadas pelo que vale para armas empunhadas.
// Cada uma tem carta própria, depois do desarmado, para o índice das armas de mão não mudar.

// NaturalWeapon é a arma que o catálogo declara numa habilidade ou num poder.
type NaturalWeapon struct {
	Name              string `json:"name"`
	Damage            string `json:"damage"`
	CritMult          int    `json:"critMult"`
	DamageType        string `json:"damageType"`
	ExtraPM           int    `json:"extraPm"`
	ExtraAttacks      int    `json:"extraAttacks"`
	AwakenedByUse     bool   `json:"awakenedByUse"`
	StepsOnSameWeapon int    `json:"stepsOnSameWeapon"`
}

// NaturalStrike é o que a carta de uma arma natural dá no ataque extra: "uma vez por rodada,
// quando usa a ação agredir para atacar com outra arma" (p25). SourceID separa as fontes,
// porque cada uma tem o próprio limite.
type NaturalStrike struct {
	SourceID string
	ExtraPM  int
	// ExtraAttacks é quantos ataques o pagamento compra: um, ou um por pata (p137).
	ExtraAttacks int
}

// NaturalScope é o escopo do que vale SÓ nas armas naturais: a margem da Presas Afiadas (p63).
// O ataque desarmado não entra, porque o quadro os separa.
const NaturalScope = "natural"

type ownedNaturalWeapon struct {
	sourceID string
	weapon   NaturalWeapon
	steps    int
}

// naturalWeaponsOf são as armas naturais da ficha: da raça, inclusive a habilidade tomada de
// outra raça (p29), dos poderes da Tormenta e da devoção.
func (r *Ruleset) naturalWeaponsOf(ch Character, evenAsleep bool) []ownedNaturalWeapon {
	out := []ownedNaturalWeapon{}
	for _, applied := range r.appliedRaces(ch) {
		abilities := slices.Clone(applied.race.Abilities)
		for _, borrowed := range r.borrowedOf(applied.race, applied.choice.abilityPicks) {
			abilities = append(abilities, borrowed.ability)
		}
		for _, ability := range abilities {
			if ability.NaturalWeapon != nil {
				out = append(out, ownedNaturalWeapon{sourceID: ability.ID, weapon: *ability.NaturalWeapon})
			}
		}
	}
	for _, id := range append(r.chosenGeneralPowers(ch).list, r.originPickedPowerIds(ch)...) {
		if weapon, found := r.book.tormentaNaturalWeapons[id]; found {
			out = append(out, ownedNaturalWeapon{sourceID: id, weapon: weapon})
		}
	}
	for _, name := range parseChoiceSet(ch.GodPowers).list {
		power := r.grantedPowerByName(name)
		if power == nil || power.NaturalWeapon == nil {
			continue
		}
		if power.NaturalWeapon.AwakenedByUse && !evenAsleep && !usedThisScene(ch, power.ID) {
			continue
		}
		out = withGrantedWeapon(out, power.ID, *power.NaturalWeapon)
	}
	return out
}

// NaturalWeaponSources são as fontes das armas naturais que a ficha tem, mesmo a que ainda não
// foi ativada: é a escolha do Foco em Arma, que dura mais que a cena (p128).
func (r *Ruleset) NaturalWeaponSources(ch Character) []string {
	out := []string{}
	for _, owned := range r.naturalWeaponsOf(ch, true) {
		out = append(out, owned.sourceID)
	}
	return out
}

// withGrantedWeapon põe a arma do poder concedido na lista. "Se já possuir outro ataque natural
// de mordida, em vez disso, o dano desse ataque aumenta em dois passos" (p135).
func withGrantedWeapon(owned []ownedNaturalWeapon, sourceID string, weapon NaturalWeapon) []ownedNaturalWeapon {
	at := slices.IndexFunc(owned, func(o ownedNaturalWeapon) bool { return o.weapon.Name == weapon.Name })
	if weapon.StepsOnSameWeapon > 0 && at >= 0 {
		owned[at].steps += weapon.StepsOnSameWeapon
		return owned
	}
	return append(owned, ownedNaturalWeapon{sourceID: sourceID, weapon: weapon})
}

// usedThisScene diz se o poder concedido foi usado na cena. O uso é contado na ativação, que
// tem o deus na chave (`deus.kallyadranoch.presas-primordiais`).
func usedThisScene(ch Character, grantedID string) bool {
	return slices.ContainsFunc(ch.PowerUses, func(use PowerUse) bool {
		return use.Scope == "scene" && use.Used > 0 && strings.HasSuffix(use.PowerID, "."+grantedID)
	})
}

func (r *Ruleset) naturalCards(ch Character, b cardBasis) []WeaponCard {
	cards := []WeaponCard{}
	for _, owned := range r.naturalWeaponsOf(ch, false) {
		natural := owned.weapon
		w := &WeaponStats{
			Damage: natural.Damage, CritRange: 20, CritMult: natural.CritMult, Type: natural.DamageType,
			Hand: "light", Purpose: "melee",
		}
		// Como o desarmado, ela não é "uma arma em uma das mãos" (p128). O escopo da fonte é o
		// que o Foco em Arma escolhe.
		scopes := slices.DeleteFunc(weaponKindScopes(w), func(scope string) bool { return scope == MeleeOneHandedScope })
		scopes = append(scopes, NaturalScope, WeaponDamageScope(owned.sourceID))
		strike := NaturalStrike{SourceID: owned.sourceID, ExtraPM: natural.ExtraPM, ExtraAttacks: max(natural.ExtraAttacks, 1)}
		if owned.sourceID == ExtraLimbsPowerID {
			legs := r.legCards(ch, b, strike)
			cards = append(cards, legs...)
			if len(legs) >= LegsLimit {
				continue
			}
		}
		card := b.card(ch, natural.Name, w, scopes, owned.steps, false)
		card.Natural = &strike
		cards = append(cards, card)
	}
	return cards
}
