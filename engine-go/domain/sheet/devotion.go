package sheet

import (
	"fmt"
	"slices"
	"strings"

	"t20engine/domain/book"
)

// DevotionOptions são as devoções que a ficha pode escolher. Quem não tem classe que sirva a um deus
// escolhe qualquer um, ou Tauron; quem tem fica com o que TODAS aceitam, e é o que faz um
// clérigo/paladino só ver quem serve aos dois.
func DevotionOptions(classes []ClassDTO) []ChoiceOption {
	_, _, gods := book.CharacterCatalogs()
	options := withTauron(godsThat(gods, func(book.God) bool { return true }))
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

// withTauron põe Tauron na lista, em ordem de nome.
func withTauron(options []ChoiceOption) []ChoiceOption {
	all := append(options, ChoiceOption{Value: book.DevotionTauron, Label: book.DevotionName(book.DevotionTauron)})
	slices.SortFunc(all, func(a, b ChoiceOption) int { return strings.Compare(a.Label, b.Label) })
	return all
}

// DevotionRequired diz se a ficha tem classe que serve a um deus, e por isso fica pendente sem ele.
func DevotionRequired(classes []ClassDTO) bool {
	return slices.ContainsFunc(classes, func(c ClassDTO) bool { return len(ClassDevotees(c.ClassName)) > 0 })
}

// DevotionRefusal diz por que a devoção não vale para estas classes, ou nil quando vale. A forja a usa
// antes de a ficha existir, e por isso ela recebe as classes e não a ficha.
func DevotionRefusal(classes []ClassDTO, god string) error {
	if god == "" {
		return nil
	}
	for _, class := range classes {
		accepted := ClassDevotees(class.ClassName)
		if len(accepted) > 0 && !slices.ContainsFunc(accepted, func(o ChoiceOption) bool { return o.Value == god }) {
			return fmt.Errorf("%s não serve a %s: escolha outro deus, ou Nenhum", class.ClassName, book.DevotionName(god))
		}
	}
	if !slices.ContainsFunc(DevotionOptions(classes), func(o ChoiceOption) bool { return o.Value == god }) {
		return fmt.Errorf("%s não é uma devoção que esta ficha possa ter", book.DevotionName(god))
	}
	return nil
}

// devotionIsValid confere o deus da ficha contra as classes dela.
func devotionIsValid(dto CharacterDTO) error {
	if dto.God == nil {
		return nil
	}
	return DevotionRefusal(dto.Classes, *dto.God)
}

// GrantedPowerIsOfGod diz se o poder é um dos que o deus concede.
func GrantedPowerIsOfGod(god, power string) bool {
	return slices.ContainsFunc(book.GrantedPowersOf(god), func(p book.GrantedPower) bool { return p.Name == power })
}

// GrantedPowerLimit é quantos poderes concedidos a devoção dá: um (p96), dois para quem tem classe que
// serve a um deus (p57, p61, p82), e nenhum na devoção sem lista, como o Panteão.
func GrantedPowerLimit(classes []ClassDTO, god string) int {
	switch {
	case len(book.GrantedPowersOf(god)) == 0:
		return 0
	case DevotionRequired(classes):
		return 2
	}
	return 1
}

// GrantedPowersMissing é quantos poderes do deus de agora faltam escolher. O do deus antigo não conta: ele
// fica na ficha até a mesa decidir, e não ocupa a vaga do novo.
func GrantedPowersMissing(dto CharacterDTO) int {
	if dto.God == nil {
		return 0
	}
	held := 0
	for _, power := range UnmarshalStrings(dto.GodPowers) {
		if GrantedPowerIsOfGod(*dto.God, power) {
			held++
		}
	}
	return max(GrantedPowerLimit(dto.Classes, *dto.God)-held, 0)
}

// DevotionChange diz o que muda na ficha ao trocar de deus, para o passo pedir a confirmação. O alvo vazio
// é ficar sem deus.
//
//	DevotionChange(dto, "azgher") // ["Deus: Khalmyr → Azgher.", "Falta escolher 2 poderes concedidos de Azgher."]
func DevotionChange(dto CharacterDTO, target string) []string {
	current := ""
	if dto.God != nil {
		current = book.DevotionName(*dto.God)
	}
	var kept, leaving []string
	for _, power := range UnmarshalStrings(dto.GodPowers) {
		if target != "" && GrantedPowerIsOfGod(target, power) {
			kept = append(kept, power)
		} else {
			leaving = append(leaving, power)
		}
	}
	lines := []string{"Deus: " + current + " → " + book.DevotionName(target) + "."}
	if target == "" {
		lines[0] = current + " deixa de ser o deus da ficha."
	}
	if len(leaving) == 1 {
		lines = append(lines, leaving[0]+" fica na ficha, mas é de outro deus. Tirar é da mesa.")
	} else if len(leaving) > 1 {
		lines = append(lines, joinedNames(leaving)+" ficam na ficha, mas são de outro deus. Tirar é da mesa.")
	}
	for _, power := range kept {
		lines = append(lines, power+" continua valendo: "+book.DevotionName(target)+" também o concede.")
	}
	limit := GrantedPowerLimit(dto.Classes, target)
	switch missing := limit - len(kept); {
	case target == "":
		if DevotionRequired(dto.Classes) {
			lines = append(lines, "A devoção fica pendente.")
		}
	case limit == 0:
		lines = append(lines, book.DevotionName(target)+" não recebe poder concedido.")
	case missing == 1:
		lines = append(lines, "Falta escolher 1 poder concedido de "+book.DevotionName(target)+".")
	case missing > 1:
		lines = append(lines, fmt.Sprintf("Falta escolher %d poderes concedidos de %s.", missing, book.DevotionName(target)))
	}
	return lines
}

// joinedNames escreve a lista como a frase pede: "A, B e C".
func joinedNames(names []string) string {
	last := len(names) - 1
	return strings.Join(names[:last], ", ") + " e " + names[last]
}
