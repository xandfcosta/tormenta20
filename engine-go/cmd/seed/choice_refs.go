package main

import (
	"encoding/json"
	"slices"

	"t20engine/domain/book"
)

// AS ESCOLHAS DO SEED apontam para opções do catálogo: a do poder
// (`powerChoices`) e a da habilidade de raça (`abilityPicks`, ALE-493).

// choiceRefs são as referências quebradas das escolhas de um `create`.
func choiceRefs(create map[string]json.RawMessage, where string, expertises []string) []referenciaQuebrada {
	broken := []referenciaQuebrada{}
	// A ESCOLHA DO PODER aponta duas vezes: a chave é o poder, e o valor é a
	// opção que ele oferece — um atributo, ou o id de uma ARMA do catálogo na
	// Especialização em Arma (ALE-491). Opção errada nasce como um +2 que não
	// cai em arma nenhuma, com a ficha abrindo normal.
	for _, powerID := range asChavesDe(create, "powerChoices") {
		power, known := book.PowerWithChoice(powerID)
		if !known {
			broken = append(broken, referenciaQuebrada{
				where: where + ", create.powerChoices{}", value: powerID, catalog: "poderes com escolha",
				neighbor: oVizinho(powerID, idsDosPoderes()),
			})
			continue
		}
		offered := []string{}
		for _, o := range power.Choice.Offered() {
			offered = append(offered, o.ID)
		}
		// A escolha contada por atributo oferece as perícias TREINADAS desta
		// ficha — as do Especialista (ALE-501) —, e a de perícia treinada
		// também — o Foco em Perícia (ALE-510).
		if power.Choice.CountAttribute != "" || power.Choice.Kind == "trained-expertise" {
			// Sem a lista, a ficha nasce com as treinadas PADRÃO do elenco
			// (o `enrichCreate`), e é delas que a escolha sai.
			if offered = listaDe(create, "trainedExpertises"); len(offered) == 0 {
				offered = standardTrained
			}
		}
		for _, pick := range escolhasDe(create, powerID) {
			if !slices.Contains(offered, pick) {
				broken = append(broken, referenciaQuebrada{
					where: where + ", create.powerChoices." + powerID, value: pick,
					catalog: "opções de " + power.Name, neighbor: oVizinho(pick, offered),
				})
			}
		}
	}
	// A ESCOLHA DE HABILIDADE DE RAÇA (ALE-493): a chave é a habilidade, e o
	// valor é perícia do livro ou poder geral — o da Tormenta na troca da
	// Deformidade, qualquer um na Memória Póstuma (ALE-502).
	expertisesAndPowers := append(append(slices.Clone(expertises), generalPowerIDs()...), raceAbilityIDs()...)
	picks := raceAbilityPicksOf(create)
	for _, ability := range sortedKeys(picks) {
		name, known := raceAbilityWithChoice(ability)
		if !known {
			broken = append(broken, referenciaQuebrada{
				where: where + ", create.raceAttributeChoices.abilityPicks{}", value: ability,
				catalog: "habilidades de raça com escolha",
			})
			continue
		}
		for _, pick := range picks[ability] {
			if !slices.Contains(expertisesAndPowers, pick) {
				broken = append(broken, referenciaQuebrada{
					where: where + ", create.raceAttributeChoices.abilityPicks." + ability, value: pick,
					catalog: "opções de " + name, neighbor: oVizinho(pick, expertisesAndPowers),
				})
			}
		}
	}
	return broken
}

// escolhasDe são as escolhas gravadas para um poder no `powerChoices`.
func escolhasDe(create map[string]json.RawMessage, powerID string) []string {
	var picks map[string][]string
	if json.Unmarshal(create["powerChoices"], &picks) != nil {
		return nil
	}
	return picks[powerID]
}

// raceAbilityPicksOf são as escolhas de habilidade de raça do `create`.
func raceAbilityPicksOf(create map[string]json.RawMessage) map[string][]string {
	var blob struct {
		AbilityPicks map[string][]string `json:"abilityPicks"`
	}
	if json.Unmarshal(create["raceAttributeChoices"], &blob) != nil {
		return nil
	}
	return blob.AbilityPicks
}

// raceAbilityWithChoice acha a habilidade de raça que pede escolha, pelo id.
func raceAbilityWithChoice(id string) (string, bool) {
	for _, race := range book.RaceTraitsByKey() {
		for _, ability := range race.Abilities {
			if ability.ID == id && ability.Choice != nil {
				return ability.Name, true
			}
		}
	}
	return "", false
}

func generalPowerIDs() []string {
	out := []string{}
	for id := range book.GeneralPowers() {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

// raceAbilityIDs são as habilidades de raça — a Memória Póstuma toma uma de
// outra raça (ALE-504).
func raceAbilityIDs() []string {
	out := []string{}
	for key, race := range book.RaceTraitsByKey() {
		if key != race.ID {
			continue
		}
		for _, ability := range race.Abilities {
			out = append(out, ability.ID)
		}
	}
	slices.Sort(out)
	return out
}
