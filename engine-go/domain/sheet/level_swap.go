package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
)

// LevelSwap é o que a troca muda nas linhas, e o que ela fez com as escolhas de efeito próprio.
type LevelSwap struct {
	RemoveIDs []int64
	Add       []LevelChoiceDTO
	// Removed e Added são as escolhas que saem e entram, para o treino e a proficiência.
	Removed, Added []LevelChoiceDTO
}

// SwapLevelChoice planeja trocar uma escolha, e recusa o que deixaria a ficha fora do livro. A
// escolha nova fica no nível da antiga: o nível nunca fica vazio.
func SwapLevelChoice(dto CharacterDTO, rowID int64, power, option string) (LevelSwap, error) {
	target, found := LevelChoiceDTO{}, false
	for _, row := range dto.LevelChoices {
		if row.ID == rowID && (row.Kind == LevelKindPower || row.Kind == LevelKindPick) {
			target, found = row, true
		}
	}
	if !found {
		return LevelSwap{}, fmt.Errorf("a escolha %d não é uma escolha desta ficha", rowID)
	}
	if err := refuseOwnedOption(dto, power, option, target); err != nil {
		return LevelSwap{}, err
	}
	swap := LevelSwap{Removed: []LevelChoiceDTO{target}}
	remove := map[int64]bool{target.ID: true}
	remaining := func(kind, ref string) bool {
		return slices.ContainsFunc(dto.LevelChoices, func(row LevelChoiceDTO) bool {
			return row.Kind == kind && row.Ref == ref && !remove[row.ID]
		})
	}
	if target.Kind == LevelKindPower {
		for _, row := range dto.LevelChoices {
			if row.Kind == LevelKindPick && row.Ref == target.Ref {
				remove[row.ID] = true
				swap.Removed = append(swap.Removed, row)
			}
		}
	} else if !remaining(LevelKindPick, target.Ref) && target.Ref != power {
		for _, row := range dto.LevelChoices {
			if row.Kind == LevelKindPower && row.Ref == target.Ref {
				remove[row.ID] = true
			}
		}
	}
	if option != "" {
		added := LevelChoiceDTO{Level: target.Level, Kind: LevelKindPick, Ref: power, Pick: option}
		swap.Add = append(swap.Add, added)
		swap.Added = append(swap.Added, added)
	}
	_, isAutomatic := book.ClassPowers()[power]
	isAutomatic = isAutomatic && book.ClassPowers()[power].GrantedAtLevel != nil
	if !isAutomatic && !remaining(LevelKindPower, power) {
		swap.Add = append(swap.Add, LevelChoiceDTO{Level: target.Level, Kind: LevelKindPower, Ref: power})
	}
	for id := range remove {
		swap.RemoveIDs = append(swap.RemoveIDs, id)
	}
	slices.Sort(swap.RemoveIDs)

	after := []LevelChoiceDTO{}
	for _, row := range dto.LevelChoices {
		if !remove[row.ID] {
			after = append(after, row)
		}
	}
	after = append(after, swap.Add...)
	if err := refuseDependents(dto, after, target.Ref); err != nil {
		return LevelSwap{}, err
	}
	if err := refuseSameTier(after, swap.Added); err != nil {
		return LevelSwap{}, err
	}
	hypothesis := WithLevelChoices(dto, after)
	hypothesis.Expertises = withTrainingOf(hypothesis.Expertises, swap)
	if err := WithChoicesValid(hypothesis); err != nil {
		return LevelSwap{}, err
	}
	return swap, nil
}

// refuseDependents recusa tirar um poder que outra escolha exige (o pré-requisito do verbete).
func refuseDependents(dto CharacterDTO, after []LevelChoiceDTO, removed string) error {
	owned := map[string]bool{}
	for _, row := range after {
		if row.Kind != LevelKindClass && row.Kind != LevelKindPick {
			owned[row.Ref] = true
		}
	}
	if owned[removed] {
		return nil
	}
	name, _, _ := powerShownAs(removed)
	for _, row := range after {
		if row.Kind != LevelKindPower {
			continue
		}
		for _, requirement := range prerequisitesOf(row.Ref) {
			if requirement.Kind != "power" {
				continue
			}
			needs := requirement.ID == removed
			if slices.Contains(requirement.IDs, removed) {
				needs = !slices.ContainsFunc(requirement.IDs, func(id string) bool { return owned[id] })
			}
			if needs {
				other, _, _ := powerShownAs(row.Ref)
				return fmt.Errorf("não dá para trocar %s: %s, do nível %d, precisa dele. Troque aquela antes", name, other, row.Level)
			}
		}
	}
	return nil
}

func prerequisitesOf(id string) []book.PowerPrerequisite {
	if power, found := book.ClassPowers()[id]; found {
		return power.Prerequisites
	}
	return book.GeneralPowers()[id].Prerequisites
}

// refuseSameTier é o limite por patamar medido de verdade, agora que cada escolha tem nível.
func refuseSameTier(after, added []LevelChoiceDTO) error {
	for _, choice := range added {
		if !isAttributeIncrease(choice) {
			continue
		}
		tier, _ := characterTier(choice.Level)
		count := 0
		for _, row := range after {
			if isAttributeIncrease(row) && row.Pick == choice.Pick {
				if other, _ := characterTier(row.Level); other == tier {
					count++
				}
			}
		}
		if count > 1 {
			return fmt.Errorf("%s se escolhe uma vez por patamar para o mesmo atributo (p38): %s já está no patamar %s",
				attributeIncreasePower, book.AttributeLabel(choice.Pick), tier)
		}
	}
	return nil
}

// withTrainingOf devolve as perícias como ficam depois da troca, para a conferência ver a ficha
// já treinada: o Treinamento em Perícia treina a escolhida (p131).
func withTrainingOf(expertises []ExpertiseDTO, swap LevelSwap) []ExpertiseDTO {
	out := slices.Clone(expertises)
	set := func(rows []LevelChoiceDTO, trained bool) {
		for _, row := range rows {
			power, chooses := book.PowerWithChoice(row.Ref)
			if !chooses || power.Choice.Kind != "untrained-expertise" {
				continue
			}
			for i := range out {
				if out[i].Name == row.Pick {
					out[i].Trained = trained
				}
			}
		}
	}
	set(swap.Removed, false)
	set(swap.Added, true)
	return out
}

// refuseOwnedOption é a regra do poder que dá o que a ficha ainda não tem: a Proficiência (p129) e
// o Treinamento em Perícia (p131). A opção que está saindo não conta como já tida.
func refuseOwnedOption(dto CharacterDTO, power, option string, target LevelChoiceDTO) error {
	chosen, chooses := book.PowerWithChoice(power)
	if !chooses || option == "" || (target.Ref == power && target.Pick == option) {
		return nil
	}
	switch chosen.Choice.Kind {
	case "proficiency":
		if slices.Contains(UnmarshalStrings(dto.Proficiencies), option) {
			return fmt.Errorf("esta ficha já sabe usar %s; o %s dá uma proficiência que ela ainda não tem",
				book.ProficiencyLabel(option), chosen.Name)
		}
	case "untrained-expertise":
		if slices.ContainsFunc(dto.Expertises, func(e ExpertiseDTO) bool { return e.Name == option && e.Trained }) {
			return fmt.Errorf("%s já é treinada; o %s treina uma perícia que ainda não é", option, chosen.Name)
		}
	}
	return nil
}
