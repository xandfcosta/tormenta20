//go:build js && wasm

// WASM entrypoint — exposes the Go rules engine to the browser so the front runs
// the SAME rules as the server (single source, no TS duplication). Two engines
// are exposed (see PORT-PLAN.md §1): the MVP `computeCharacterSheet` (flattened
// CharacterInput) and the REAL derive — `primeEngineCatalogs` + `computeSheetV2`
// (raw Character → rich ComputedSheetV2 with every breakdown). The front primes
// the catalogs once (same JSON `ensureCatalogs` fetches) then calls computeSheetV2
// synchronously.
package main

import (
	"encoding/json"
	"syscall/js"

	"t20engine/engine"
)

// primedCatalogs holds the catalogs after primeEngineCatalogs — computeSheetV2
// needs them (the collection layer reads catalogs). One instance per page.
var primedCatalogs *engine.Catalogs

// computeSheet runs the MVP engine over a flattened CharacterInput.
func computeSheet(_ js.Value, args []js.Value) any {
	var in engine.CharacterInput
	if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(engine.ComputeCharacterSheet(&in))
	return string(out)
}

// primeEngineCatalogs ingests the fetched-catalog JSON (items/races/origins/
// classPowers/generalPowers/racas/tormentaPowerIds) into the shared Catalogs.
func primeEngineCatalogs(_ js.Value, args []js.Value) any {
	cats, err := engine.PrimeEngineCatalogs([]byte(args[0].String()))
	if err != nil {
		return errorJSON(err)
	}
	primedCatalogs = cats
	return `{"ok":true}`
}

// computeSheetV2 runs the REAL derive (collection → resolution → breakdowns) over
// a raw Character + active-conditional ids. Requires primeEngineCatalogs first.
func computeSheetV2(_ js.Value, args []js.Value) any {
	if primedCatalogs == nil {
		return `{"error":"engine catalogs not primed — call primeEngineCatalogs first"}`
	}
	var ch engine.Character
	if err := json.Unmarshal([]byte(args[0].String()), &ch); err != nil {
		return errorJSON(err)
	}
	conditionals := map[string]bool{}
	if len(args) > 1 {
		if raw := args[1].String(); raw != "" {
			var ids []string
			if err := json.Unmarshal([]byte(raw), &ids); err != nil {
				return errorJSON(err)
			}
			for _, id := range ids {
				conditionals[id] = true
			}
		}
	}
	out, _ := json.Marshal(primedCatalogs.ComputeSheetV2(ch, conditionals))
	return string(out)
}

// errorJSON returns a JSON string carrying the error, matching the sheet
// functions' shape so the TS wrapper reads `.error` uniformly.
func errorJSON(err error) string {
	out, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(out)
}

func main() {
	js.Global().Set("computeCharacterSheet", js.FuncOf(computeSheet))
	js.Global().Set("primeEngineCatalogs", js.FuncOf(primeEngineCatalogs))
	js.Global().Set("computeSheetV2", js.FuncOf(computeSheetV2))
	select {} // keep the runtime alive
}
