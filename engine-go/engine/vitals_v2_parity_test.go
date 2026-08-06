package engine

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// TestVitalsParity proves the catalog-driven vitals (ComputeVitals) on real data:
// for each seed it builds the same VitalContext the front assembles (attrTotals
// from engine effects, no conditionals) and asserts {pvMax,pmMax} match the TS
// oracle (`vitals`, dumped from `enginePools`). Inc.3's target. The lone god-power
// case is arcanista-erudito (Bênção do Mana). Regenerate:
//
//	GEN_ORACLE=1 pnpm --filter frontend test parity-oracle
func TestVitalsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "parity"))
	catalogs := primeFromDump(t, dir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read parity dir %s: %v", dir, err)
	}
	var slugs []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" && e.Name()[0] != '_' {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	if len(slugs) != 16 {
		t.Fatalf("expected 16 oracle files, found %d", len(slugs))
	}

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char   Character `json:"char"`
				Vitals any       `json:"vitals"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			ctx := catalogs.vitalContextFor(oracle.Char)
			got := roundTrip(t, catalogs.ComputeVitals(ctx))
			if !reflect.DeepEqual(got, oracle.Vitals) {
				diffReport(t, "vitals", got, oracle.Vitals)
			}
		})
	}
}

// vitalContextFor builds the VitalContext the front assembles in enginePools:
// parsed ownership blobs + attrTotals from base effects (no conditionals). Lives
// on the test since the real front builds the context in TS and passes it in.
func (c *Catalogs) vitalContextFor(ch Character) VitalContext {
	effects := ComputeItemEffects(c.ActiveItemsFor(ch))
	attrTotals := map[string]int{}
	for _, a := range AttributeKeys {
		attrTotals[a] = effectiveAttribute(ch, a, effects)
	}
	level := 0
	classes := make([]ClassEntry, len(ch.Classes))
	for i, ce := range ch.Classes {
		classes[i] = ClassEntry{ClassName: ce.ClassName, Level: ce.Level}
		level += ce.Level
	}
	raceID := ""
	if len(ch.Races) > 0 {
		raceID = ch.Races[0].Race
	}
	return VitalContext{
		Level:              level,
		Classes:            classes,
		RaceID:             raceID,
		RaceAbilityChoices: parseChoiceSet(ch.RaceAbilityChoices).list,
		PowerIDs:           parseChoiceSet(ch.ClassPowers).list,
		ClassChoices:       parseClassChoices(ch.ClassChoices),
		GodPower:           ch.GodPower,
		Origin:             ch.Origin,
		OriginChoices:      parseChoiceSet(ch.OriginChoices).list,
		AttrTotals:         attrTotals,
	}
}
