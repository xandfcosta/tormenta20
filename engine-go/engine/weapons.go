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
		}
		ex := expertiseBreakdown(ch, weaponSkillState(ch, skill, attribute), effects)
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
