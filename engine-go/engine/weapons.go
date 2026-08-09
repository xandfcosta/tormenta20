package engine

// WeaponCard is one wielded-weapon formula card (combat-magic-stats.tsx
// WeaponFormulaCards): attack = the weapon's attack perícia (Luta melee/thrown,
// Pontaria ranged) + global attack mods; damage adds Força for melee/thrown (not
// ranged) + global damage mods. Go owns the NUMBERS; the structural row labels
// (½ nível, FOR, Treino) and the crit string are applied on the front. Per-weapon
// mods (desbalanceada, Certeira…) already ride the Luta/Pontaria mirror inside
// effects, so the expertise breakdown carries them — no scope:'this' added here.
type WeaponCard struct {
	Name        string             `json:"name"`
	Skill       string             `json:"skill"`     // "Luta" | "Pontaria"
	Attribute   string             `json:"attribute"` // the skill perícia's attribute
	Attack      int                `json:"attack"`
	Expertise   ExpertiseBreakdown `json:"expertise"`
	AttackAll   TotalContribs      `json:"attackAll"`
	Damage      string             `json:"damage"`      // dice, e.g. "1d8"
	StrDamage   int                `json:"strDamage"`   // Força folded into melee/thrown damage (0 ranged)
	DamageBonus int                `json:"damageBonus"` // strDamage + damageAll.total
	DamageAll   TotalContribs      `json:"damageAll"`
	CritRange   int                `json:"critRange"`
	CritMult    int                `json:"critMult"`
}

// ComputeWeaponCards resolves the wielded-weapon cards for a raw Character under
// the given active conditionals (Fúria's global attack/damage mods land in
// attackAll/damageAll). Only catalog weapons in a hand slot, capped at two.
func (c *Catalogs) ComputeWeaponCards(ch Character, activeConditionals map[string]bool) []WeaponCard {
	effects := ApplyActiveConditionals(ComputeItemEffects(c.ActiveItemsFor(ch)), activeConditionals)
	attackAll := totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"})
	damageAll := totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"})
	forTotal := effectiveAttribute(ch, "strength", effects)
	dexTotal := effectiveAttribute(ch, "dexterity", effects)
	hasAcuidade := parseChoiceSet(ch.ClassPowers).has["acuidade-com-arma"]

	cards := []WeaponCard{}
	for _, it := range ch.Items {
		if it.Equipped == nil || (*it.Equipped != "wielded" && *it.Equipped != "wielded2") {
			continue
		}
		if it.CatalogID == nil {
			continue
		}
		catalog := c.getCatalogItem(*it.CatalogID)
		if catalog == nil || catalog.Weapon == nil {
			continue
		}
		w := catalog.Weapon
		skill, attribute := "Luta", "strength"
		strDamage := forTotal
		if w.Purpose == "ranged" {
			skill, attribute = "Pontaria", "dexterity"
			strDamage = 0
		} else {
			// Finesse (Adaga / Acuidade com Arma): use Destreza when it beats Força.
			dexAttack, dexDamage := weaponDexUse(w, hasAcuidade, forTotal, dexTotal)
			if dexAttack {
				attribute = "dexterity"
			}
			if dexDamage {
				strDamage = dexTotal
			}
		}
		// weaponSkillState returns the stored Luta row (attribute strength); force
		// the resolved attribute so a finessed melee attack sums Destreza (ALE-31).
		state := weaponSkillState(ch, skill, attribute)
		state.Attribute = attribute
		ex := expertiseBreakdown(ch, state, effects)
		cards = append(cards, WeaponCard{
			Name:        it.Name,
			Skill:       skill,
			Attribute:   ex.Attribute,
			Attack:      ex.Total + attackAll.Total,
			Expertise:   ex,
			AttackAll:   attackAll,
			Damage:      w.Damage,
			StrDamage:   strDamage,
			DamageBonus: strDamage + damageAll.Total,
			DamageAll:   damageAll,
			CritRange:   w.CritRange,
			CritMult:    w.CritMult,
		})
		if len(cards) == 2 {
			break
		}
	}
	return cards
}

// weaponDexUse mirrors weapon-cards.ts weaponDexUse: whether a wielded weapon may
// use Destreza instead of Força on attack/damage (T20 p145). Only when DES beats
// FOR (the rule is optional, so the sheet takes the better). Attack finesse = the
// weapon's inherent flag (Adaga) OR the Acuidade power on a light-melee/thrown/
// ágil weapon; damage finesse is Acuidade-only. Ranged never applies (ALE-31).
func weaponDexUse(w *WeaponStats, hasAcuidade bool, forTotal, dexTotal int) (attack, damage bool) {
	if w.Purpose == "ranged" || dexTotal <= forTotal {
		return false, false
	}
	acuidade := hasAcuidade &&
		((w.Hand == "light" && w.Purpose == "melee") || w.Purpose == "thrown" || hasTrait(w.Traits, "agil"))
	return w.Finesse || acuidade, acuidade
}

func hasTrait(traits []string, t string) bool {
	for _, x := range traits {
		if x == t {
			return true
		}
	}
	return false
}

// weaponSkillState mirrors expertise.ts expertiseStateFor: the stored Luta/Pontaria
// row if the character has one, else a default untrained state for the skill.
func weaponSkillState(ch Character, name, attribute string) CharacterExpertise {
	for _, ex := range ch.Expertises {
		if ex.Name == name {
			return ex
		}
	}
	return CharacterExpertise{Name: name, Attribute: attribute, Trained: false}
}
