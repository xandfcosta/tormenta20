package sheet

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/book"
)

// AS ESCOLHAS DAS HABILIDADES DE RAÇA (ALE-493): a Deformidade do lefou (p24) e
// o Vanguardista do kliren (p28).
//
// Elas espelham o `powerChoices` (decisão do dono): a habilidade declara a
// escolha no catálogo, e o que se escolheu mora no blob da raça —
// `raceAttributeChoices` da primária, a entrada da secundária em
// `secondaryRaceChoices` —, em `abilityPicks`, pelo id da habilidade. O blob
// guarda também a distribuição de atributo, e por isso toda escrita nele MESCLA
// a chave dela em vez de regravar o objeto.

const abilityPicksKey = "abilityPicks"

// tormentaSwap é o `swapFor` da Deformidade: um dos bônus vira um poder da
// Tormenta.
const tormentaSwap = "tormenta-power"

// RaceAbilityPicks são as escolhas de todas as raças que valem na ficha, pelo
// id da habilidade.
func RaceAbilityPicks(dto CharacterDTO) map[string][]string {
	out := map[string][]string{}
	for i, r := range dto.Races {
		blob, found := raceBlobOf(dto, i, r.Race)
		if !found {
			continue
		}
		for id, picks := range picksIn(blob) {
			out[id] = picks
		}
	}
	return out
}

// WithRaceAbilityPick liga ou desliga uma opção na habilidade: tocar o que já
// foi escolhido tira. Quem confere o resultado é o `WithChoicesValid`.
//
//	err := WithRaceAbilityPick(&dto, "lefou-deformidade", "Furtividade")
func WithRaceAbilityPick(dto *CharacterDTO, abilityID, option string) error {
	at, race := raceWithAbility(*dto, abilityID)
	if at < 0 {
		return fmt.Errorf("a habilidade %q não é de nenhuma raça desta ficha", abilityID)
	}
	blob, found := raceBlobOf(*dto, at, race)
	if !found {
		return fmt.Errorf("%s é raça secundária não aplicada: as habilidades dela não valem", book.RaceLabel(race))
	}
	picks := picksIn(blob)
	if i := slices.Index(picks[abilityID], option); i >= 0 {
		picks[abilityID] = slices.Delete(slices.Clone(picks[abilityID]), i, i+1)
	} else {
		picks[abilityID] = append(picks[abilityID], option)
	}
	if len(picks[abilityID]) == 0 {
		delete(picks, abilityID)
	}
	raw, err := json.Marshal(picks)
	if err != nil {
		return err
	}
	blob[abilityPicksKey] = raw
	return writeRaceBlob(dto, at, race, blob)
}

// WithRaceAttributeKey grava UMA chave no blob da raça primária, mantendo as
// outras: a distribuição de atributo e a ascendência moram com as escolhas das
// habilidades, e regravar o objeto as apagaria.
//
//	blob := WithRaceAttributeKey(dto.RaceAttributeChoices, "floatingPicks", picks)
func WithRaceAttributeKey(blob, key string, value any) (string, error) {
	object := objectOf(blob)
	raw, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	object[key] = raw
	out, err := json.Marshal(object)
	return string(out), err
}

// HasRaceAbilityPicks diz se algum blob de raça guarda escolha de habilidade —
// o atalho que poupa carregar a ficha nos gestos que podem invalidá-la.
func HasRaceAbilityPicks(raceAttributeChoices, secondaryRaceChoices string) bool {
	return strings.Contains(raceAttributeChoices, abilityPicksKey) ||
		strings.Contains(secondaryRaceChoices, abilityPicksKey)
}

// raceAbilityPicksFit confere as escolhas contra o catálogo e a ficha: só de
// habilidade que pede escolha, no máximo `count`, sem repetir; perícia do livro
// ou da ficha, Ofício que a ficha TEM, e no máximo um poder da Tormenta na
// troca — que não pode ser um poder que a ficha já escolheu.
func raceAbilityPicksFit(dto CharacterDTO) error {
	abilities := raceAbilitiesWithChoice(dto)
	for id, picks := range RaceAbilityPicks(dto) {
		ability, known := abilities[id]
		if !known {
			return fmt.Errorf("a habilidade %q não pede escolha, ou não é de uma raça desta ficha", id)
		}
		if err := abilityPicksFit(dto, ability, picks); err != nil {
			return err
		}
	}
	return nil
}

func abilityPicksFit(dto CharacterDTO, ability book.RaceAbility, picks []string) error {
	choice := *ability.Choice
	if len(picks) > choice.Count {
		return fmt.Errorf("%s escolhe %d, e ficariam %d: tire uma antes", ability.Name, choice.Count, len(picks))
	}
	swaps := 0
	for i, pick := range picks {
		if slices.Contains(picks[:i], pick) {
			return fmt.Errorf("%s escolheu %s duas vezes", ability.Name, pick)
		}
		if choice.SwapFor == tormentaSwap && isTormentaPower(pick) {
			swaps++
			if slices.Contains(UnmarshalStrings(dto.ClassPowers), pick) {
				return fmt.Errorf("%s: %s já é poder escolhido desta ficha", ability.Name, book.GeneralPowers()[pick].Name)
			}
			continue
		}
		if err := pickIsSkill(dto, ability, choice.Kind, pick); err != nil {
			return err
		}
	}
	if swaps > 1 {
		return fmt.Errorf("%s troca UM bônus por um poder da Tormenta, e há %d poderes", ability.Name, swaps)
	}
	return nil
}

// pickIsSkill confere a perícia escolhida: `craft` é um Ofício que a ficha
// tem; `expertise` é perícia do livro ou da ficha.
func pickIsSkill(dto CharacterDTO, ability book.RaceAbility, kind, pick string) error {
	onSheet := slices.ContainsFunc(dto.Expertises, func(e ExpertiseDTO) bool { return e.Name == pick })
	if kind == "craft" {
		if !onSheet || !strings.HasPrefix(pick, "Ofício") {
			return fmt.Errorf("%s escolhe um Ofício que a ficha tem, e %q não seria um — crie-o na aba Perícias, ou tire-o do %s antes",
				ability.Name, pick, ability.Name)
		}
		return nil
	}
	inBook := slices.ContainsFunc(book.Expertises(), func(e book.Expertise) bool { return e.Name == pick })
	if !inBook && !onSheet {
		return fmt.Errorf("%q não é perícia do livro nem desta ficha (%s)", pick, ability.Name)
	}
	return nil
}

func isTormentaPower(id string) bool {
	power, found := book.GeneralPowers()[id]
	return found && power.Kind == "tormenta"
}

// raceAbilitiesWithChoice são as habilidades com escolha das raças da ficha.
func raceAbilitiesWithChoice(dto CharacterDTO) map[string]book.RaceAbility {
	out := map[string]book.RaceAbility{}
	for _, r := range dto.Races {
		traits, found := book.RaceTraitsByKey()[r.Race]
		if !found {
			continue
		}
		for _, ability := range traits.Abilities {
			if ability.Choice != nil {
				out[ability.ID] = ability
			}
		}
	}
	return out
}

// raceWithAbility acha a raça da ficha que tem a habilidade, e a posição dela.
func raceWithAbility(dto CharacterDTO, abilityID string) (int, string) {
	for i, r := range dto.Races {
		traits, found := book.RaceTraitsByKey()[r.Race]
		if !found {
			continue
		}
		if slices.ContainsFunc(traits.Abilities, func(a book.RaceAbility) bool { return a.ID == abilityID }) {
			return i, r.Race
		}
	}
	return -1, ""
}

// raceBlobOf é o blob da raça na posição `at`: o `raceAttributeChoices` da
// primária, ou a entrada da secundária — que só existe quando ela foi aplicada.
func raceBlobOf(dto CharacterDTO, at int, race string) (map[string]json.RawMessage, bool) {
	if at == 0 {
		return objectOf(dto.RaceAttributeChoices), true
	}
	for _, entry := range secondaryEntries(dto.SecondaryRaceChoices) {
		if raceOfEntry(entry) == race {
			return entry, true
		}
	}
	return nil, false
}

func writeRaceBlob(dto *CharacterDTO, at int, race string, blob map[string]json.RawMessage) error {
	if at == 0 {
		raw, err := json.Marshal(blob)
		dto.RaceAttributeChoices = string(raw)
		return err
	}
	entries := secondaryEntries(dto.SecondaryRaceChoices)
	for i, entry := range entries {
		if raceOfEntry(entry) == race {
			entries[i] = blob
		}
	}
	raw, err := json.Marshal(entries)
	dto.SecondaryRaceChoices = string(raw)
	return err
}

func picksIn(blob map[string]json.RawMessage) map[string][]string {
	picks := map[string][]string{}
	if raw, found := blob[abilityPicksKey]; found && json.Unmarshal(raw, &picks) != nil {
		return map[string][]string{}
	}
	return picks
}

// objectOf lê um blob de objeto; torto ou vazio vira objeto vazio, como os
// irmãos.
func objectOf(blob string) map[string]json.RawMessage {
	object := map[string]json.RawMessage{}
	if json.Unmarshal([]byte(blob), &object) != nil || object == nil {
		return map[string]json.RawMessage{}
	}
	return object
}

func secondaryEntries(blob string) []map[string]json.RawMessage {
	entries := []map[string]json.RawMessage{}
	if json.Unmarshal([]byte(blob), &entries) != nil {
		return []map[string]json.RawMessage{}
	}
	return entries
}

func raceOfEntry(entry map[string]json.RawMessage) string {
	var race string
	_ = json.Unmarshal(entry["race"], &race)
	return race
}

// RaceAbilityChoice é uma habilidade com escolha de uma raça da ficha, com o
// que já foi escolhido — o que o diálogo e a pendência leem.
type RaceAbilityChoice struct {
	Race    string
	Ability book.RaceAbility
	Picks   []string
}

// RaceAbilityChoices são as habilidades com escolha das raças que VALEM na
// ficha, na ordem das raças e do catálogo.
func RaceAbilityChoices(dto CharacterDTO) []RaceAbilityChoice {
	picks := RaceAbilityPicks(dto)
	out := []RaceAbilityChoice{}
	for i, r := range dto.Races {
		traits, found := book.RaceTraitsByKey()[r.Race]
		if _, applied := raceBlobOf(dto, i, r.Race); !found || !applied {
			continue
		}
		for _, ability := range traits.Abilities {
			if ability.Choice != nil {
				out = append(out, RaceAbilityChoice{Race: r.Race, Ability: ability, Picks: picks[ability.ID]})
			}
		}
	}
	return out
}
