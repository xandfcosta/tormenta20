package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// The non-item ActiveItem sources: race, origin, class powers, general powers,
// and the Tormenta Carisma penalty. Each reads the primed catalogs through the
// Catalogs receiver.

// raceActiveItems: the primary race always, plus
// any opted-in secondary. Attribute mods come from the persisted choices; the
// race's own attribute mods (fixed-race duplicates) are stripped.
func (r *Ruleset) raceActiveItems(ch Character) []ActiveItem {
	variantChoices := parseChoiceSet(ch.RaceAbilityChoices).has
	result := []ActiveItem{}
	for _, applied := range r.appliedRaces(ch) {
		race, choice := applied.race, applied.choice
		mods := r.raceAttributeMods(race.ID, choice)
		for _, m := range raceModifiers(race, variantChoices) {
			if m.Target.K != "attribute" {
				mods = append(mods, m)
			}
		}
		mods = append(mods, r.abilityChoiceModifiers(race, choice.abilityPicks)...)
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
	if size := r.raceSizeItem(ch); size != nil {
		result = append(result, *size)
	}
	return result
}

// raceAttributeMods: a race's attribute deltas (from its
// floating/ascendência choices) as `attribute` modifiers. Empty on incomplete
// choices, sem lançar: dado ruim vira escolha vazia.
func (r *Ruleset) raceAttributeMods(raceID string, choice raceAttrChoice) []Modifier {
	race := r.raceEntryByID(raceID)
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

// appliedRace é uma raça que conta na ficha, com a escolha dela: a primária
// sempre, a secundária só quando aplicada.
type appliedRace struct {
	race   *RaceDefinition
	choice raceAttrChoice
}

func (r *Ruleset) appliedRaces(ch Character) []appliedRace {
	primary := parseRaceAttributeChoices(ch.RaceAttributeChoices)
	secondaries := parseSecondaryRaceChoices(ch.SecondaryRaceChoices)
	out := []appliedRace{}
	for i, entry := range ch.Races {
		choice := primary
		if i != 0 {
			sec, ok := secondaries[entry.Race]
			if !ok {
				continue // non-applied secondary → no mechanics
			}
			choice = sec
		}
		if race := r.getRace(entry.Race); race != nil {
			out = append(out, appliedRace{race: race, choice: choice})
		}
	}
	return out
}

// abilityChoiceModifiers são os modificadores das habilidades que pedem
// escolha, um por perícia escolhida. O poder da Tormenta que a Deformidade
// troca por um dos bônus não é perícia, e fica de fora (`RaceHeldPowers`).
func (r *Ruleset) abilityChoiceModifiers(race *RaceDefinition, picks map[string][]string) []Modifier {
	out := []Modifier{}
	for _, ability := range race.Abilities {
		if ability.Choice == nil {
			continue
		}
		skills := []string{}
		for _, pick := range picks[ability.ID] {
			if !r.isTormentaPower(pick) {
				skills = append(skills, pick)
			}
		}
		out = append(out, expandChoiceModifiers(ability.Modifiers, skills, nil)...)
	}
	return out
}

// RaceHeldPowers são os poderes da Tormenta que a Deformidade trocou por um
// bônus (p24). A ficha os TEM, e eles não contam para a perda de Carisma.
//
//	r.RaceHeldPowers(lefou) // ["olhos-vermelhos"]
func (r *Ruleset) RaceHeldPowers(ch Character) []string {
	out := []string{}
	for _, applied := range r.appliedRaces(ch) {
		for _, ability := range applied.race.Abilities {
			for _, pick := range applied.choice.abilityPicks[ability.ID] {
				if ability.Choice != nil && r.isTormentaPower(pick) {
					out = append(out, pick)
				}
			}
		}
	}
	return out
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
	picks := parsePowerChoices(ch.PowerChoices)
	out := []ActiveItem{}
	for _, entry := range ch.Classes {
		owned := r.ownedClassPowers(entry.ClassName, entry.Level, chosen.has, choices[entry.ClassName])
		for _, power := range owned {
			mods := expandChoiceModifiers(power.Modifiers, picks[power.ID], power.choiceNames())
			if len(mods) == 0 {
				continue
			}
			for i := range mods {
				mods[i].classLevel = entry.Level
				mods[i] = growDice(mods[i])
			}
			out = append(out, ActiveItem{SourceID: power.ID, Source: power.Name, Equipped: &vestedWear, Modifiers: mods})
		}
	}
	return out
}

// expandChoiceModifiers troca cada modificador `fromChoice` por uma cópia por
// escolha, com a escolha no alvo (ALE-491).
//
// Duas escolhas são o poder tomado duas vezes (p38: "você pode escolher este
// poder várias vezes"), e por isso viram duas cópias e não uma com o dobro: o
// atributo de cada uma pode ser outro.
//
//	expandChoiceModifiers([+1 atributo ← escolha], ["strength", "wisdom"], nil)
//	// [+1 strength, +1 wisdom]
func expandChoiceModifiers(mods []Modifier, picks []string, names map[string]string) []Modifier {
	out := make([]Modifier, 0, len(mods))
	for _, m := range mods {
		if m.FromChoice == "" {
			out = append(out, m)
			continue
		}
		if m.FromChoice == "flag" {
			out = append(out, flagChoiceModifiers(m, picks, names)...)
			continue
		}
		for _, pick := range picks {
			copied := m
			copied.FromChoice = ""
			// `scope` é PREFIXO + escolha: a Especialização em Arma escreve
			// `weapon:` e a escolha completa com o id da arma (p65).
			if m.FromChoice == "scope" {
				copied.Target.Scope = m.Target.Scope + pick
			} else {
				copied.Target.Name = pick
			}
			out = append(out, copied)
		}
	}
	return out
}

// flagChoiceModifiers faz UM interruptor por escolha distinta: a escolha entra
// na flag e no rótulo, e cada repetição soma o `repeatBonus` (ALE-503).
//
// "Escolha outro tipo de terreno para receber o bônus ou aumente o bônus em um
// tipo de terreno já escolhido em +2" (Explorador, p50): floresta duas vezes é
// UMA floresta com +2, e não duas.
//
//	flagChoiceModifiers(+Sab na Defesa ← terreno, ["floresta", "floresta"], nomes)
//	// [+Sab+2 na Defesa, flag explorador:floresta, "no terreno: Floresta"]
func flagChoiceModifiers(m Modifier, picks []string, names map[string]string) []Modifier {
	order, times := []string{}, map[string]int{}
	for _, pick := range picks {
		if times[pick] == 0 {
			order = append(order, pick)
		}
		times[pick]++
	}
	out := make([]Modifier, 0, len(order))
	for _, pick := range order {
		copied := m
		copied.FromChoice = ""
		condition := *m.Condition
		condition.Flag = m.Condition.Flag + ":" + pick
		label := names[pick]
		if label == "" {
			label = pick
		}
		condition.Label = m.Condition.Label + label
		copied.Condition = &condition
		if m.Scale != nil {
			scale := *m.Scale
			scale.plus = scale.RepeatBonus * (times[pick] - 1)
			copied.Scale = &scale
		}
		out = append(out, copied)
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

// tormentaCarismaItem: the escalating Carisma loss over the count of real
// poderes da Tormenta picked.
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
	// O poder que a Deformidade trocou NÃO entra: "ele também não conta para
	// perda de Carisma" (p24, ALE-493).
	count := len(picked)
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

// growDice resolve o DADO QUE CRESCE com o nível na classe: o "1d6 de dano
// extra; a cada dois níveis, +1d6" do Ataque Furtivo (p73) é um modificador
// só, `dice: 1d6` com `scale: {per: levelStep, step: 2, round: up}`, e o número
// de dados é o de degraus — 1d6 no 1º nível, 2d6 no 3º, 10d6 no 19º.
//
// Na coleta, e não na ficha, porque o nível na classe já está aqui e o dado não
// depende de mais nada; o modificador sai sem `scale`, como se o catálogo
// tivesse escrito o número do nível.
func growDice(m Modifier) Modifier {
	if m.Dice == "" || m.Scale == nil || (m.Scale.Per != "level" && m.Scale.Per != "levelStep") {
		return m
	}
	count, faces, ok := strings.Cut(m.Dice, "d")
	n, err := strconv.Atoi(count)
	if !ok || err != nil {
		return m
	}
	m.Dice = strconv.Itoa(evalModifierScale(n, m.Scale, m.classLevel, nil)) + "d" + faces
	m.Scale = nil
	return m
}
