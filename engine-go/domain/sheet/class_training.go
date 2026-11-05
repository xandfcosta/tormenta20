package sheet

import (
	"slices"
	"strings"

	"t20engine/domain/book"
)

// O TREINO QUE A PRIMEIRA CLASSE MANDA ESCOLHER (p114): as perícias da lista
// dela, a escolha entre duas de quem tem uma, e tantas quanto a Inteligência,
// que não precisam ser da lista. A multiclasse não ganha nada disso (p35).
//
// A conta lê a coluna de treino, e não guarda escolha nenhuma: é o mesmo treino
// que a aba Perícias marca, então a ficha que já foi marcada à mão entra na conta
// sem migração, e nenhuma das duas telas desfaz o que a outra fez. Passar do
// limite não é recusado: o sistema mostra a conta, e a palavra é do mestre.

// craftExpertise é o Ofício na lista das classes. A linha "Ofício (ferreiro)" que a
// ficha cria conta como ele; a perícia inventada de outro nome não é do livro e
// fica fora da conta.
const craftExpertise = "Ofício"

// TrainingBlock é uma das contas do passo.
type TrainingBlock struct {
	Name    string
	Offered []string
	// Trained são as de Offered com treino marcado, e Chosen quantas delas (ou
	// das que sobraram de outra conta) esta conta aceita, nunca além de Limit.
	Trained []string
	Chosen  int
	Limit   int
}

// ClassTraining é o passo inteiro.
type ClassTraining struct {
	ClassName string
	Blocks    []TrainingBlock
}

// Chosen e Limit são a soma das contas, para o "1 de 4" da lista.
func (t ClassTraining) Chosen() int {
	total := 0
	for _, block := range t.Blocks {
		total += block.Chosen
	}
	return total
}

func (t ClassTraining) Limit() int {
	total := 0
	for _, block := range t.Blocks {
		total += block.Limit
	}
	return total
}

// ClassTrainingOf monta o passo, ou devolve nil para a ficha sem classe ou sem o
// mundo dela. `elsewhere` são as perícias que o motor diz que outra fonte treina,
// como a origem: elas não gastam escolha, e só aparecem se a coluna estiver
// marcada, para poder ser desmarcada.
//
// Uma conta nunca pede mais do que tem a oferecer: a origem que treina Luta e
// Pontaria deixaria o "Luta ou Pontaria" pendente para sempre.
//
//	ClassTrainingOf(barbaro, nil).Limit() // 4, mais a Inteligência
func ClassTrainingOf(dto CharacterDTO, elsewhere map[string]bool) *ClassTraining {
	if len(dto.Classes) == 0 {
		return nil
	}
	rule, found := book.ClassExpertisesOf(dto.Classes[0].ClassName)
	if !found {
		return nil
	}
	intelligence, err := countAttributeValue(dto, book.PowerChoice{CountAttribute: "intelligence"})
	if err != nil {
		return nil
	}

	either := []string{}
	if rule.EitherOr != nil {
		either = rule.EitherOr.Options
	}
	eitherBlock := TrainingBlock{Name: joinedByOr(either), Limit: min(1, len(either))}
	listBlock := TrainingBlock{Name: "Da lista da classe", Limit: rule.ChooseCount}
	mindBlock := TrainingBlock{Name: "Por Inteligência", Limit: max(0, intelligence)}
	byChoice := trainedByOtherChoices(dto)
	fromList, fromMind := 0, 0

	for _, row := range dto.Expertises {
		listed := row.Name
		if row.Custom {
			if !strings.HasPrefix(row.Name, craftExpertise) {
				continue
			}
			listed = craftExpertise
		}
		// A escolha de um poder não se desfaz aqui: destreinar é recusado.
		if slices.Contains(rule.Fixed, row.Name) || byChoice[row.Name] || (elsewhere[row.Name] && !row.Trained) {
			continue
		}
		inList := slices.Contains(rule.ChoosePool, listed)
		block := &mindBlock
		switch {
		case slices.Contains(either, row.Name):
			block = &eitherBlock
		case inList:
			block = &listBlock
		}
		block.Offered = append(block.Offered, row.Name)
		if !row.Trained {
			continue
		}
		block.Trained = append(block.Trained, row.Name)
		switch {
		case elsewhere[row.Name]:
		case block == &eitherBlock && eitherBlock.Chosen == 0:
			eitherBlock.Chosen = 1
		case inList:
			fromList++
		default:
			fromMind++
		}
	}

	eitherBlock.Limit = min(eitherBlock.Limit, len(eitherBlock.Offered))
	listBlock.Limit = min(listBlock.Limit, len(listBlock.Offered))
	mindBlock.Limit = min(mindBlock.Limit, len(mindBlock.Offered))
	listBlock.Chosen = min(fromList, listBlock.Limit)
	mindBlock.Chosen = min(fromMind+fromList-listBlock.Chosen, mindBlock.Limit)

	training := &ClassTraining{ClassName: dto.Classes[0].ClassName}
	for _, block := range []TrainingBlock{eitherBlock, listBlock, mindBlock} {
		if block.Limit > 0 {
			training.Blocks = append(training.Blocks, block)
		}
	}
	return training
}

// trainedByOtherChoices são as perícias que outra escolha gravou na coluna de treino:
// o Treinamento em Perícia (p131), a Memória Póstuma (p29) e o Versátil (p19). Na coluna elas parecem
// iguais às da classe, e só a escolha guardada as separa.
func trainedByOtherChoices(dto CharacterDTO) map[string]bool {
	names := map[string]bool{}
	for id, picks := range PowerPicks(dto.PowerChoices) {
		if power, known := book.PowerWithChoice(id); known && power.Choice.Kind == "untrained-expertise" {
			for _, pick := range picks {
				names[pick] = true
			}
		}
	}
	abilities := raceAbilitiesWithChoice(dto)
	for id, picks := range RaceAbilityPicks(dto) {
		if ability, known := abilities[id]; known && IsTrainedOrPower(*ability.Choice) {
			for _, pick := range picks {
				names[pick] = true
			}
		}
	}
	return names
}

// joinedByOr escreve a escolha entre duas: "Luta ou Pontaria".
func joinedByOr(options []string) string {
	if len(options) != 2 {
		return ""
	}
	return options[0] + " ou " + options[1]
}
