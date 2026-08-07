// Package catalog serves the static reference data (spells, bestiary, items, …)
// the frontend fetches from GET /catalog/:resource — byte-identical to the Nest
// CatalogService, exported from t20-data by the frontend catalog-export harness
// and embedded here so the Go server needs no TS import.
package catalog

import "embed"

//go:embed data/*.json
var files embed.FS

// resources is the ordered CatalogService registry — the GET /catalog index.
var resources = []string{
	"spells", "bestiary", "items", "conditions", "deuses", "races", "origins",
	"race-defs", "class-powers", "general-powers", "granted-powers", "origens",
	"tormenta-powers", "divine-powers", "activations",
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
