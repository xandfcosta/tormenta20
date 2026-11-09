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

// EnteringClassKeepsTheDevotion recusa a classe nova que não serve ao deus da ficha: "a classe com menos
// opções determina a que deve ser seguida" (p96). O livro deixa a devoção ser mudada nesse caso; o app
// recusa e diz qual trocar, e o mestre contorna editando a ficha (decisão do dono). Quem não tem deus
// entra, e a devoção fica pendente.
//
// O "se não houver devoções compatíveis, a multiclasse não pode ser feita" da mesma página não tem ramo
// aqui: Tauron está em toda lista (`withTauron`), e nenhum par de classes do livro fica sem deus.
//
//	EnteringClassKeepsTheDevotion(dto, "Paladino") // com o deus Tenebra: "Paladino não serve a Tenebra…"
func EnteringClassKeepsTheDevotion(dto CharacterDTO, class string) error {
	if dto.God == nil || *dto.God == "" {
		return nil
	}
	classes := append(slices.Clone(dto.Classes), ClassDTO{ClassName: class, Level: 1})
	if DevotionRefusal(classes, *dto.God) == nil {
		return nil
	}
	god := book.DevotionName(*dto.God)
	return fmt.Errorf("%s não serve a %s, o deus desta ficha: troque a devoção por um deus que %s aceite antes de entrar na classe (p96)",
		class, god, class)
}

// devotionIsValid confere o deus da ficha contra as classes dela.
func devotionIsValid(dto CharacterDTO) error {
	if dto.God == nil {
		return nil
	}
	return DevotionRefusal(dto.Classes, *dto.God)
}

// GrantedPowersFor são os poderes que esta ficha escolhe sob o deus pedido: os dele, e os que as regras
// da mesa emprestam a quem tem esse deus. A lista emprestada SOMA à do deus (decisão do dono).
//
//	GrantedPowersFor(dto, "tauron") // os de Arsenal, com a regra "devoto de Tauron escolhe poderes de Arsenal"
func GrantedPowersFor(dto CharacterDTO, god string) []book.GrantedPower {
	out := slices.Clone(book.GrantedPowersOf(god))
	if dto.Ruleset == nil || god == "" {
		return out
	}
	projected := dto
	projected.God = &god
	ec, err := EngineCharacterFrom(projected)
	if err != nil {
		// A conversão é um JSON de ida e volta do próprio agregado: falhar aqui é defeito, e a ficha
		// segue com a lista do livro, que é a que vale sem regra nenhuma.
		return out
	}
	for _, lender := range dto.Ruleset.BorrowedGods(ec) {
		for _, power := range book.GrantedPowersOf(lender) {
			if !slices.ContainsFunc(out, func(p book.GrantedPower) bool { return p.Name == power.Name }) {
				out = append(out, power)
			}
		}
	}
	return out
}

// GrantedPowerNames são os nomes de `GrantedPowersFor`, para quem pergunta "este poder é do deus?"
// várias vezes: a lista se calcula uma vez, porque a regra da mesa pede converter a ficha.
func GrantedPowerNames(dto CharacterDTO, god string) []string {
	names := []string{}
	for _, power := range GrantedPowersFor(dto, god) {
		names = append(names, power.Name)
	}
	return names
}

// GrantedPowerLimit é quantos poderes concedidos a devoção dá: um (p96), dois para quem tem classe que
// serve a um deus (p57, p61, p82), e nenhum na devoção sem lista, como o Panteão.
func GrantedPowerLimit(dto CharacterDTO, god string) int {
	switch {
	case len(GrantedPowersFor(dto, god)) == 0:
		return 0
	case DevotionRequired(dto.Classes):
		return 2
	}
	return 1
}

// ExtraGrantedPowers é quantos poderes concedidos do deus da ficha passam dos que a devoção dá. Os
// concedidos são poderes gerais (p124), e cada um a mais ocupa uma vaga de poder da classe (decisão do
// dono). O de outro deus, numa ficha de antes de a troca tirá-lo, não conta, como no `GrantedPowersMissing`.
//
//	ExtraGrantedPowers(dto) // 1, para o guerreiro devoto de Khalmyr com dois concedidos: a devoção dá um
func ExtraGrantedPowers(dto CharacterDTO) int {
	held := UnmarshalStrings(dto.GodPowers)
	// Toda devoção com lista dá ao menos um: com um só marcado, não há o que contar, e a conta abaixo
	// converte a ficha quando há regra da mesa.
	if dto.God == nil || len(held) <= 1 {
		return 0
	}
	ofTheGod := GrantedPowerNames(dto, *dto.God)
	count := 0
	for _, power := range held {
		if slices.Contains(ofTheGod, power) {
			count++
		}
	}
	return max(count-GrantedPowerLimit(dto, *dto.God), 0)
}

// GrantedPowersMissing é quantos poderes do deus de agora faltam escolher. O de outro deus, numa ficha de
// antes de a troca tirá-lo, não ocupa vaga.
func GrantedPowersMissing(dto CharacterDTO) int {
	if dto.God == nil {
		return 0
	}
	held := 0
	ofTheGod := GrantedPowerNames(dto, *dto.God)
	for _, power := range UnmarshalStrings(dto.GodPowers) {
		if slices.Contains(ofTheGod, power) {
			held++
		}
	}
	return max(GrantedPowerLimit(dto, *dto.God)-held, 0)
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
	ofTheTarget := GrantedPowerNames(dto, target)
	for _, power := range UnmarshalStrings(dto.GodPowers) {
		if target != "" && slices.Contains(ofTheTarget, power) {
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
		lines = append(lines, leaving[0]+" sai da ficha.")
	} else if len(leaving) > 1 {
		lines = append(lines, book.WrittenList(leaving)+" saem da ficha.")
	}
	for _, power := range kept {
		lender := book.DevotionName(target) + " também o concede."
		if !slices.ContainsFunc(book.GrantedPowersOf(target), func(p book.GrantedPower) bool { return p.Name == power }) {
			lender = "uma regra da mesa o empresta a quem é devoto de " + book.DevotionName(target) + "."
		}
		lines = append(lines, power+" continua valendo: "+lender)
	}
	limit := GrantedPowerLimit(dto, target)
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

// ItemsWornAsWielded são os itens da ficha que uma regra da mesa deixa vestir valendo como empunhados.
// Uma conversão só para a mochila inteira, e não uma por item.
func ItemsWornAsWielded(dto CharacterDTO) (map[int64]bool, error) {
	out := map[int64]bool{}
	if dto.Ruleset == nil {
		return out, nil
	}
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return nil, err
	}
	for _, it := range ec.Items {
		if dto.Ruleset.WearsAsWielded(ec, it) {
			out[it.ID] = true
		}
	}
	return out, nil
}
