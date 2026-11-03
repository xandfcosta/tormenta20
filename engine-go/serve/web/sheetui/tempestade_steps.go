package sheetui

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// OS PASSOS DO ESCOLHER (ALE-473), pelo mockup aprovado: cada escolha que a
// ficha pede abre sozinha, e a lista diz quanto falta em cada uma. O dado é o
// do painel (`choicesPanel`); daqui sai só a divisão em passos.

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
	// `devotion`, `counted`, `general` ou `training`.
	Kind    string
	Race    raceChoiceCard
	Class   classChoiceCard
	Counted countedChoice
	// Power é o poder geral do passo `general`.
	Power powerChoice
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

// classTrainingOf monta o passo das perícias da classe. Quem outra fonte já treina
// sai da conta do motor, que é a mesma que a aba Perícias mostra.
func classTrainingOf(dto sheet.CharacterDTO, computed engine.ComputedSheet) *sheet.ClassTraining {
	elsewhere := map[string]bool{}
	for _, expertise := range computed.Expertises {
		if len(expertise.TrainedBy) > 0 {
			elsewhere[expertise.Name] = true
		}
	}
	return sheet.ClassTrainingOf(dto, elsewhere)
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
	if t := v.Choices.Training; t != nil && t.Limit() > 0 {
		all = append(all, choiceStep{
			ID: "class-" + t.ClassName + "-training", Title: t.ClassName + " · perícias", Meta: "treino da classe",
			Count: strconv.Itoa(t.Chosen()) + " de " + strconv.Itoa(t.Limit()), Done: t.Chosen() >= t.Limit(), Kind: "training",
		})
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
		for _, counted := range card.Counted {
			all = append(all, choiceStep{
				ID: prefix + "-counted-" + counted.ID, Title: card.ClassName + " · " + counted.Name, Meta: counted.Detail,
				Count: strconv.Itoa(counted.Chosen) + " de " + strconv.Itoa(counted.Limit), Done: counted.Chosen >= counted.Limit,
				Kind: "counted", Class: card, Counted: counted,
			})
		}
	}
	all = append(all, generalChoiceSteps(v)...)
	all = append(all, devotionStep(v))
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

// devotionStep é o passo da devoção, que toda ficha tem e só clérigo, paladino e druida têm de
// resolver.
func devotionStep(v View) choiceStep {
	devotion := v.Choices.Devotion
	step := choiceStep{ID: "devotion", Title: "Devoção", Meta: devotion.Name, Kind: "devotion", Done: true}
	if devotion.Required && devotion.Gods.Chosen == "" {
		step.Meta, step.Count, step.Done = "escolha um", "falta 1", false
	}
	if missing := devotion.Limit - devotion.Held; missing > 0 {
		step.Count, step.Done = fmt.Sprintf("falta %d", missing), false
	}
	return step
}

// generalChoiceSteps são os poderes gerais que a ficha TEM e cuja escolha
// falta — o Foco em Perícia que a origem dá (p92) sem a perícia. Sem passo
// próprio, a pendência entrava no crachá e não na lista, e o passo da classe
// podia dizer "feito" com ela em aberto. As opções são as da linha do poder na
// lista da classe.
func generalChoiceSteps(v View) []choiceStep {
	var steps []choiceStep
	for _, p := range v.Choices.Pending {
		if p.PowerID == "" {
			continue
		}
		power, found := ownedPowerChoice(v, p.PowerID)
		if !found {
			continue
		}
		label := strings.TrimPrefix(p.Label, power.Name+": escolher ")
		steps = append(steps, choiceStep{
			ID: "general-" + p.PowerID, Title: power.Name + " · " + label, Meta: "poder geral",
			Count: "falta 1", Kind: "general", Power: power,
		})
	}
	return steps
}

// ownedPowerChoice acha a linha do poder na lista de alguma classe.
func ownedPowerChoice(v View, id string) (powerChoice, bool) {
	for _, card := range v.Choices.Classes {
		for _, power := range card.Powers {
			if power.ID == id {
				return power, true
			}
		}
	}
	return powerChoice{}, false
}

// pickerStep é o passo de uma escolha de valor só, como o caminho, com
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
