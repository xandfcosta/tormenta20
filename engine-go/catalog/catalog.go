// Package catalog serves the static reference data (spells, bestiary, items, …)
// the frontend fetches from GET /catalog/:resource — the same payload the
// CatalogService, exported from t20-data by the frontend catalog-export harness
// and embedded here so the Go server needs no TS import.
package catalog

import (
	"embed"
	"encoding/json"
	"log"
	"sync"
)

//go:embed data/*.json
var files embed.FS

// Spell is the subset of a SPELL_CATALOG entry the API's cast/apply paths read.
type Spell struct {
	Circle int    `json:"circle"`
	School string `json:"school"`
	// Class lists the spell appears on — the per-spell PM limit is the level in
	// the class that PROVIDES the ability (p224), so the gate needs to know.
	Classes  []string  `json:"classes"`
	Augments []Augment `json:"augments"`
	Buff     *Buff     `json:"buff"`
}

type Augment struct {
	PmCost int    `json:"pmCost"`
	Kind   string `json:"kind"`
}

// Buff carries the modifiers an applied spell effect stores (raw JSON so it
// re-serializes byte-identical to the catalog).
type Buff struct {
	DefaultScope string          `json:"defaultScope"`
	Modifiers    json.RawMessage `json:"modifiers"`
}

var (
	spellsOnce sync.Once
	spellsByID map[string]Spell
)

// LookupSpell returns the parsed spell entry, or (zero, false) if unknown.
func LookupSpell(id string) (Spell, bool) {
	spellsOnce.Do(func() {
		spellsByID = map[string]Spell{}
		if b, err := files.ReadFile("data/spells.json"); err == nil {
			_ = json.Unmarshal(b, &spellsByID)
		}
	})
	sp, ok := spellsByID[id]
	return sp, ok
}

// Item is the subset of a CATALOG_ITEMS entry the consume + seed paths read.
type Item struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Slots      float64     `json:"slots"`
	Consumable *Consumable `json:"consumable"`
}

// Consumable mirrors the item consumable spec (scope + instant dice + effect mods).
type Consumable struct {
	Scope      string          `json:"scope"` // 'instant' | 'scene' | 'day'
	OncePerDay bool            `json:"oncePerDay"`
	Instant    *Instant        `json:"instant"`
	Modifiers  json.RawMessage `json:"modifiers"`
}

type Instant struct {
	Hp *DiceGain `json:"hp"`
	Mp *DiceGain `json:"mp"`
}

type DiceGain struct {
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus"`
}

var (
	itemsOnce sync.Once
	itemsByID map[string]Item
)

// LookupItem returns the parsed catalog item, or (zero, false) if unknown.
func LookupItem(id string) (Item, bool) {
	itemsOnce.Do(func() {
		itemsByID = map[string]Item{}
		if b, err := files.ReadFile("data/items.json"); err == nil {
			var items []Item
			if json.Unmarshal(b, &items) == nil {
				for _, it := range items {
					itemsByID[it.ID] = it
				}
			}
		}
	})
	it, ok := itemsByID[id]
	return it, ok
}

// Activation is the subset of an ACTIVATION_SPECS entry the power-grant path reads.
type Activation struct {
	ID    string           `json:"id"`
	Grant *ActivationGrant `json:"grant"`
}

// ActivationGrant is the effect a power activation grants: a temp-HP pool scaled by an
// attribute (kind "temp-hp"), or a fixed set of active-effect modifiers (kind "active-effect").
type ActivationGrant struct {
	Kind      string          `json:"kind"`
	Scope     string          `json:"scope"`
	Attribute string          `json:"attribute"`
	Modifiers json.RawMessage `json:"modifiers"`
}

var (
	activationsOnce sync.Once
	activationsByID map[string]Activation
)

// LookupActivation returns the parsed activation spec by id, or (zero, false) if unknown.
func LookupActivation(id string) (Activation, bool) {
	activationsOnce.Do(func() {
		activationsByID = map[string]Activation{}
		b, err := files.ReadFile("data/activations.json")
		if err != nil {
			log.Printf("catalog: read activations.json failed: %v", err)
			return
		}
		var specs []Activation
		if err := json.Unmarshal(b, &specs); err != nil {
			log.Printf("catalog: parse activations.json failed — power grants disabled: %v", err)
			return
		}
		for _, a := range specs {
			activationsByID[a.ID] = a
		}
	})
	a, ok := activationsByID[id]
	return a, ok
}

// resources is the ordered CatalogService registry — the GET /catalog index.
//
// The last four are AUTHORED HERE rather than dumped from t20-data like the
// others: they were the last book tables the front imported at BUILD time, so
// they were the last catalog bytes in the bundle (ALE-102). Serving them is what
// let those four t20-data modules die — the point is severing the dependency,
// not the 8 KB. Their shape is pinned by `rules_tables_test.go`, which is the
// schema validation that replaces per-field transcription tests.
var resources = []string{
	"spells", "bestiary", "items", "conditions", "deuses", "races", "origins",
	"race-defs", "class-powers", "general-powers", "granted-powers", "origens",
	"tormenta-powers", "divine-powers", "activations",
	"class-expertises", "devoto-terms", "gm-tables", "dungeon-design",
}

var valid = func() map[string]bool {
	m := make(map[string]bool, len(resources))
	for _, r := range resources {
		m[r] = true
	}
	return m
}()

// Resources returns the accepted resource names (the /catalog index payload).
func Resources() []string { return resources }

// Resource returns the raw JSON for a resource, or (nil, false) if unknown.
func Resource(name string) ([]byte, bool) {
	if !valid[name] {
		return nil, false
	}
	b, err := files.ReadFile("data/" + name + ".json")
	if err != nil {
		return nil, false
	}
	return b, true
}

// Options returns the character-creation option lists JSON (/characters/options).
func Options() ([]byte, error) {
	return files.ReadFile("data/options.json")
}
