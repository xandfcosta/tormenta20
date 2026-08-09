package engine

import (
	"fmt"
	"strings"
)

// This file ports the top of derived.ts' collection layer: ActiveItemsFor (the
// entry point) and the item-level modifier assembly — catalog mods, overlays
// (melhorias/materiais), non-proficiency penalties (p142), weapon-attack
// mirroring, and the two opt-in homebrew rules. Together with collect_entities.go
// (race/origin/class/tormenta) it produces the []ActiveItem the resolution engine
// (ComputeItemEffects) consumes. See PORT-PLAN.md §2/§4 (slice 2).

// vestedWear is the shared 'vested' wear-state pointer used by every
// non-equipment ActiveItem (race/origin/class/effect). Read-only.
var vestedWear = "vested"

const deformidadePericiaBonus = 2 // deformidade.ts DEFORMIDADE_PERICIA_BONUS

// homebrewVestedOK mirrors items/homebrew.ts HOMEBREW_VESTED_OK — esotéricos
// that may be WORN and still grant their empunhado bonus (opt-in).
var homebrewVestedOK = map[string]bool{"medalhao-de-prata": true}

// expertiseNamesSet mirrors expertises.ts EXPERTISE_NAMES — the valid perícia
// names Deformidade may target.
var expertiseNamesSet = toSet([]string{
	"Acrobacia", "Adestramento", "Atletismo", "Atuação", "Cavalgar", "Conhecimento",
	"Cura", "Diplomacia", "Enganação", "Fortitude", "Furtividade", "Guerra",
	"Iniciativa", "Intimidação", "Intuição", "Investigação", "Jogatina", "Ladinagem",
	"Luta", "Misticismo", "Nobreza", "Ofício", "Percepção", "Pilotagem", "Pontaria",
	"Reflexos", "Religião", "Sobrevivência", "Vontade",
})

// ActiveItemsFor ports derived.ts activeItemsFor: collect every active modifier
// source into []ActiveItem — the input the resolution engine consumes. Order is
// preserved 1:1 with the TS so the parity dump compares byte-equal.
func (c *Catalogs) ActiveItemsFor(ch Character) []ActiveItem {
	proficiencies := parseProficiencySet(ch.Proficiencies)
	items := []ActiveItem{}
	for _, it := range ch.Items {
		if it.Equipped == nil {
			continue
		}
		items = append(items, c.itemActiveItem(it, proficiencies))
	}
	for _, eff := range ch.ActiveEffects {
		mods := parseEffectModifiers(eff.Modifiers)
		if len(mods) == 0 {
			continue
		}
		scope := "cena"
		if eff.Scope == "day" {
			scope = "dia"
		}
		items = append(items, ActiveItem{
			Source:    fmt.Sprintf("%s (%s)", c.effectSourceName(eff.CatalogID), scope),
			Equipped:  &vestedWear,
			Modifiers: mods,
		})
	}
	items = append(items, c.raceActiveItems(ch)...)
	if origin := c.originActiveItem(ch); origin != nil {
		items = append(items, *origin)
	}
	items = append(items, c.classActiveItems(ch)...)
	items = append(items, c.generalPowerActiveItem(ch)...)
	if tormenta := c.tormentaCarismaItem(ch); tormenta != nil {
		items = append(items, *tormenta)
	}
	if cond := conditionActiveItem(ch); cond != nil {
		items = append(items, *cond)
	}
	return items
}

// conditionActiveItem mirrors derived.ts `conditionActiveItem`: the p394 status
// conditions as a synthetic ActiveItem so their numeric penalties flow through
// the resolution engine (ALE-28). The modifier table below duplicates t20-data
// `CONDITION_MODIFIERS` byte-for-byte (Go can't import the TS catalog, like the
// rest of the engine). Appended last — same position as the TS collector.
func conditionActiveItem(ch Character) *ActiveItem {
	ids := parseStringArray(ch.ActiveConditions)
	mods := []Modifier{}
	for _, id := range ids {
		mods = append(mods, conditionModifiers(id)...)
	}
	if len(mods) == 0 {
		return nil
	}
	return &ActiveItem{Source: "Condições", Equipped: &vestedWear, Modifiers: mods}
}

func conditionModifiers(id string) []Modifier {
	return conditionModifierTable[id]
}

// condMod builds a status-condition modifier (bonusType "condition": book p394
// "aplique apenas o mais severo" → resolveStack keeps the worst per target).
func condMod(target ModifierTarget, amount int) Modifier {
	return Modifier{Target: target, Amount: amount, BonusType: "condition"}
}

func condDefense(n int) Modifier { return condMod(ModifierTarget{K: "defense"}, n) }
func condAllSkills(n int) Modifier {
	return condMod(ModifierTarget{K: "expertiseAll"}, n)
}
func condSkill(name string, n int) Modifier {
	return condMod(ModifierTarget{K: "expertise", Name: name}, n)
}
func condByAttr(attr string, n int) Modifier {
	return condMod(ModifierTarget{K: "expertiseByAttribute", Attribute: attr}, n)
}
func condAttack(n int) Modifier {
	return condMod(ModifierTarget{K: "attack", Scope: "all"}, n)
}
func condForDesCon(n int) []Modifier {
	return []Modifier{condByAttr("strength", n), condByAttr("dexterity", n), condByAttr("constitution", n)}
}
func condIntSabCar(n int) []Modifier {
	return []Modifier{condByAttr("intelligence", n), condByAttr("wisdom", n), condByAttr("charisma", n)}
}

// conditionModifierTable duplicates t20-data condition-modifiers.ts — keep in sync.
var conditionModifierTable = map[string][]Modifier{
	"abalado":      {condAllSkills(-2)},
	"apavorado":    {condAllSkills(-5)},
	"vulneravel":   {condDefense(-2)},
	"desprevenido": {condDefense(-5), condSkill("Reflexos", -5)},
	"indefeso":     {condDefense(-10)},
	"fraco":        condForDesCon(-2),
	"debilitado":   condForDesCon(-5),
	"frustrado":    condIntSabCar(-2),
	"esmorecido":   condIntSabCar(-5),
	"fatigado":     append(condForDesCon(-2), condDefense(-2)),
	"exausto":      append(condForDesCon(-5), condDefense(-2)),
	"cego":         {condDefense(-5), condByAttr("strength", -5), condByAttr("dexterity", -5)},
	"ofuscado":     {condAttack(-2), condSkill("Percepção", -2)},
	"fascinado":    {condSkill("Percepção", -5)},
	"surdo":        {condSkill("Iniciativa", -5)},
	"enredado":     {condDefense(-2), condAttack(-2)},
	"agarrado":     {condDefense(-5), condSkill("Reflexos", -5), condAttack(-2)},
	"caido":        {condSkill("Luta", -5)},
}

// itemActiveItem ports the per-item branch of activeItemsFor's map: base +
// overlay + material mods (ownMods), then penalties, mirrors, and homebrew, in
// the exact TS concatenation order.
func (c *Catalogs) itemActiveItem(it CharacterItem, prof map[string]bool) ActiveItem {
	var catalog *CatalogItem
	if it.CatalogID != nil {
		catalog = c.getCatalogItem(*it.CatalogID)
	}
	improvementIDs := parseImprovementIds(it.Improvements)

	ownMods := []Modifier{}
	if catalog != nil {
		ownMods = append(ownMods, catalog.Modifiers...)
	}
	for _, id := range improvementIDs {
		ownMods = append(ownMods, overlayModsWithProvenance(c.getCatalogItem(id))...)
	}
	if it.Material != nil {
		ownMods = append(ownMods, overlayModsWithProvenance(c.getCatalogItem(*it.Material))...)
	}

	mods := append([]Modifier{}, ownMods...)
	if catalog != nil {
		mods = append(mods, nonProficiencyPenalties(catalog, prof)...)
	}
	mods = append(mods, mirrorWeaponAttackMods(catalog, ownMods)...)
	mods = append(mods, equilibradaHomebrewMods(catalog, improvementIDs)...)
	mods = append(mods, vestedEsotericHomebrewMods(it.Equipped, catalog, ownMods)...)
	return ActiveItem{Source: it.Name, Equipped: it.Equipped, Modifiers: mods}
}

// overlayModsWithProvenance ports derived.ts: an overlay's modifiers with the
// overlay NAME folded into each note (so breakdown rows name the melhoria/material).
func overlayModsWithProvenance(overlay *CatalogItem) []Modifier {
	if overlay == nil {
		return []Modifier{}
	}
	out := make([]Modifier, 0, len(overlay.Modifiers))
	for _, m := range overlay.Modifiers {
		nm := m
		if !strings.Contains(m.Note, overlay.Name) {
			if m.Note != "" {
				nm.Note = overlay.Name + ": " + m.Note
			} else {
				nm.Note = overlay.Name
			}
		}
		out = append(out, nm)
	}
	return out
}

// mirrorWeaponAttackMods ports derived.ts: a weapon's own {attack,scope:this}
// mods mirrored onto its Luta/Pontaria perícia (T20 attacks are expertise tests).
func mirrorWeaponAttackMods(catalog *CatalogItem, ownMods []Modifier) []Modifier {
	if catalog == nil || catalog.Weapon == nil {
		return []Modifier{}
	}
	expertise := attackExpertiseFor(catalog.Weapon.Purpose)
	out := []Modifier{}
	for _, m := range ownMods {
		if m.Target.K != "attack" || m.Target.Scope != "this" {
			continue
		}
		condition := m.Condition
		if condition == nil {
			condition = &ModifierCondition{C: "wielded"}
		}
		note := m.Note
		if note == "" {
			note = "bônus desta arma"
		}
		out = append(out, Modifier{
			Target:    ModifierTarget{K: "expertise", Name: expertise},
			Amount:    m.Amount,
			BonusType: "untyped",
			Condition: condition,
			Note:      note,
		})
	}
	return out
}

// equilibradaHomebrewMods ports derived.ts: an opt-in +2 that nets out a
// desbalanceada weapon's -2 when the Equilibrada melhoria is attached.
func equilibradaHomebrewMods(catalog *CatalogItem, improvementIDs []string) []Modifier {
	if catalog == nil || catalog.Weapon == nil {
		return []Modifier{}
	}
	if !contains(catalog.Weapon.Traits, "desbalanceada") {
		return []Modifier{}
	}
	if !contains(improvementIDs, "melhoria-equilibrada") {
		return []Modifier{}
	}
	return []Modifier{{
		Target:    ModifierTarget{K: "expertise", Name: attackExpertiseFor(catalog.Weapon.Purpose)},
		Amount:    2,
		BonusType: "untyped",
		Condition: &ModifierCondition{C: "context", Note: "Homebrew: Equilibrada anula a desbalanceada (-2 → 0)"},
		Note:      "anula desbalanceada",
	}}
}

// vestedEsotericHomebrewMods ports derived.ts: a HOMEBREW_VESTED_OK esotérico
// worn (vested) keeps its wielded-gated bonuses behind one flagOn toggle.
func vestedEsotericHomebrewMods(equipped *string, catalog *CatalogItem, ownMods []Modifier) []Modifier {
	if equipped == nil || *equipped != "vested" {
		return []Modifier{}
	}
	if catalog == nil || !homebrewVestedOK[catalog.ID] {
		return []Modifier{}
	}
	out := []Modifier{}
	for _, m := range ownMods {
		if m.Condition == nil || m.Condition.C != "wielded" {
			continue
		}
		nm := m
		nm.Condition = &ModifierCondition{
			C:     "flagOn",
			Flag:  "homebrew-vestido-" + catalog.ID,
			Label: "Homebrew: esotérico vestido mantém o bônus (RAW exige empunhar, p159)",
		}
		out = append(out, nm)
	}
	return out
}

// nonProficiencyPenalties ports derived.ts: T20 p142 penalties for using a
// weapon/armor/shield without the required proficiency.
func nonProficiencyPenalties(catalog *CatalogItem, prof map[string]bool) []Modifier {
	required := requiredProficiency(catalog)
	if required == "" || prof[required] {
		return []Modifier{}
	}
	if strings.HasPrefix(catalog.Category, "weapon-") {
		return weaponNonProficiency(catalog)
	}
	basePenalty := -1
	for _, m := range catalog.Modifiers {
		if m.Target.K == "armorPenalty" {
			basePenalty = m.Amount
			break
		}
	}
	return []Modifier{
		{
			Target:    ModifierTarget{K: "flag", Name: "cannot-apply-dex-to-defense"},
			Amount:    1,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "vested"},
			Note:      "sem proficiência",
		},
		{
			Target:    ModifierTarget{K: "expertiseAll"},
			Amount:    basePenalty,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "vested"},
			Note:      catalog.Name + " sem proficiência",
		},
	}
}

func weaponNonProficiency(catalog *CatalogItem) []Modifier {
	purpose := ""
	if catalog.Weapon != nil {
		purpose = catalog.Weapon.Purpose
	}
	return []Modifier{
		{
			Target:    ModifierTarget{K: "attack", Scope: "this"},
			Amount:    -5,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "wielded"},
			Note:      "sem proficiência",
		},
		{
			Target:    ModifierTarget{K: "expertise", Name: attackExpertiseFor(purpose)},
			Amount:    -5,
			BonusType: "untyped",
			Condition: &ModifierCondition{C: "wielded"},
			Note:      "sem proficiência: -5 em testes de ataque (p142)",
		},
	}
}

// attackExpertiseFor maps a weapon purpose to its attack perícia (T20 resolves
// attacks as expertise tests): melee → Luta, everything else → Pontaria.
func attackExpertiseFor(purpose string) string {
	if purpose == "melee" {
		return "Luta"
	}
	return "Pontaria"
}

// effectSourceName ports the item branch of entities/character/effect-source.ts:
// an ActiveEffect's display name. Spell/activation catalogs aren't primed in the
// engine (no seed effect needs them), so this covers the manual pool + item
// sources and falls back to the raw id — matching the TS final default.
func (c *Catalogs) effectSourceName(catalogID string) string {
	if catalogID == "manual-temp-hp" {
		return "PV temporários (manual)"
	}
	if item := c.getCatalogItem(catalogID); item != nil {
		return item.Name
	}
	return catalogID
}
