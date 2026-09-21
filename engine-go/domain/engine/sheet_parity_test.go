package engine

import (
	"path/filepath"
	"reflect"
	"testing"
)

// A camada de decomposição (`ComputeSheet`) sobre dado REAL: para cada
// personagem da semente, prima os catálogos, computa a ficha inteira e afirma
// que toda decomposição bate com o oráculo (`sheet`) semanticamente.
//
// Cada personagem é conferido DUAS vezes: sem nenhum condicional ligado, e com
// todos ligados. A segunda passada é a única cobertura de oráculo do
// `ApplyActiveConditionals` — a dobra que reroda o `resolveStack` por alvo —, e
// ela morde no `bardo-versatil-nv7`, cujos dois condicionais de Inspiração caem
// no mesmo alvo com o mesmo `bonusType`: +1 e +2 têm de resolver para +2 nas 29
// perícias, e não para +3.
//
// Regenere o oráculo quando a regra mudar:
//
//	cd engine-go && go run ./cmd/genoracle
func TestSheetParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeFromDump(t, dir)
	withConditionals := 0

	for _, slug := range parityOracleSlugs(t, dir) {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char               Character `json:"char"`
				Sheet              any       `json:"sheet"`
				ActiveConditionals []string  `json:"activeConditionals"`
				WithConditionals   any       `json:"sheetWithConditionals"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, catalogs.ComputeSheet(oracle.Char, map[string]bool{}))
			if !reflect.DeepEqual(got, oracle.Sheet) {
				diffReport(t, "sheet", got, oracle.Sheet)
			}

			// A segunda passada só EXISTE para exercitar `ApplyActiveConditionals`:
			// num personagem sem condicional nenhum ela repete a primeira e conta
			// como cobertura. Rodar só onde há o que ligar deixa claro quantos
			// realmente exercitam a dobra.
			if len(oracle.ActiveConditionals) == 0 {
				return
			}
			withConditionals++
			on := toSet(oracle.ActiveConditionals)
			gotOn := roundTrip(t, catalogs.ComputeSheet(oracle.Char, on))
			if !reflect.DeepEqual(gotOn, oracle.WithConditionals) {
				diffReport(t, "sheetWithConditionals", gotOn, oracle.WithConditionals)
			}
		})
	}

	// A dobra dos condicionais é o que pega a Inspiração dupla do bardo: se um
	// dia NENHUM oráculo tiver condicional, a cobertura dela vira zero em
	// silêncio — e é isso que esta linha impede.
	if withConditionals == 0 {
		t.Error("nenhum oráculo exercitou ApplyActiveConditionals — a dobra ficou sem prova")
	}
}
