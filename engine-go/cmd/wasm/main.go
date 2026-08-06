//go:build js && wasm

// WASM entrypoint — exposes ComputeCharacterSheet to the browser so the front
// runs the SAME Go rules as the server (single source), no TS duplication.
package main

import (
	"encoding/json"
	"syscall/js"

	"t20engine/engine"
)

func computeSheet(_ js.Value, args []js.Value) any {
	var in engine.CharacterInput
	if err := json.Unmarshal([]byte(args[0].String()), &in); err != nil {
		return map[string]any{"error": err.Error()}
	}
	out, _ := json.Marshal(engine.ComputeCharacterSheet(&in))
	return string(out)
}

func main() {
	js.Global().Set("computeCharacterSheet", js.FuncOf(computeSheet))
	select {} // keep the runtime alive
}
