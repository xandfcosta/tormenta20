package engine

import "fmt"

// Pure, catalog-free rules ported 1:1 from t20-data: race/origin modifier
// assembly (race-logic.ts / origin-logic.ts), atributo resolution
// (racas-attr.ts), item proficiency (item-classify.ts), Tormenta Carisma math
// (tormenta-carisma.ts), and class-power ownership (classes/ownership.ts). All
// operate on values passed in — no globals.

// raceModifiers ports abilities/race-logic.ts raceModifiers: attribute bonuses
// as `attribute` mods, then each ability's modifiers + the chosen variant's.
func raceModifiers(race *RaceDefinition, variantChoices map[string]bool) []Modifier {
	out := []Modifier{}
	for _, attr := range AttributeKeys {
		amount, ok := race.AttributeBonuses[attr]
		if !ok || amount == 0 {
			continue
		}
		out = append(out, Modifier{
			Target:    ModifierTarget{K: "attribute", Name: attr},
			Amount:    amount,
			BonusType: "untyped",
			Note:      race.Name,
		})
	}
	for _, ability := range race.Abilities {
		out = append(out, ability.Modifiers...)
		if len(ability.Variants) == 0 {
			continue
		}
		for _, v := range ability.Variants {
			if variantChoices[v.ID] {
				out = append(out, v.Modifiers...)
				break
			}
		}
	}
	return out
}

// originModifiers ports abilities/origin-logic.ts: sum the modifiers of the
// chosen benefits (benefits then poderUnico), matching the TS iteration order.
func originModifiers(origin *OriginDefinition, choiceSet map[string]bool) []Modifier {
	out := []Modifier{}
	all := make([]OriginBenefit, 0, len(origin.Benefits)+1)
	all = append(all, origin.Benefits...)
	all = append(all, origin.PoderUnico)
	for _, benefit := range all {
		if !choiceSet[benefit.ID] {
			continue
		}
		out = append(out, benefit.Modifiers...)
	}
	return out
}

// resolveAtributoDeltas ports racas-attr.ts resolveAtributoMod: resolve a raça's
// atributoMod into an ORDERED list of attribute deltas (the order
// raceAttributeMods emits them). Returns an error on invalid choices, mirroring
// the TS throws (the caller swallows them into no mods, like derived.ts'
// try/catch). Named apart from the MVP engine's map-returning resolveAtributoMod
// (races.go), which serves the flattened CharacterInput.
func resolveAtributoDeltas(raca *Raca, floatingPicks []string, ascendencia string) ([]attrDelta, error) {
	mod := raca.AtributoMod
	switch mod.Kind {
	case "fixed":
		return mod.Mods.pairs, nil
	case "floating":
		return resolveFloating(raca, mod, floatingPicks)
	default: // subraca-gated
		variant, ok := mod.Variants[ascendencia]
		if ascendencia == "" || !ok {
			return nil, fmt.Errorf(
				"resolveAtributoMod: %s requires a valid ascendência, got %q", raca.Name, ascendencia)
		}
		return variant.pairs, nil
	}
}

func resolveFloating(raca *Raca, mod AtributoMod, picks []string) ([]attrDelta, error) {
	if len(picks) != mod.Count {
		return nil, fmt.Errorf(
			"resolveAtributoMod: %s requires exactly %d floating picks, got %d",
			raca.Name, mod.Count, len(picks))
	}
	if hasDuplicates(picks) {
		return nil, fmt.Errorf("resolveAtributoMod: %s floating picks must be distinct", raca.Name)
	}
	if mod.Exclude != "" && contains(picks, mod.Exclude) {
		return nil, fmt.Errorf(
			"resolveAtributoMod: %s cannot place +%d in %s", raca.Name, mod.Value, mod.Exclude)
	}
	result := make([]attrDelta, 0, len(picks)+1)
	for _, a := range picks {
		result = append(result, attrDelta{attr: a, amount: mod.Value})
	}
	if mod.Penalty != nil {
		result = append(result, attrDelta{attr: mod.Penalty.Attribute, amount: mod.Penalty.Value})
	}
	return result, nil
}

// requiredProficiency ports items/catalog/item-classify.ts: the proficiency an
// item requires to use without penalty, or "" for none.
func requiredProficiency(item *CatalogItem) string {
	switch item.Category {
	case "weapon-simple":
		return "armas-simples"
	case "weapon-martial":
		return "armas-marciais"
	case "weapon-exotic":
		return "armas-exoticas"
	case "weapon-firearm":
		return "armas-de-fogo"
	case "armor-light":
		return "armaduras-leves"
	case "armor-heavy":
		return "armaduras-pesadas"
	case "shield":
		return "escudos"
	}
	return ""
}

// carismaLossFromPowers (tormenta-carisma.ts) already lives in tormenta.go —
// the collection layer reuses it.

// ownsClassPower ports classes/ownership.ts ownsClassPower: auto by level,
// elective by picked id, or grantedByChoice matching a classChoices value.
func ownsClassPower(
	power *ClassPower,
	classLevel int,
	chosen map[string]bool,
	choice ClassChoiceSelections,
) bool {
	if power.GrantedAtLevel != nil && *power.GrantedAtLevel <= classLevel {
		return true
	}
	if chosen[power.ID] {
		return true
	}
	if power.GrantedByChoice == nil {
		return false
	}
	return choice.value(power.GrantedByChoice.Field) == power.GrantedByChoice.Value
}

// ClassChoiceSelections mirrors classes/ownership.ts ClassChoiceSelections — one
// class's devoto/caminho picks.
type ClassChoiceSelections struct {
	Devoto  string `json:"devoto"`
	Caminho string `json:"caminho"`
}

func (s ClassChoiceSelections) value(field string) string {
	if field == "devoto" {
		return s.Devoto
	}
	if field == "caminho" {
		return s.Caminho
	}
	return ""
}
