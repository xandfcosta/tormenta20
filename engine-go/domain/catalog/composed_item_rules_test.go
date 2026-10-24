package catalog

import (
	"encoding/json"
	"testing"
)

// TODO ITEM COMPOSTO DIZ O QUE FAZ, OU DIZ POR QUE NÃO (p337, p340, ALE-448).
//
// O verbete de um item específico tem duas metades: a COMPOSIÇÃO, que o
// `composedOf` resolve sozinho, e a REGRA PRÓPRIA — "fornece redução de fogo
// 10", "você pode gastar 6 PM para invocar uma tempestade de gelo". É a segunda
// que este guarda cobra.
//
// Ela entra de três jeitos, e um item que não usa NENHUM está pela metade em
// silêncio: modificador (o número), ativação (o gesto, no `activations.json`) ou
// a nota de `unmodeled` dizendo por que o motor não a aplica. Sem isso,
// "transcrevi os dezoito" e "transcrevi a composição de dezoito e a regra de
// cinco" têm a mesma cara no terminal.
//
// E o `modeled` é a outra ponta: quem tem número ou gesto diz QUAL metade
// entrou, porque a nota de `unmodeled` ao lado fala só da outra.
func TestEveryComposedItemSaysWhatItDoesOrWhyNot(t *testing.T) {
	type composedItem struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		Category    string          `json:"category"`
		Description string          `json:"description"`
		Modifiers   json.RawMessage `json:"modifiers"`
		Modeled     []string        `json:"modeled"`
		Unmodeled   string          `json:"unmodeled"`
		ComposedOf  json.RawMessage `json:"composedOf"`
	}
	type activation struct {
		ID string `json:"id"`
	}

	// DUAS FASES, como no `itemsOfCategory`: decodificar o arquivo inteiro numa
	// struct só quebra na primeira entrada de outra espécie que discorde do
	// tipo — o `unmodeled` é TEXTO aqui e no encanto, e um MAPA por metade no
	// material. Quem não é composto nem chega a ser decodificado.
	todos := []composedItem{}
	raw, err := resourceJSON("items")
	if err != nil {
		t.Fatalf("ler items.json: %v", err)
	}
	var crus []json.RawMessage
	if err := json.Unmarshal(raw, &crus); err != nil {
		t.Fatalf("items.json não é uma lista JSON: %v", err)
	}
	for _, cru := range crus {
		var espiada struct {
			Name       string          `json:"name"`
			ComposedOf json.RawMessage `json:"composedOf"`
		}
		if err := json.Unmarshal(cru, &espiada); err != nil {
			t.Fatalf("item sem forma legível: %v", err)
		}
		if len(espiada.ComposedOf) == 0 {
			continue
		}
		var item composedItem
		if err := json.Unmarshal(cru, &item); err != nil {
			t.Fatalf("o item composto %q não cabe na forma esperada: %v",
				espiada.Name, err)
		}
		todos = append(todos, item)
	}
	var ativacoes []activation
	rawAtivacoes, err := resourceJSON("activations")
	if err != nil {
		t.Fatalf("ler activations.json: %v", err)
	}
	if err := json.Unmarshal(rawAtivacoes, &ativacoes); err != nil {
		t.Fatalf("activations.json: %v", err)
	}
	temGesto := map[string]bool{}
	for _, a := range ativacoes {
		temGesto[a.ID] = true
	}

	comRegra, soNota := 0, 0
	for _, item := range todos {
		aplica := len(item.Modifiers) > 2 || temGesto["item."+item.ID]
		switch {
		case !aplica && item.Unmodeled == "":
			t.Errorf("%s não tem número, nem gesto, NEM diz por que não.\n"+
				"A regra própria do verbete entra de um dos três jeitos — sem "+
				"nenhum, o item entra com metade da regra e nada acusa.", item.Name)
		case aplica && len(item.Modeled) == 0:
			t.Errorf("%s aplica alguma coisa e não diz QUAL: sem o `modeled`, a nota "+
				"de `unmodeled` ao lado parece falar do item inteiro", item.Name)
		case aplica:
			comRegra++
		default:
			soNota++
		}
		if item.Description == "" {
			t.Errorf("%s entrou sem a regra do livro em prosa", item.Name)
		}
	}
	if comRegra+soNota == 0 {
		t.Fatal("nenhum item composto no catálogo: este guarda não mediu nada")
	}
	t.Logf("itens compostos medidos: %d — %d o motor aplica em parte, %d só dizem "+
		"por que não", comRegra+soNota, comRegra, soNota)
}
