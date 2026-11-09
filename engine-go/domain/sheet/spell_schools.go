package sheet

import (
	"slices"

	"t20engine/domain/book"
)

// AS CLASSES QUE LANÇAM PELAS ESCOLAS QUE ESCOLHERAM (p44, p61): o bardo lança "magias arcanas […] que
// pertençam a essas escolas", e o druida as divinas. A escolha mora no poder de classe que escolhe um
// NÚMERO de escolas (`Kind: "school"` com `Count`), e é ele que diz quem lança assim — não há lista de
// classes aqui. O `Count` é o que separa: o Especialista em Escola do arcanista também escolhe escola, mas
// para somar na CD dela (p38), e o arcanista lança toda arcana.
//
// A TRADIÇÃO da magia sai das classes que o verbete lista, pela `list` de cada uma: a Explosão de Chamas
// lista só o Arcanista, que é arcano, e por isso o bardo de evocação a lança. Pela lista inteira, não: ela
// diz "Bardo" só em parte das arcanas.

// CastsTheSpell diz se esta ficha lança, pela classe `class`, a magia da escola `school` cujo verbete lista
// as classes `classes`. A classe que não escolhe escola segue a lista do verbete.
//
//	sheet.CastsTheSpell(dto, "Bardo", "evocacao", []string{"Arcanista"}) // true com evocação nas escolas do bardo
func CastsTheSpell(dto CharacterDTO, class, school string, classes []string) bool {
	power := schoolChoicePower(class)
	if power == "" {
		return slices.Contains(classes, class)
	}
	progressions := book.SpellProgressions()
	tradition := progressions[class].List
	sameTradition := slices.ContainsFunc(classes, func(listed string) bool {
		prog, casts := progressions[listed]
		return casts && prog.List == tradition
	})
	return sameTradition && slices.Contains(PowerPicks(dto.PowerChoices)[power], school)
}

// schoolChoicePower é o poder em que a classe escolhe as escolas que lança, ou vazio quando ela não escolhe.
func schoolChoicePower(class string) string {
	for id, power := range book.ClassPowers() {
		if power.ClassName == class && power.Choice != nil && power.Choice.Kind == "school" && power.Choice.Count > 0 {
			return id
		}
	}
	return ""
}
