package api

import (
	"context"
	"encoding/json"
	"fmt"
	"t20engine/book"
	"t20engine/plataforma"
	"t20engine/sheet"

	"t20engine/db/sqlcgen"
)

// saveProficiencies valida contra o catálogo, tira as repetidas e grava.
//
// Extraída na ALE-272 (fatia 2) porque a ficha em Datastar passou a gravar
// proficiencia pelo mesmo caminho, e a regra e uma so: o painel do piloto e o
// `PATCH /characters/{id}/proficiencies` guardam com a MESMA validacao. Duas
// copias divergiriam no dia em que uma categoria nova chegasse — e a copia
// esquecida aceitaria o que a outra recusa.
//
// Devolve a lista de DESCONHECIDAS separada do erro porque as duas respostas sao
// diferentes: categoria invalida e 422 com o campo, falha de banco e 500.
func (sr sheetRules) saveProficiencies(
	ctx context.Context, id int64, categorias []string,
) (string, []string, error) {
	var unknown []string
	seen := map[string]bool{}
	dedup := []string{}
	for _, cat := range categorias {
		if !book.IsProficiencyCategory(cat) {
			unknown = append(unknown, fmt.Sprintf("Unknown category %q", cat))
		}
		if !seen[cat] {
			seen[cat] = true
			dedup = append(dedup, cat)
		}
	}
	if len(unknown) > 0 {
		return "", unknown, nil
	}
	proficiencies := sheet.MarshalStrings(&dedup)
	if err := sr.queries.SetProficiencies(ctx, sqlcgen.SetProficienciesParams{
		Proficiencies: proficiencies, UpdatedAt: plataforma.NowISO(), ID: id,
	}); err != nil {
		return "", nil, err
	}
	return proficiencies, nil, nil
}

// compactJSON normalizes an object blob to compact JSON (matching JSON.stringify
// of the sanitized value, minus the deferred sanitization).
func compactJSON(raw json.RawMessage) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return "{}"
	}
	b, _ := json.Marshal(v)
	return string(b)
}
