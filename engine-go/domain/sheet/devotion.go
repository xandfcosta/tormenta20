package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
)

// DevotionOptions são as devoções que a ficha pode escolher. Quem não tem classe que sirva a um deus
// escolhe qualquer um; quem tem fica com o que TODAS aceitam, e é o que faz um clérigo/paladino só
// ver quem serve aos dois.
func DevotionOptions(classes []ClassDTO) []ChoiceOption {
	_, _, gods := book.CharacterCatalogs()
	options := godsThat(gods, func(book.God) bool { return true })
	limited := false
	for _, class := range classes {
		accepted := ClassDevotees(class.ClassName)
		if len(accepted) == 0 {
			continue
		}
		if !limited {
			options, limited = accepted, true
			continue
		}
		options = slices.DeleteFunc(slices.Clone(options), func(o ChoiceOption) bool {
			return !slices.ContainsFunc(accepted, func(a ChoiceOption) bool { return a.Value == o.Value })
		})
	}
	return options
}

// DevotionRequired diz se a ficha tem classe que serve a um deus, e por isso fica pendente sem ele.
func DevotionRequired(classes []ClassDTO) bool {
	return slices.ContainsFunc(classes, func(c ClassDTO) bool { return len(ClassDevotees(c.ClassName)) > 0 })
}

// devotionIsValid confere o deus da ficha contra as classes dela.
func devotionIsValid(dto CharacterDTO) error {
	if dto.God == nil || *dto.God == "" {
		return nil
	}
	god := *dto.God
	for _, class := range dto.Classes {
		accepted := ClassDevotees(class.ClassName)
		if len(accepted) > 0 && !slices.ContainsFunc(accepted, func(o ChoiceOption) bool { return o.Value == god }) {
			return fmt.Errorf("%s não serve a %s: escolha uma das devoções da classe", class.ClassName, book.DevotionName(god))
		}
	}
	if !slices.ContainsFunc(DevotionOptions(dto.Classes), func(o ChoiceOption) bool { return o.Value == god }) {
		return fmt.Errorf("%s não é uma devoção que esta ficha possa ter", book.DevotionName(god))
	}
	return nil
}

// GrantedPowerIsOfGod diz se o poder é um dos que o deus concede.
func GrantedPowerIsOfGod(god, power string) bool {
	return slices.ContainsFunc(book.GrantedPowersOf(god), func(p book.GrantedPower) bool { return p.Name == power })
}
