package catalog

import (
	"encoding/json"
	"testing"
)

// TODA CONDIÇÃO ANUNCIADA EXISTE NA TABELA DE CONDIÇÕES (p394, ALE-441).
//
// O anúncio leva um ID, e quem o traduz para a tela é o `book.ConditionName` —
// que, não achando, devolve o próprio id. Então um `sangrand` digitado errado
// não estoura em lugar nenhum: a mesa lê "sangrand" na faixa do ataque e
// conclui que o app escreve mal.
//
// É a mesma família do `TestEveryReferencedElementIdExists`: ponteiro que não
// aponta para nada, e nada mais acusa.
func TestEveryAnnouncedConditionExistsInTheConditionTable(t *testing.T) {
	condicoes := map[string]json.RawMessage{}
	raw, err := files.ReadFile("data/conditions.json")
	if err != nil {
		t.Fatalf("ler conditions.json: %v", err)
	}
	if err := json.Unmarshal(raw, &condicoes); err != nil {
		t.Fatalf("conditions.json: %v", err)
	}
	if len(condicoes) < 20 {
		t.Fatalf("só %d condições no catálogo: o guarda mediria contra quase nada",
			len(condicoes))
	}

	itens, err := files.ReadFile("data/items.json")
	if err != nil {
		t.Fatalf("ler items.json: %v", err)
	}
	var lidos []struct {
		Name      string `json:"name"`
		Modifiers []struct {
			Target struct {
				K    string `json:"k"`
				Name string `json:"name"`
			} `json:"target"`
		} `json:"modifiers"`
	}
	if err := json.Unmarshal(itens, &lidos); err != nil {
		t.Fatalf("items.json: %v", err)
	}

	medidos := 0
	for _, item := range lidos {
		for _, m := range item.Modifiers {
			if m.Target.K != "announcedCondition" {
				continue
			}
			medidos++
			if _, existe := condicoes[m.Target.Name]; !existe {
				t.Errorf("%s anuncia a condição %q, que não está na tabela de condições — "+
					"a mesa leria o id cru na faixa do ataque", item.Name, m.Target.Name)
			}
		}
	}
	// O DENOMINADOR: cinco encantos anunciam hoje, e uma varredura que parasse
	// de achá-los passaria verde sobre zero.
	if medidos < 5 {
		t.Fatalf("o guarda achou só %d anúncios: ele parou de achar o que veio medir",
			medidos)
	}
	t.Logf("anúncios medidos: %d", medidos)
}
