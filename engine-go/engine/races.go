package engine

import (
	"sort"
	"strings"
)

// variantKeys lists a subraca-gated map's keys, sorted for a stable message.
func variantKeys(variants map[string]map[string]int) string {
	keys := make([]string, 0, len(variants))
	for k := range variants {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

// raca mirrors the racas.ts Raca shape (only the fields the sheet reads).
type raca struct {
	name         string
	deslocamento int
	tamanho      string
	atributoMod  atributoMod
}

// atributoMod is the tagged union: fixed / floating / subraca-gated.
type atributoMod struct {
	kind     string // "fixed" | "floating" | "subraca-gated"
	fixed    map[string]int
	count    int
	value    int
	exclude  string
	penalty  *attrPenalty
	variants map[string]map[string]int
}

type attrPenalty struct {
	attribute string
	value     int
}

// racas keyed by slug id (racas.ts RACAS).
var racas = map[string]raca{
	"humano": {"Humano", 9, "Médio",
		atributoMod{kind: "floating", count: 3, value: 1}},
	"anao": {"Anão", 6, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"constitution": 2, "wisdom": 1, "dexterity": -1}}},
	"dahllan": {"Dahllan", 9, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"wisdom": 2, "dexterity": 1, "intelligence": -1}}},
	"elfo": {"Elfo", 12, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"intelligence": 2, "dexterity": 1, "constitution": -1}}},
	"goblin": {"Goblin", 9, "Pequeno",
		atributoMod{kind: "fixed", fixed: map[string]int{"dexterity": 2, "intelligence": 1, "charisma": -1}}},
	"lefou": {"Lefou", 9, "Médio",
		atributoMod{kind: "floating", count: 3, value: 1, exclude: "charisma",
			penalty: &attrPenalty{"charisma", -1}}},
	"minotauro": {"Minotauro", 9, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"strength": 2, "constitution": 1, "wisdom": -1}}},
	"qareen": {"Qareen", 9, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"charisma": 2, "intelligence": 1, "wisdom": -1}}},
	"golem": {"Golem", 6, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"strength": 2, "constitution": 1, "charisma": -1}}},
	"hynne": {"Hynne", 6, "Pequeno",
		atributoMod{kind: "fixed", fixed: map[string]int{"dexterity": 2, "charisma": 1, "strength": -1}}},
	"kliren": {"Kliren", 9, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"intelligence": 2, "charisma": 1, "strength": -1}}},
	"medusa": {"Medusa", 9, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"dexterity": 2, "charisma": 1}}},
	"osteon": {"Osteon", 9, "Médio",
		atributoMod{kind: "floating", count: 3, value: 1, exclude: "constitution",
			penalty: &attrPenalty{"constitution", -1}}},
	"sereia-tritao": {"Sereia/Tritão", 9, "Médio",
		atributoMod{kind: "floating", count: 3, value: 1}},
	"silfide": {"Sílfide", 9, "Minúsculo",
		atributoMod{kind: "fixed", fixed: map[string]int{"charisma": 2, "dexterity": 1, "strength": -2}}},
	"suraggel": {"Suraggel", 9, "Médio",
		atributoMod{kind: "subraca-gated", variants: map[string]map[string]int{
			"aggelus": {"wisdom": 2, "charisma": 1},
			"sulfure": {"dexterity": 2, "intelligence": 1},
		}}},
	"trog": {"Trog", 9, "Médio",
		atributoMod{kind: "fixed", fixed: map[string]int{"constitution": 2, "strength": 1, "intelligence": -1}}},
}

// resolveAtributoMod mirrors racas.ts resolveAtributoMod. Returns the mod map
// or an error (invalid floating count / missing ascendência) — caller turns
// the error into a warning + zero mods, matching raceAttrMods.
func resolveAtributoMod(r raca, floatingPicks []string, ascendencia string) (map[string]int, error) {
	m := r.atributoMod
	switch m.kind {
	case "fixed":
		out := map[string]int{}
		for k, v := range m.fixed {
			out[k] = v
		}
		return out, nil
	case "floating":
		if len(floatingPicks) != m.count {
			return nil, errf("resolveAtributoMod: %s requires exactly %d floating picks, got %d", r.name, m.count, len(floatingPicks))
		}
		if hasDuplicates(floatingPicks) {
			return nil, errf("resolveAtributoMod: %s floating picks must be distinct", r.name)
		}
		if m.exclude != "" && contains(floatingPicks, m.exclude) {
			return nil, errf("resolveAtributoMod: %s cannot place +%d in %s", r.name, m.value, m.exclude)
		}
		out := map[string]int{}
		for _, a := range floatingPicks {
			out[a] = m.value
		}
		if m.penalty != nil {
			out[m.penalty.attribute] = m.penalty.value
		}
		return out, nil
	default: // subraca-gated
		v, ok := m.variants[ascendencia]
		if ascendencia == "" || !ok {
			got := ascendencia
			if got == "" {
				got = "undefined"
			}
			return nil, errf("resolveAtributoMod: %s requires ascendência in [%s], got %s",
				r.name, variantKeys(m.variants), got)
		}
		out := map[string]int{}
		for k, val := range v {
			out[k] = val
		}
		return out, nil
	}
}
