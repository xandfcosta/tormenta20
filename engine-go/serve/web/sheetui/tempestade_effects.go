package sheetui

import (
	"regexp"
	"strconv"
	"strings"

	"t20engine/domain/search"
)

// A ABA EFEITOS DA FICHA (ALE-473). O dado é o do painel
// (`effectsPanel`); daqui sai só o que a aba lê. As descrições da condição,
// da postura e do efeito aplicado são as da coluna (`activeEffects`), uma só
// por efeito.

// effectsOfKind são os efeitos ativos de um tipo, na ordem da coluna.
func effectsOfKind(v View, kinds ...string) []activeEffect {
	var out []activeEffect
	for _, e := range activeEffects(v) {
		for _, kind := range kinds {
			if e.Kind == kind {
				out = append(out, e)
			}
		}
	}
	return out
}

var notIDSafe = regexp.MustCompile(`[^a-z0-9-]+`)

// situationalSheetID é o `id` da folha de um situacional. A chave pode ter
// dois-pontos ("flag:terreno"), e o id fica só com o que o seletor aceita.
func situationalSheetID(row situationalRow) string {
	return "situational-" + notIDSafe.ReplaceAllString(row.Key, "-")
}

// situationalMeta é a nota da linha: de onde vem e, com degrau pago, quanto.
//
//	situationalMeta(row) // "Machado silvestre · +4 · 2 PM", com o degrau
func situationalMeta(row situationalRow) string {
	meta := row.Source
	for _, d := range row.Degrees {
		if d.Chosen && row.Active {
			meta += " · " + d.Label + " · " + d.Pm
		}
	}
	return meta
}

// situationalToggles é o gesto do interruptor: a chave vai no sinal.
func situationalToggles(v View, row situationalRow) string {
	return "$conditional = '" + row.Key + "'; " + sheetPost(v, "/efeitos/situacao")
}

// degreeChooses escolhe o degrau E liga o situacional: escolher o degrau é
// dizer que o PM foi gasto.
func degreeChooses(v View, row situationalRow, steps int) string {
	return "$conditional = '" + row.Key + "'; $conditional_degrees = " + strconv.Itoa(steps) + "; " +
		sheetPost(v, "/efeitos/situacao/degraus")
}

// effectPickFilters são as buscas dos dois escolhedores da aba, filtradas no
// servidor como as do grimório.
type effectPickFilters struct {
	ConditionSearch string
	BuffSearch      string
	BuffCircle      string
}

// pickerOptionsShown são as opções que passam pela busca do nome e, quando há
// um, pelo círculo.
//
//	pickerOptionsShown(v.Effects.BuffOptions, "escudo", "") // só o Escudo da Fé
func pickerOptionsShown(options []pickerOption, query, circle string) []pickerOption {
	var shown []pickerOption
	for _, o := range options {
		if circle != "" && o.Circle != circle {
			continue
		}
		if strings.TrimSpace(query) != "" && !strings.Contains(search.Fold(o.Label), search.Fold(query)) {
			continue
		}
		shown = append(shown, o)
	}
	return shown
}

// buffCircleOptions são as opções do filtro do escolhedor de magia: "Todos" e
// os círculos que têm magia aplicável, como no grimório.
func buffCircleOptions(v View) []filterOption {
	has := map[string]bool{}
	for _, o := range v.Effects.BuffOptions {
		has[o.Circle] = true
	}
	options := []filterOption{{Value: "", Label: "Todos", Active: v.EffectPicks.BuffCircle == ""}}
	for circle := 1; circle <= 5; circle++ {
		if name := circleName(circle); has[name] {
			options = append(options, filterOption{Value: name, Label: name, Active: v.EffectPicks.BuffCircle == name})
		}
	}
	return options
}
