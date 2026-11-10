package book

import (
	"fmt"
	"slices"
	"strings"
)

// A ESCOLHA DO ENCANTO (p335). A Anticriatura é "letal contra um tipo de criatura (ou uma raça de
// humanoides)", e o tipo é escolhido por quem encanta a arma — o mestre. A escolha mora no encanto da arma, e
// não num verbete por tipo como a coleção de livros: a faixa 6-10 da tabela de sorteio é de um verbete só, e
// "uma raça de humanoides" abriria um verbete por raça (decisão do dono).

// anyHumanoidRace é a opção do d6 que aceita o id de qualquer raça do livro: "6) uma raça de humanoides".
const anyHumanoidRace = "raca-humanoide"

// FitsEnchantChoices recusa a escolha que não fecha com os encantos: o encanto que pede escolha e não a tem, a
// escolha fora das opções do livro, e a escolha num encanto que não pede nenhuma (ou que a arma não tem).
//
//	FitsEnchantChoices([]string{"encanto-anticriatura"}, map[string]string{"encanto-anticriatura": "morto-vivo"}) // nil
func FitsEnchantChoices(ids []string, choices map[string]string) error {
	for _, id := range ids {
		options := enchantChoiceOptions(id)
		if len(options) == 0 {
			continue
		}
		chosen, found := choices[id]
		if !found || chosen == "" {
			return fmt.Errorf("escolha o tipo de criatura da %s: %s (p335)", enchantName(id), writtenOptions(options))
		}
		if !acceptsTheChoice(options, chosen) {
			return fmt.Errorf("a %s não escolhe %q: as opções são %s (p335)",
				enchantName(id), chosen, writtenOptions(options))
		}
	}
	for id := range choices {
		if !slices.Contains(ids, id) || len(enchantChoiceOptions(id)) == 0 {
			return fmt.Errorf("o encanto %q não escolhe tipo de criatura nesta arma", id)
		}
	}
	return nil
}

// TargetOfTheChosenKind diz se o alvo é do tipo escolhido, pelo que a mesa sabe dele. Quem decide é o mestre;
// isto é o que a faixa dele lê. A raça só se confere contra o personagem, que é quem tem raça na mesa: o
// humanoide do bestiário não diz a dele.
//
//	TargetOfTheChosenKind("morto-vivo", CreatureNature{Kind: "animal"}) // TargetIsNotTheChosenKind
func TargetOfTheChosenKind(chosen string, target CreatureNature) ChosenKindMatch {
	if chosen == "" {
		return TargetKindIsUnknown
	}
	if !isRace(chosen) {
		if target.Kind == "" {
			return TargetKindIsUnknown
		}
		return matchOf(target.Kind == chosen)
	}
	if target.Race != "" {
		return matchOf(raceID(target.Race) == chosen)
	}
	if target.Kind == "" || target.Kind == "humanoide" {
		return TargetKindIsUnknown
	}
	return TargetIsNotTheChosenKind
}

// ChosenKindName é a escolha escrita para a tela: o tipo como o bestiário o escreve, em caixa baixa porque
// entra no meio da frase ("contra morto-vivo"), e a raça pelo nome do livro.
func ChosenKindName(chosen string) string {
	if isRace(chosen) {
		return strings.ToLower(RaceLabel(chosen))
	}
	return strings.ToLower(TypeName(chosen))
}

// TargetNatureName é o que a mesa sabe do alvo, escrito na régua da escolha: a raça, quando a escolha é uma
// raça e o alvo tem uma; senão o tipo. Vazio quando a mesa não sabe nenhum dos dois.
//
//	TargetNatureName("elfo", CreatureNature{Kind: "humanoide"}) // "humanoide"
func TargetNatureName(chosen string, target CreatureNature) string {
	if isRace(chosen) && target.Race != "" {
		return ChosenKindName(raceID(target.Race))
	}
	if target.Kind == "" {
		return ""
	}
	return ChosenKindName(target.Kind)
}

// CreatureNature é o que a mesa sabe de um alvo: o tipo, do bestiário e do bloco do mestre, e a raça, do
// personagem. Vazio quando a linha não diz.
type CreatureNature struct {
	Kind string
	Race string
}

// ChosenKindMatch é a leitura do alvo contra o tipo escolhido.
type ChosenKindMatch string

const (
	TargetIsTheChosenKind    ChosenKindMatch = "is"
	TargetIsNotTheChosenKind ChosenKindMatch = "isnt"
	TargetKindIsUnknown      ChosenKindMatch = "unknown"
)

func matchOf(same bool) ChosenKindMatch {
	if same {
		return TargetIsTheChosenKind
	}
	return TargetIsNotTheChosenKind
}

func enchantChoiceOptions(id string) []string {
	if item := ItemByID(id); item != nil && item.Boost != nil {
		return item.Boost.AgainstOneOf
	}
	return nil
}

func enchantName(id string) string {
	if item := ItemByID(id); item != nil {
		return item.Name
	}
	return id
}

func acceptsTheChoice(options []string, chosen string) bool {
	if chosen == anyHumanoidRace {
		return false
	}
	return slices.Contains(options, chosen) || (slices.Contains(options, anyHumanoidRace) && isRace(chosen))
}

// isRace diz se a escolha é o id de uma raça do livro. Os ids de tipo e de raça não se cruzam: o golem é
// `golem`, e o tipo é `construto`.
func isRace(chosen string) bool {
	race, found := RaceTraitsByKey()[chosen]
	return found && race.ID == chosen
}

// raceID lê a raça que a ficha guarda — o slug, ou o rótulo das fichas antigas — como o id do livro.
func raceID(stored string) string {
	if race, found := RaceTraitsByKey()[stored]; found {
		return race.ID
	}
	return stored
}

func writtenOptions(options []string) string {
	out := make([]string, 0, len(options))
	for _, option := range options {
		if option == anyHumanoidRace {
			out = append(out, "o id de uma raça de humanoides")
			continue
		}
		out = append(out, fmt.Sprintf("%q", option))
	}
	return strings.Join(out, ", ")
}
