package engine

import "slices"

// UnskilledUsePenalty é o que custa usar a arma do jeito para o qual ela não foi feita:
// arremessar a que não tem alcance (p143), ou bater com a azagaia (p146).
const UnskilledUsePenalty = -5

// ThrowingEnchant é o encanto que faz a arma sem alcance voar sem a penalidade (p335).
const ThrowingEnchant = "encanto-arremesso"

// weaponOtherUse é o outro jeito de usar uma arma, com o que ele custa.
type weaponOtherUse struct {
	stats         WeaponStats
	suffix        string
	penaltySource string
}

// otherUseOf é o outro uso desta arma, ou nulo se ela só tem um.
//
// É do livro e vale para toda arma corpo a corpo: "armas com alcance podem ser
// usadas para ataques à distância", e "armas sem alcance podem ser arremessadas em
// alcance curto com –5" (p143). A azagaia é o caso inverso, e o único: "pode ser
// usada como arma corpo a corpo, mas você sofre uma penalidade de –5" (p146).
//
// O encanto Arremesso tira esse −5, embora a p335 não o cite: "pode ser arremessada em
// alcance curto" sem isso não daria ao ataque nada que toda arma já não tenha.
//
//	otherUseOf(lanca, nil).stats.Purpose // "thrown"
func otherUseOf(w *WeaponStats, enchants []string) *weaponOtherUse {
	use := weaponOtherUse{stats: *w}
	switch {
	case w.Purpose == "melee":
		use.stats.Purpose, use.suffix = "thrown", " (arremesso)"
		if w.Range == "" && !slices.Contains(enchants, ThrowingEnchant) {
			use.penaltySource = "arma sem alcance (p143)"
		}
	case w.Purpose == "thrown" && w.MeleeWithPenalty:
		use.stats.Purpose, use.suffix = "melee", " (corpo a corpo)"
		use.penaltySource = "azagaia corpo a corpo (p146)"
	default:
		return nil
	}
	return &use
}

// applied põe na carta o que este uso custa, com o nome na conta.
func (u *weaponOtherUse) applied(card WeaponCard) WeaponCard {
	if u.penaltySource == "" {
		return card
	}
	card.Improvised = true
	card.Attack += UnskilledUsePenalty
	card.AttackOwn.Total += UnskilledUsePenalty
	card.AttackOwn.Contributions = append(card.AttackOwn.Contributions,
		BreakdownContribution{Source: u.penaltySource, Amount: UnskilledUsePenalty})
	return card
}

// refusingRangedUses tira da carta o impulso que só vale corpo a corpo, uso por uso.
//
// O uso à distância volta sem o bônus e marcado, para o pedido ser recusado: deixá-lo
// passar cobraria o PM em troca de nada. O uso corpo a corpo fica com o bônus, mesmo
// quando ele é o outro uso, como bater com a azagaia.
func refusingRangedUses(boosted, plain WeaponCard) WeaponCard {
	other := boosted.OtherUse
	if other != nil && !other.IsMelee() {
		refused := *plain.OtherUse
		refused.RefusesBoost = true
		other = &refused
	}
	if !boosted.IsMelee() {
		boosted = plain
		boosted.RefusesBoost = true
	}
	boosted.OtherUse = other
	return boosted
}
