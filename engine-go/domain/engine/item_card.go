package engine

import "slices"

// itemCard é a carta de uma arma do inventário empunhada em `slot`: numa mão, nas duas, ou numa
// pata insetoide (p137). O que a arma tem de seu — passo, encanto, melhoria, a outra forma de usar —
// vem com ela para onde quer que seja empunhada.
func (r *Ruleset) itemCard(ch Character, it CharacterItem, catalog *CatalogItem, slot string,
	effects ItemEffects, shared cardBasis) WeaponCard {
	w := weaponInTheGrip(catalog.Weapon, slot)
	weaponScope := WeaponDamageScope(*it.CatalogID)
	// O PASSO ANDA O DADO ANTES DE TUDO, e a ordem é da p231: o crítico
	// multiplica "os dados de dano do ataque (INCLUINDO quaisquer aumentos
	// por passos)". Andar depois multiplicaria o dado de antes.
	steps := DamageStepsOf(r.ownItemMods(it, catalog), &slot) +
		StatFor(effects, ModifierTarget{K: "damageStep", Scope: weaponScope}).Total
	nonLethal := StatFor(effects, ModifierTarget{K: "nonLethalDamage", Scope: weaponScope}).Total > 0
	if it.Oversized {
		steps += oversizedStep(r.raceSizeKey(ch))
	}
	cardOf := func(name string, use *WeaponStats) WeaponCard {
		scopes := append([]string{weaponScope, WieldedScope}, weaponKindScopes(use)...)
		if slot == LegSlot {
			// A pata não é "uma arma em uma das mãos" (p128).
			scopes = slices.DeleteFunc(scopes, func(scope string) bool { return scope == MeleeOneHandedScope })
		}
		if it.Oversized {
			scopes = append(scopes, OversizedWeaponScope)
		}
		card := shared.card(ch, name, use, scopes, steps, nonLethal)
		card.IgnoresSituations = r.situationsIgnoredBy(effects, weaponScope, it, catalog)
		card.CatalogID = *it.CatalogID
		card.ItemID, card.Doses = it.ID, it.Doses
		return card
	}
	card := cardOf(it.Name, w)
	if other := otherUseOf(w, it.Enchants); other != nil {
		use := other.applied(cardOf(it.Name+other.suffix, &other.stats))
		card.OtherUse = &use
	}
	return card
}
