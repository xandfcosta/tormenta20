//go:build js && wasm

// WASM entrypoint — exposes the Go rules engine to the browser so the front runs
// the SAME rules as the server (single source, no TS duplication). Two engines
// The catalog-driven derive is exposed: `primeEngineCatalogs` + `computeSheetV2`
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
	ch, conditionals, errJSON := parseCharArgs(args)
	if errJSON != "" {
		return errJSON
	}
	out, _ := json.Marshal(primedCatalogs.ComputeSheetV2(*ch, conditionals))
	return string(out)
}

// computeEffects returns the resolved ItemEffects (byTarget/flags/conditional)
// for a raw Character + active-conditional ids — the derive's choke point, so
// the front's `characterEffects`/`useCharacterEffects` run on Go while the thin
// breakdown helpers stay TS over these effects (Inc.2 task #7). Flags marshal as
// a sorted array (ItemEffects.MarshalJSON); the TS wrapper rebuilds the Set.
func computeEffects(_ js.Value, args []js.Value) any {
	ch, conditionals, errJSON := parseCharArgs(args)
	if errJSON != "" {
		return errJSON
	}
	effects := engine.ApplyActiveConditionals(
		engine.ComputeItemEffects(primedCatalogs.ActiveItemsFor(*ch)), conditionals)
	out, _ := json.Marshal(effects)
	return string(out)
}

// resolveConditionalDisplay runs the non-stacking display resolution over an
// active stance's conditional rows (t20-data resolveConditionalDisplay) — pure,
// needs no primed catalogs. Returns the surviving {target, amount} rows.
func resolveConditionalDisplay(_ js.Value, args []js.Value) any {
	var rows []engine.ConditionalDisplayInput
	if err := json.Unmarshal([]byte(args[0].String()), &rows); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(engine.ResolveConditionalDisplay(rows))
	return string(out)
}

// parseCharArgs unmarshals the shared (charJson, conditionalsJson) argument pair
// used by computeSheetV2 + computeEffects. Returns an error-JSON string when the
// catalogs aren't primed or a payload is malformed.
func parseCharArgs(args []js.Value) (*engine.Character, map[string]bool, string) {
	if primedCatalogs == nil {
		return nil, nil, `{"error":"engine catalogs not primed — call primeEngineCatalogs first"}`
	}
	var ch engine.Character
	if err := json.Unmarshal([]byte(args[0].String()), &ch); err != nil {
		return nil, nil, errorJSON(err)
	}
	conditionals := map[string]bool{}
	if len(args) > 1 {
		if raw := args[1].String(); raw != "" {
			var ids []string
			if err := json.Unmarshal([]byte(raw), &ids); err != nil {
				return nil, nil, errorJSON(err)
			}
			for _, id := range ids {
				conditionals[id] = true
			}
		}
	}
	return &ch, conditionals, ""
}

// computeVitals runs the catalog-driven vitals pipeline (PV/PM máximos) over a
// normalized VitalContext the front builds. Requires primeEngineCatalogs first.
func computeVitals(_ js.Value, args []js.Value) any {
	if primedCatalogs == nil {
		return `{"error":"engine catalogs not primed — call primeEngineCatalogs first"}`
	}
	var ctx engine.VitalContext
	if err := json.Unmarshal([]byte(args[0].String()), &ctx); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(primedCatalogs.ComputeVitals(ctx))
	return string(out)
}

// computeEquippedFlags returns the always-on flags carried by a character's
// equipped items with item provenance (effect-source.ts equippedItemFlagEffects)
// — the last effects-cru consumer to move off TS. Requires primeEngineCatalogs
// first (item modifiers come from the catalog).
func computeEquippedFlags(_ js.Value, args []js.Value) any {
	if primedCatalogs == nil {
		return `{"error":"engine catalogs not primed — call primeEngineCatalogs first"}`
	}
	var items []engine.CharacterItem
	if err := json.Unmarshal([]byte(args[0].String()), &items); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(primedCatalogs.ComputeEquippedFlags(items))
	return string(out)
}

// computeWeaponCards resolves the wielded-weapon formula cards (attack/damage/
// crit) for a raw Character + active-conditional ids. Requires primeEngineCatalogs.
func computeWeaponCards(_ js.Value, args []js.Value) any {
	ch, conditionals, errJSON := parseCharArgs(args)
	if errJSON != "" {
		return errJSON
	}
	out, _ := json.Marshal(primedCatalogs.ComputeWeaponCards(*ch, conditionals))
	return string(out)
}

// pointBuyStatus runs the creation point-buy rules (p17) over a base attribute
// spread — pure, needs no primed catalogs. Returns {spent, warnings}.
func pointBuyStatus(_ js.Value, args []js.Value) any {
	var attrs map[string]int
	if err := json.Unmarshal([]byte(args[0].String()), &attrs); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(engine.PointBuyStatusFor(attrs))
	return string(out)
}

// spellPmLimit answers the p224 ceiling for ONE spell — the number the cast
// dialog must gate on. The client used to read the HUD's per-character summary
// instead, which is a DIFFERENT number, and offered augments the server refused
// (ALE-92). Requires primeEngineCatalogs. Args: (characterJson, spellClassesJson).
func spellPmLimit(_ js.Value, args []js.Value) any {
	if primedCatalogs == nil {
		return `{"error":"engine catalogs not primed — call primeEngineCatalogs first"}`
	}
	var ch engine.Character
	if err := json.Unmarshal([]byte(args[0].String()), &ch); err != nil {
		return errorJSON(err)
	}
	var spellClasses []string
	if err := json.Unmarshal([]byte(args[1].String()), &spellClasses); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(map[string]int{"limit": primedCatalogs.SpellPmLimitFor(ch, spellClasses)})
	return string(out)
}

// boardPathCost mede um caminho no mapa de batalha (T20 p238) para a tela poder
// mostrar o custo ENQUANTO o jogador arrasta, sem uma ida ao servidor por
// quadrado. É a mesma função que o gateway roda quando o movimento é confirmado
// — o front antecipa, o Go decide, e nunca existem duas implementações da regra.
//
// Pura: não precisa de `primeEngineCatalogs`.
func boardPathCost(_ js.Value, args []js.Value) any {
	var payload struct {
		Path      []engine.Square `json:"path"`
		Difficult []engine.Square `json:"difficult"`
		Budget    int             `json:"budget"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &payload); err != nil {
		return errorJSON(err)
	}
	terrain := engine.MoveTerrain{Difficult: map[engine.Square]bool{}}
	for _, square := range payload.Difficult {
		terrain.Difficult[square] = true
	}
	out, _ := json.Marshal(engine.PathCost(payload.Path, terrain, payload.Budget))
	return string(out)
}

// boardReach devolve as casas ALCANÇÁVEIS a partir de um quadrado (T20 p238),
// para a tela acender o losango em vez de fazer a mesa contar. Uma chamada por
// seleção, e não uma por casa: o cliente não sabe somar diagonal, e é bom que
// não saiba.
//
// Pura: não precisa de `primeEngineCatalogs`.
func boardReach(_ js.Value, args []js.Value) any {
	var payload struct {
		From      engine.Square   `json:"from"`
		Difficult []engine.Square `json:"difficult"`
		Budget    int             `json:"budget"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &payload); err != nil {
		return errorJSON(err)
	}
	terrain := engine.MoveTerrain{Difficult: map[engine.Square]bool{}}
	for _, square := range payload.Difficult {
		terrain.Difficult[square] = true
	}
	out, _ := json.Marshal(engine.ReachableSquares(payload.From, payload.Budget, terrain))
	return string(out)
}

// boardMeasure é a RÉGUA da mesa: a distância entre dois quadrados, em
// quadrados e em metros, com a faixa de alcance do livro (T20 p224).
//
// Mora no motor pela mesma razão do custo do caminho: a diagonal dobrada é
// regra, e uma segunda implementação na tela seria uma segunda verdade. Aqui não
// há ida ao servidor NENHUMA — a régua não muda estado e ninguém mais precisa
// vê-la, então ela vive inteira no navegador.
//
// Pura: não precisa de `primeEngineCatalogs`.
func boardMeasure(_ js.Value, args []js.Value) any {
	var payload struct {
		From engine.Square `json:"from"`
		To   engine.Square `json:"to"`
	}
	if err := json.Unmarshal([]byte(args[0].String()), &payload); err != nil {
		return errorJSON(err)
	}
	out, _ := json.Marshal(engine.Measure(payload.From, payload.To))
	return string(out)
}

// boardBudget converte deslocamento em metros para quadrados (T20 p106).
func boardBudget(_ js.Value, args []js.Value) any {
	out, _ := json.Marshal(map[string]int{"squares": engine.SquaresForDisplacement(args[0].Float())})
	return string(out)
}

// boardFootprint devolve o lado da peça em quadrados por tamanho (T20 p107).
func boardFootprint(_ js.Value, args []js.Value) any {
	out, _ := json.Marshal(map[string]int{"footprint": engine.FootprintForSize(args[0].String())})
	return string(out)
}

// errorJSON returns a JSON string carrying the error, matching the sheet
// functions' shape so the TS wrapper reads `.error` uniformly.
func errorJSON(err error) string {
	out, _ := json.Marshal(map[string]string{"error": err.Error()})
	return string(out)
}

func main() {
	js.Global().Set("primeEngineCatalogs", js.FuncOf(primeEngineCatalogs))
	js.Global().Set("computeSheetV2", js.FuncOf(computeSheetV2))
	js.Global().Set("computeEffects", js.FuncOf(computeEffects))
	js.Global().Set("computeVitals", js.FuncOf(computeVitals))
	js.Global().Set("resolveConditionalDisplay", js.FuncOf(resolveConditionalDisplay))
	js.Global().Set("computeEquippedFlags", js.FuncOf(computeEquippedFlags))
	js.Global().Set("pointBuyStatus", js.FuncOf(pointBuyStatus))
	js.Global().Set("computeWeaponCards", js.FuncOf(computeWeaponCards))
	js.Global().Set("spellPmLimit", js.FuncOf(spellPmLimit))
	js.Global().Set("boardPathCost", js.FuncOf(boardPathCost))
	js.Global().Set("boardReach", js.FuncOf(boardReach))
	js.Global().Set("boardMeasure", js.FuncOf(boardMeasure))
	js.Global().Set("boardBudget", js.FuncOf(boardBudget))
	js.Global().Set("boardFootprint", js.FuncOf(boardFootprint))
	select {} // keep the runtime alive
}
