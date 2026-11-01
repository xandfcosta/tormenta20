package sheet

import "fmt"

// ClassCanLeave diz se esta classe pode sair da ficha, e o que a segura.
//
// A primeira classe não sai: foi ela que deu os PV iniciais, as perícias e as proficiências
// (p35). As outras saem no 1º nível, se nada escolhido depender delas. A pergunta é feita à
// ficha como ela ficaria, com a mesma conferência que vale para qualquer escolha. As magias
// são do `character.LeaveClass`, porque o catálogo delas não é deste pacote.
//
//	ClassCanLeave(guerreiro3Ladino1, "Ladino") // nil, se nada veio do Ladino
func ClassCanLeave(dto CharacterDTO, class string) error {
	index := -1
	remaining := dto
	remaining.Classes = make([]ClassDTO, 0, len(dto.Classes))
	for i, c := range dto.Classes {
		if c.ClassName == class {
			index = i
			continue
		}
		remaining.Classes = append(remaining.Classes, c)
	}
	switch index {
	case -1:
		return fmt.Errorf("%s não é uma classe deste personagem", class)
	case 0:
		return fmt.Errorf("%s é a primeira classe do personagem: foi ela que deu os PV iniciais, as perícias e as proficiências (p35)", class)
	}
	if dto.Classes[index].Level != 1 {
		return fmt.Errorf("%s está no nível %d, e só o 1º nível sai da ficha", class, dto.Classes[index].Level)
	}
	if err := chosenFitPowers(remaining); err != nil {
		return fmt.Errorf("não dá para tirar %s: %w", class, err)
	}
	choice := ClassChoiceSelections(dto)[class]
	if choice.Devotee != "" {
		return fmt.Errorf("não dá para tirar %s: o devoto %s foi escolhido nela",
			class, optionLabel(ClassDevotees(class), choice.Devotee))
	}
	if choice.Path != "" {
		return fmt.Errorf("não dá para tirar %s: o caminho %s foi escolhido nela",
			class, optionLabel(classPaths[class].Options, choice.Path))
	}
	return nil
}

// optionLabel é o rótulo que a pessoa escolheu. O valor gravado serve de resposta quando
// ele saiu da lista, porque ainda diz mais do que nada.
func optionLabel(options []ChoiceOption, value string) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}
