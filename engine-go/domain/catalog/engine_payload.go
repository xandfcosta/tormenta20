package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// EnginePayload monta, a partir dos arquivos embutidos, o JSON que o
// `engine.PrimeEngineCatalogs` lê.
//
// O motor calculava a ficha de um despejo em disco (`parity/_catalogs.json`)
// mantido à mão ao lado destes arquivos; as duas cópias carregavam a mesma
// coisa e só um teste as comparava, e não em todas as chaves (ALE-494). Montar
// aqui faz do embutido a única fonte. A função devolve BYTES, e não os tipos do
// motor, porque `catalog` e `engine` não se conhecem — quem junta os dois é o
// `book.EngineCatalogs`.
//
//	catalogs, err := engine.PrimeEngineCatalogs(catalog.EnginePayload())
func EnginePayload() []byte {
	payloadOnce.Do(func() {
		payloadBytes, payloadErr = assembleEnginePayload()
	})
	if payloadErr != nil {
		// Recurso embutido que não lê é binário quebrado, e não há ficha certa
		// a servir a partir dele.
		panic(payloadErr)
	}
	return payloadBytes
}

var (
	payloadOnce  sync.Once
	payloadBytes []byte
	payloadErr   error
)

// enginePayloadKeys liga cada chave da entrada do motor ao recurso embutido
// que ela é. `races` é `race-defs` (as habilidades) e `racas` é `races` (a
// tabela de atributo por raça): os nomes cruzados vêm do despejo que existia e são a forma
// que o motor desserializa.
var enginePayloadKeys = map[string]string{
	"items":         "items",
	"races":         "race-defs",
	"origins":       "origins",
	"classPowers":   "class-powers",
	"generalPowers": "general-powers",
	"grantedPowers": "granted-powers",
	"racas":         "races",
}

func assembleEnginePayload() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(enginePayloadKeys)+1)
	for key, resource := range enginePayloadKeys {
		raw, err := resourceJSON(resource)
		if err != nil {
			return nil, fmt.Errorf("catalog.EnginePayload: recurso %q (chave %q): %w", resource, key, err)
		}
		out[key] = raw
	}
	// O motor só pergunta "este poder é da Tormenta?", então leva os ids.
	raw, err := resourceJSON("tormenta-powers")
	if err != nil {
		return nil, fmt.Errorf("catalog.EnginePayload: recurso \"tormenta-powers\": %w", err)
	}
	var byID map[string]json.RawMessage
	if err := json.Unmarshal(raw, &byID); err != nil {
		return nil, fmt.Errorf("catalog.EnginePayload: tormenta-powers.json devia ser um mapa por id: %w", err)
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	idsJSON, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	out["tormentaPowerIds"] = idsJSON
	return json.Marshal(out)
}
