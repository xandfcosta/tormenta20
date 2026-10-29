package sheetui

import (
	"encoding/json"
	"strconv"
)

// OS PASSOS DO ESCOLHER (ALE-473), pelo mockup aprovado: cada escolha que a
// ficha pede abre sozinha, e a lista diz quanto falta em cada uma. O dado é o
// da ficha de hoje (`choicesPanel`); daqui sai só a divisão em passos.

// choiceStep é um passo do Escolher (ver o GLOSSARY).
type choiceStep struct {
	// ID é o valor do sinal `choice_step` e o fim do `id` da seção.
	ID    string
	Title string
	Meta  string
	// Count é o "1 de 9" do cabeçalho do passo; vazio quando a escolha é
	// de um valor só.
	Count string
	Done  bool
	// Kind diz o que o passo desenha: `race`, `origin`, `powers`, `path`,
	// `devotee` ou `counted`.
	Kind    string
	Race    raceChoiceCard
	Class   classChoiceCard
	Counted countedChoice
}

// Tag é o que a lista mostra na ponta do passo: quanto falta, ou "feito".
func (s choiceStep) Tag() string {
	if s.Done {
		return "feito"
	}
	if s.Count == "" {
		return "falta 1"
	}
	return s.Count
}

// choiceStepsOf são os passos na ordem em que o livro monta um personagem —
// raça, origem, classe —, com os que faltam antes dos feitos.
//
//	choiceStepsOf(v) // [Raça · Humano (falta 1), …, Guerreiro · caminho (feito)]
func choiceStepsOf(v View) []choiceStep {
	var all []choiceStep
	for i, card := range v.Choices.Races {
		step := choiceStep{ID: "race-" + strconv.Itoa(i), Title: "Raça · " + card.Race, Meta: "habilidades da raça", Kind: "race", Race: card}
		if card.Attribute != nil {
			step.Meta = attributeWrittenChoice(*card.Attribute)
		}
		if n := pendingIn(v, "raca"); n > 0 {
			step.Count = "falta " + strconv.Itoa(n)
		} else {
			step.Done = true
		}
		all = append(all, step)
	}
	if o := v.Choices.Origin; o != nil {
		all = append(all, choiceStep{ID: "origin", Title: "Origem · " + o.Name, Meta: "benefícios", Count: originWritten(*o), Done: o.Left <= 0, Kind: "origin"})
	}
	for _, card := range v.Choices.Classes {
		prefix := "class-" + card.ClassName
		if len(card.Powers) > 0 {
			all = append(all, choiceStep{
				ID: prefix + "-powers", Title: card.ClassName + " " + strconv.FormatInt(card.Level, 10), Meta: "poderes",
				Count: strconv.Itoa(card.Used) + " de " + strconv.Itoa(card.Slots), Done: card.Used >= card.Slots, Kind: "powers", Class: card,
			})
		}
		if card.Path != nil {
			all = append(all, pickerStep(prefix+"-path", card.ClassName+" · caminho", "path", card, *card.Path))
		}
		if card.Devotee != nil {
			all = append(all, pickerStep(prefix+"-devotee", card.ClassName+" · devoto", "devotee", card, *card.Devotee))
		}
		for _, counted := range card.Counted {
			all = append(all, choiceStep{
				ID: prefix + "-counted-" + counted.ID, Title: card.ClassName + " · " + counted.Name, Meta: counted.Detail,
				Count: strconv.Itoa(counted.Chosen) + " de " + strconv.Itoa(counted.Limit), Done: counted.Chosen >= counted.Limit,
				Kind: "counted", Class: card, Counted: counted,
			})
		}
	}
	steps := make([]choiceStep, 0, len(all))
	for _, done := range []bool{false, true} {
		for _, s := range all {
			if s.Done == done {
				steps = append(steps, s)
			}
		}
	}
	return steps
}

// pickerStep é o passo de uma escolha de valor só — o caminho, o devoto —, com
// o que foi escolhido na nota.
func pickerStep(id, title, kind string, card classChoiceCard, picker pickerChoice) choiceStep {
	step := choiceStep{ID: id, Title: title, Meta: "escolha um", Kind: kind, Class: card, Done: picker.Chosen != ""}
	for _, o := range picker.Options {
		if o.Active {
			step.Meta = o.Label
		}
	}
	return step
}

// pendingSteps conta os passos que faltam, para o "Faltam N escolhas".
func pendingSteps(steps []choiceStep) int {
	n := 0
	for _, s := range steps {
		if !s.Done {
			n++
		}
	}
	return n
}

// stepsTitle é o cabeçalho da lista dos passos: "Faltam 3 escolhas".
func stepsTitle(steps []choiceStep) string {
	if n := pendingSteps(steps); n > 0 {
		return pendingTitle(n)
	}
	return "Tudo escolhido"
}

// choicesOpens é o gesto do Escolher. No desktop a lista e o passo dividem o
// diálogo, e ele abre no primeiro passo; no celular é uma camada de cada vez,
// e ele abre na lista. Carrega também a distribuição de atributos que a raça
// já gravou — sem isso as opções nasceriam desmarcadas sobre uma escolha
// feita.
func choicesOpens(v View) string {
	chosen := []string{}
	for _, card := range v.Choices.Races {
		if card.Attribute != nil && card.Attribute.Kind != "ascendencia" {
			chosen = card.Attribute.Chosen
		}
	}
	written, _ := json.Marshal(chosen)
	first := ""
	if steps := choiceStepsOf(v); len(steps) > 0 {
		first = steps[0].ID
	}
	return "$choice_step = matchMedia('(min-width: 64rem)').matches ? '" + first + "' : ''; " +
		"$power_pick_filter = ''; $race_attributes = " + string(written)
}
