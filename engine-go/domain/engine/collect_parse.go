package engine

import "encoding/json"

// Parsers for the Character's JSON-string columns. Every one degrades to an
// empty value on malformed JSON, so the collection layer never throws on bad
// data — a sheet with one corrupt column still draws.

// orderedSet is a string set that also preserves first-seen order — the Go stand
// -in for a JS `Set` built from a JSON array, whose insertion order some callers
// iterate (generalPowerActiveItem, originPickedPowerIds). Go maps randomize, so
// membership (has) and iteration (list) are kept separate.
type orderedSet struct {
	list []string
	has  map[string]bool
}

func newOrderedSet() orderedSet { return orderedSet{has: map[string]bool{}} }

func (s *orderedSet) add(v string) {
	if s.has[v] {
		return
	}
	s.has[v] = true
	s.list = append(s.list, v)
}

// parseChoiceSet: a JSON string[] column into a
// deduped, order-preserving set. Non-arrays / bad JSON → empty.
func parseChoiceSet(raw string) orderedSet {
	set := newOrderedSet()
	for _, v := range parseStringArray(raw) {
		set.add(v)
	}
	return set
}

// parseStringArray decodes a JSON array, keeping only string elements in order
// (no dedup) — the shared core of parseImprovementIds / the proficiency parse.
func parseStringArray(raw string) []string {
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return nil
	}
	out := []string{}
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func parseProficiencySet(raw string) map[string]bool { return toSet(parseStringArray(raw)) }

// parseEffectModifiers: a JSON Modifier[] blob, or empty.
func parseEffectModifiers(raw string) []Modifier {
	var mods []Modifier
	if err := json.Unmarshal([]byte(raw), &mods); err != nil {
		return nil
	}
	return mods
}

// parseClassChoices: the classChoices JSON keyed by className.
// Arrays / bad JSON → empty map.
func parseClassChoices(raw string) map[string]ClassChoiceSelections {
	var m map[string]ClassChoiceSelections
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return map[string]ClassChoiceSelections{}
	}
	return m
}

type deformidadeStored struct {
	pericias      []string
	tormentaPower string
}

// raceAttrChoice is one race attribute choice; `present` distinguishes an
// absent secondary-race entry (TS `undefined`) from a real empty choice.
type raceAttrChoice struct {
	floatingPicks []string
	ascendencia   string
	deformidade   *deformidadeStored
	present       bool
}

// rawRaceChoice is the on-disk JSON shape shared by primary + secondary choices.
type rawRaceChoice struct {
	FloatingPicks []string        `json:"floatingPicks"`
	Ascendencia   string          `json:"ascendencia"`
	Deformidade   *rawDeformidade `json:"deformidade"`
}

type rawDeformidade struct {
	Pericias      []string `json:"pericias"`
	TormentaPower string   `json:"tormentaPower"`
}

func (r rawRaceChoice) toChoice() raceAttrChoice {
	return raceAttrChoice{
		floatingPicks: r.FloatingPicks,
		ascendencia:   r.Ascendencia,
		deformidade:   r.Deformidade.toStored(),
		present:       true,
	}
}

func (d *rawDeformidade) toStored() *deformidadeStored {
	if d == nil || d.Pericias == nil {
		return nil
	}
	return &deformidadeStored{pericias: d.Pericias, tormentaPower: d.TormentaPower}
}

// parseRaceAttributeChoices: the primary race's attribute
// choices. Always returns a present choice (empty on bad JSON), like the TS.
func parseRaceAttributeChoices(raw string) raceAttrChoice {
	var r rawRaceChoice
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		return raceAttrChoice{floatingPicks: []string{}, present: true}
	}
	return r.toChoice()
}

// parseSecondaryRaceChoices: opted-in secondary races keyed by
// race name. Non-arrays / bad JSON → empty map.
func parseSecondaryRaceChoices(raw string) map[string]raceAttrChoice {
	var arr []struct {
		Race string `json:"race"`
		rawRaceChoice
	}
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return map[string]raceAttrChoice{}
	}
	out := map[string]raceAttrChoice{}
	for _, e := range arr {
		if e.Race == "" {
			continue
		}
		out[e.Race] = e.rawRaceChoice.toChoice()
	}
	return out
}
