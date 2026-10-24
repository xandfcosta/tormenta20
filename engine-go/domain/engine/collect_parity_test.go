package engine

import (
	"path/filepath"
	"reflect"
	"testing"

	"t20engine/domain/catalog"
)

// TestActiveItemsParity proves the ported collection layer (ActiveItemsFor) on
// real data: for each seed character it primes the embedded catalogs,
// re-collects the raw Character, and asserts the []ActiveItem match o oráculo
// (`activeItems`, gerado por `go run ./cmd/genoracle`) semantically. This is
// slice 2's target — the resolution test (slice 1) covers the downstream half.
//
// Regenere o oráculo quando a regra mudar:
//
//	cd engine-go && go run ./cmd/genoracle
func TestActiveItemsParity(t *testing.T) {
	dir := filepath.Clean(filepath.Join(mustWd(t), "..", "..", "parity"))
	catalogs := primeBook(t)

	slugs := parityOracleSlugs(t, dir)

	for _, slug := range slugs {
		slug := slug
		t.Run(slug, func(t *testing.T) {
			var oracle struct {
				Char        Character `json:"char"`
				ActiveItems any       `json:"activeItems"`
			}
			readJSON(t, filepath.Join(dir, slug), &oracle)

			got := roundTrip(t, BookRuleset(catalogs).ActiveItemsFor(oracle.Char))
			if !reflect.DeepEqual(got, oracle.ActiveItems) {
				diffReport(t, "activeItems", got, oracle.ActiveItems)
			}
		})
	}
}

// primeBook prima o motor com o catálogo embutido — o mesmo que o servidor
// carrega. Um motor novo por chamada, para nenhum teste ver o que outro fez.
//
// É o único import de `catalog` no pacote, e é de TESTE: em produção quem junta
// os dois é o `book.EngineCatalogs`, que este pacote não pode importar.
func primeBook(t *testing.T) *Catalogs {
	t.Helper()
	catalogs, err := PrimeEngineCatalogs(catalog.EnginePayload())
	if err != nil {
		t.Fatalf("primar o catálogo embutido: %v", err)
	}
	return catalogs
}
