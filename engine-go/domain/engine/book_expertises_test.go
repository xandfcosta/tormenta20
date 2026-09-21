package engine

import (
	"encoding/json"
	"testing"

	"t20engine/domain/catalog"
)

// TestTheArmorPenaltyAgreesWithTheBook (ALE-264).
//
// O motor tem as três perícias de penalidade de armadura escritas à mão desde o
// porte do `derived.ts`. O catálogo de perícias, que nasceu agora, traz a MESMA
// regra colhida da Tabela 2-1 (p115) do livro.
//
// As duas fontes concordam hoje. Este teste é a costura que impede a divergência
// silenciosa: se alguém corrigir a tabela do catálogo e esquecer o motor — ou o
// contrário —, a ficha passaria a calcular com uma regra e a tela a mostrar
// outra. É a mesma família do `TestDumpAgreesWithEmbeddedCatalog`.
//
// Vive em `engine/` e não em `api/` porque o dono da regra é o motor: é ele que
// aplica a penalidade no cálculo.
func TestTheArmorPenaltyAgreesWithTheBook(t *testing.T) {
	raw, ok := catalog.Resource("expertises")
	if !ok {
		t.Fatal("catálogo de perícias ausente")
	}
	var expertises []struct {
		Name         string `json:"name"`
		ArmorPenalty bool   `json:"penalidadeDeArmadura"`
	}
	if err := json.Unmarshal(raw, &expertises); err != nil {
		t.Fatalf("perícias: %v", err)
	}
	if len(expertises) != 29 {
		t.Fatalf("%d perícias — o livro tem 29", len(expertises))
	}

	fromBook := map[string]bool{}
	for _, p := range expertises {
		if p.ArmorPenalty {
			fromBook[p.Name] = true
		}
	}
	if len(fromBook) != len(armorPenaltyExpertises) {
		t.Fatalf("o livro marca %d perícias com penalidade e o motor conhece %d",
			len(fromBook), len(armorPenaltyExpertises))
	}
	for name := range armorPenaltyExpertises {
		if !fromBook[name] {
			t.Errorf("o motor aplica penalidade de armadura em %q e a Tabela 2-1 não marca", name)
		}
	}
}
