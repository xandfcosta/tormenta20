package catalog

import (
	"encoding/json"
	"testing"
)

// itemsOfCategory lê do `items.json` só as entradas de UMA categoria.
//
// Duas fases, e a primeira é a razão de existir: decodificar o arquivo inteiro
// numa struct só quebra na primeira entrada de OUTRA espécie que discorde do
// tipo — o `unmodeled` é texto no encanto e um mapa por metade no material, e
// um guarda que decodificasse tudo morria num item que ele nem vinha medir,
// com uma mensagem sobre JSON em vez de sobre a regra.
//
// @example materiais := itemsOfCategory[specialMaterial](t, "material")
func itemsOfCategory[T any](t *testing.T, category string) []T {
	t.Helper()
	raw, err := files.ReadFile("data/items.json")
	if err != nil {
		t.Fatalf("ler items.json: %v", err)
	}
	var crus []json.RawMessage
	if err := json.Unmarshal(raw, &crus); err != nil {
		t.Fatalf("items.json não é uma lista JSON: %v", err)
	}
	achados := []T{}
	for _, cru := range crus {
		var espiada struct {
			Category string `json:"category"`
			Name     string `json:"name"`
		}
		if err := json.Unmarshal(cru, &espiada); err != nil {
			t.Fatalf("item sem categoria legível: %v", err)
		}
		if espiada.Category != category {
			continue
		}
		var entrada T
		if err := json.Unmarshal(cru, &entrada); err != nil {
			t.Fatalf("a entrada %q da categoria %q não cabe na forma esperada: %v",
				espiada.Name, category, err)
		}
		achados = append(achados, entrada)
	}
	if len(achados) == 0 {
		t.Fatalf("nenhuma entrada em %q: não há o que medir, e verde aqui não "+
			"valeria nada", category)
	}
	return achados
}
