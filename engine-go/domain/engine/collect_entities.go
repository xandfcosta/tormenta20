package engine

import (
	"encoding/json"
	"fmt"
)

// The non-item ActiveItem sources: race, origin, class powers, general powers,
// and the Tormenta Carisma penalty. Each reads the primed catalogs through the
// Catalogs receiver.

// raceActiveItems: the primary race always, plus
// any opted-in secondary. Attribute mods come from the persisted choices; the
// race's own attribute mods (fixed-race duplicates) are stripped.
func (r *Ruleset) raceActiveItems(ch Character) []ActiveItem {
	variantChoices := parseChoiceSet(ch.RaceAbilityChoices).has
	primary := parseRaceAttributeChoices(ch.RaceAttributeChoices)
	secondaries := parseSecondaryRaceChoices(ch.SecondaryRaceChoices)
	result := []ActiveItem{}
	for i, entry := range ch.Races {
		choice := primary
		if i != 0 {
			sec, ok := secondaries[entry.Race]
			if !ok {
				continue // non-applied secondary → no mechanics
			}
			choice = sec
		}
		race := r.getRace(entry.Race)
		if race == nil {
			continue
		}
		mods := r.raceAttributeMods(entry.Race, choice)
		for _, m := range raceModifiers(race, variantChoices) {
			if m.Target.K != "attribute" {
				mods = append(mods, m)
			}
		}
		mods = append(mods, r.deformidadeModifiers(entry.Race, choice.deformity)...)
		if len(mods) == 0 {
			continue
		}
		result = append(result, ActiveItem{
			SourceID:  race.ID,
			Source:    "Raça: " + race.Name,
			Equipped:  &vestedWear,
			Modifiers: mods,
		})
	}
	return result
}

// raceAttributeMods: a race's attribute deltas (from its
// floating/ascendência choices) as `attribute` modifiers. Empty on incomplete
// choices, sem lançar: dado ruim vira escolha vazia.
func (r *Ruleset) raceAttributeMods(raceName string, choice raceAttrChoice) []Modifier {
	race := r.raceEntryByName(raceName)
	if race == nil {
		return []Modifier{}
	}
	deltas, err := resolveAttributeDeltas(race, choice.floatingPicks, choice.ancestry)
	if err != nil {
		return []Modifier{}
	}
	out := []Modifier{}
	for _, d := range deltas {
		if d.amount == 0 {
			continue
		}
		out = append(out, Modifier{
			Target:    ModifierTarget{K: "attribute", Name: d.attr},
			Amount:    d.amount,
			BonusType: "untyped",
			Note:      race.Name,
		})
	}
	return out
}

// deformidadeModifiers: Deformidade (Lefou p23) as +2 on each
// chosen perícia. The Carisma loss is emitted separately (tormentaCarismaItem).
func (r *Ruleset) deformidadeModifiers(raceName string, draft *deformidadeStored) []Modifier {
	if draft == nil || r.raceWithDeformidade(raceName) == "" {
		return []Modifier{}
	}
	out := []Modifier{}
	for _, n := range draft.expertises {
		if !expertiseNamesSet[n] {
			continue
		}
		out = append(out, Modifier{
			Target:    ModifierTarget{K: "expertise", Name: n},
			Amount:    deformidadeExpertiseBonus,
			BonusType: "untyped",
			Note:      "Deformidade",
		})
	}
	return out
}

// deformidadeHeldPower: the Deformidade-swapped poder da
// Tormenta, from either race blob.
func (r *Ruleset) deformidadeHeldPower(ch Character) string {
	if len(ch.Races) > 0 {
		primaryRace := ch.Races[0].Race
		primary := parseRaceAttributeChoices(ch.RaceAttributeChoices)
		if r.raceWithDeformidade(primaryRace) != "" && heldTormenta(primary.deformity) != "" {
			return heldTormenta(primary.deformity)
		}
	}
	for race, choice := range parseSecondaryRaceChoices(ch.SecondaryRaceChoices) {
		if r.raceWithDeformidade(race) != "" && heldTormenta(choice.deformity) != "" {
			return heldTormenta(choice.deformity)
		}
	}
	return ""
}

func heldTormenta(d *deformidadeStored) string {
	if d == nil {
		return ""
	}
	return d.tormentaPower
}

// originActiveItem: chosen origin benefits' modifiers, plus the
// modifiers of any concretely picked free-pick power. Nil when it grants nothing.
func (r *Ruleset) originActiveItem(ch Character) *ActiveItem {
	origin := r.getOrigin(ch.Origin)
	if origin == nil {
		return nil
	}
	choices := parseChoiceSet(ch.OriginChoices)
	mods := originModifiers(origin, choices.has, r.generalPowerByUid)
	for _, id := range r.originPickedPowerIds(ch) {
		if p := r.getGeneralPower(id); p != nil {
			mods = append(mods, p.Modifiers...)
		}
	}
	if len(mods) == 0 {
		return nil
	}
	return &ActiveItem{SourceID: origin.ID, Source: "Origem: " + origin.Name, Equipped: &vestedWear, Modifiers: mods}
}

// originPickedPowerIds: for each CHOSEN free-pick origin
// benefit, the power ids named in powerChoices. Iterates originChoices in its
// stored order so the resulting modifier order is stable.
func (r *Ruleset) originPickedPowerIds(ch Character) []string {
	chosen := parseChoiceSet(ch.OriginChoices)
	if len(chosen.list) == 0 {
		return []string{}
	}
	var blob map[string]json.RawMessage
	if err := json.Unmarshal([]byte(ch.PowerChoices), &blob); err != nil {
		return []string{}
	}
	out := []string{}
	for _, benefitID := range chosen.list {
		b := r.getOriginBenefit(benefitID)
		if b == nil || b.PowerPick == "" {
			continue
		}
		raw, ok := blob[benefitID]
		if !ok {
			continue
		}
		out = append(out, jsonStringArray(raw)...)
	}
	return out
}

// classActiveItems: one ActiveItem per owned class power that
// carries modifiers, named by the poder (not an opaque class bundle).
func (r *Ruleset) classActiveItems(ch Character) []ActiveItem {
	chosen := parseChoiceSet(ch.ClassPowers)
	choices := parseClassChoices(ch.ClassChoices)
	out := []ActiveItem{}
	for _, entry := range ch.Classes {
		owned := r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName])
		for _, power := range owned {
			if len(power.Modifiers) == 0 {
				continue
			}
			out = append(out, ActiveItem{SourceID: power.ID, Source: power.Name, Equipped: &vestedWear, Modifiers: power.Modifiers})
		}
	}
	return out
}

// generalPowerActiveItem: general powers (Poder de Combate…)
// stored in the classPowers blob by bare id. Iterates in stored order.
func (r *Ruleset) generalPowerActiveItem(ch Character) []ActiveItem {
	chosen := parseChoiceSet(ch.ClassPowers)
	out := []ActiveItem{}
	for _, id := range chosen.list {
		power := r.getGeneralPower(id)
		if power == nil || len(power.Modifiers) == 0 {
			continue
		}
		out = append(out, ActiveItem{SourceID: power.ID, Source: power.Name, Equipped: &vestedWear, Modifiers: power.Modifiers})
	}
	return out
}

// godPowerActiveItem é o poder concedido pelo deus de quem é devoto.
//
// A devoção NÃO era fonte de modificador (ALE-397). O `GodPower` só entrava
// pelo `vitalGrantMods`, que existe para PV e PM e descarta todo alvo que não
// seja um dos dois — então a Bênção do Mana funcionava, por ser `maxPm`, e as
// Escamas Dracônicas não mudavam a Defesa de ninguém.
//
// É UM poder e não uma lista: o livro dá um poder concedido ao se tornar
// devoto (p96), e o druida, que recebe dois, ainda não é modelado — a coluna
// `godPower` guarda um nome só.
func (r *Ruleset) godPowerActiveItem(ch Character) *ActiveItem {
	if ch.GodPower == "" {
		return nil
	}
	power := r.grantedPowerByName(ch.GodPower)
	if power == nil || len(power.Modifiers) == 0 {
		return nil
	}
	return &ActiveItem{
		SourceID:  power.ID,
		Source:    "Devoção: " + power.Name,
		Equipped:  &vestedWear,
		Modifiers: power.Modifiers,
	}
}

// tormentaCarismaItem: the escalating Carisma loss over the
// TOTAL count of real poderes da Tormenta (picked + the Deformidade-held one).
func (r *Ruleset) tormentaCarismaItem(ch Character) *ActiveItem {
	uniq := newOrderedSet()
	for _, id := range parseChoiceSet(ch.ClassPowers).list {
		uniq.add(id)
	}
	for _, id := range r.originPickedPowerIds(ch) {
		uniq.add(id)
	}
	picked := []string{}
	for _, id := range uniq.list {
		if r.isTormentaPower(id) {
			picked = append(picked, id)
		}
	}
	held := r.deformidadeHeldPower(ch)
	count := len(picked)
	if held != "" && !contains(picked, held) {
		count++
	}
	if count == 0 {
		return nil
	}
	return &ActiveItem{
		// Sem id de verbete: a fonte é o CONJUNTO de poderes da Tormenta, e não
		// um deles. O id fica vazio de propósito.
		Source:   "Poderes da Tormenta",
		Equipped: &vestedWear,
		Modifiers: []Modifier{{
			Target:    ModifierTarget{K: "attribute", Name: "charisma"},
			Amount:    -carismaLossFromPowers(count),
			BonusType: "untyped",
			Note:      fmt.Sprintf("%d poder(es) da Tormenta (p136)", count),
		}},
	}
}

// jsonStringArray decodes a JSON array, keeping only string elements — the
// bytes-level twin of parseStringArray (descarta o que não for string).
func jsonStringArray(raw json.RawMessage) []string {
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil
	}
	out := []string{}
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
