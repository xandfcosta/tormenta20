package sheetui

import (
	"regexp"
	"strconv"
)

// A ABA EFEITOS DA FICHA PARALELA (ALE-473). O dado é o da ficha de hoje
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

// situationalToggles é o gesto do interruptor: a chave vai no sinal, como na
// ficha de hoje.
func situationalToggles(v View, row situationalRow) string {
	return "$conditional = '" + row.Key + "'; " + sheetPost(v, "/efeitos/situacao")
}

// degreeChooses escolhe o degrau E liga o situacional, pela regra do
// `degreePicker`: escolher o degrau é dizer que o PM foi gasto.
func degreeChooses(v View, row situationalRow, steps int) string {
	return "$conditional = '" + row.Key + "'; $conditional_degrees = " + strconv.Itoa(steps) + "; " +
		sheetPost(v, "/efeitos/situacao/degraus")
}
