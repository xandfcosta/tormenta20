package engine

import (
	"encoding/json"
	"testing"

	"t20engine/catalog"
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
	bruto, ok := catalog.Resource("pericias")
	if !ok {
		t.Fatal("catálogo de perícias ausente")
	}
	var pericias []struct {
		Name                 string `json:"name"`
		PenalidadeDeArmadura bool   `json:"penalidadeDeArmadura"`
	}
	if err := json.Unmarshal(bruto, &pericias); err != nil {
		t.Fatalf("perícias: %v", err)
	}
	if len(pericias) != 29 {
		t.Fatalf("%d perícias — o livro tem 29", len(pericias))
	}

	doLivro := map[string]bool{}
	for _, p := range pericias {
		if p.PenalidadeDeArmadura {
			doLivro[p.Name] = true
		}
	}
	if len(doLivro) != len(armorPenaltyExpertises) {
		t.Fatalf("o livro marca %d perícias com penalidade e o motor conhece %d",
			len(doLivro), len(armorPenaltyExpertises))
	}
	for nome := range armorPenaltyExpertises {
		if !doLivro[nome] {
			t.Errorf("o motor aplica penalidade de armadura em %q e a Tabela 2-1 não marca", nome)
		}
	}
}
