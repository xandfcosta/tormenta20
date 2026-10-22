package catalog

import (
	"encoding/json"
	"testing"
)

// TODA POÇÃO APONTA PARA UMA MAGIA QUE O LIVRO TEM (p341, ALE-451).
//
// Uma poção não carrega regra: ela carrega um PONTEIRO — *"contêm o efeito de
// uma magia"*. Um id de magia torto não dá erro em lugar nenhum: a ficha do
// item mostraria "contém" e o nome do id cru, que é a forma mais silenciosa de
// um catálogo mentir.
//
// E o `kind` decide a AÇÃO de ativar (beber, aplicar num objeto, arremessar),
// enquanto o `scope` é o que a tela escreve — "imediato", "1 cena", "1 dia".
// Os dois são vocabulário fechado, e um valor novo entra reprovando pelo nome.
func TestEveryPotionHoldsASpellTheBookHas(t *testing.T) {
	type potion struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Consumable *struct {
			Scope   string `json:"scope"`
			Spell   string `json:"spell"`
			Kind    string `json:"kind"`
			Augment string `json:"augment"`
		} `json:"consumable"`
	}
	type spell struct {
		ID       string `json:"id"`
		Duration string `json:"duration"`
	}

	// O `spells.json` é um OBJETO por id e o `items.json` é uma LISTA: os dois
	// catálogos guardam a mesma espécie de coisa em formas diferentes, e quem
	// lê precisa saber de cada um.
	magias := mapOf[spell](t, "spells")
	frascos := map[string]bool{"pocao": true, "oleo": true, "granada": true}
	janelas := map[string]bool{"instant": true, "scene": true, "day": true}

	medidas := 0
	for _, item := range listOf[potion](t, "items") {
		if item.Consumable == nil || item.Consumable.Spell == "" {
			continue
		}
		medidas++
		magia, existe := magias[item.Consumable.Spell]
		if !existe {
			t.Errorf("%s contém a magia %q, e o livro não tem magia com esse id",
				item.Name, item.Consumable.Spell)
		}
		if !frascos[item.Consumable.Kind] {
			t.Errorf("%s é do tipo %q, e o livro tem três: poção, óleo e granada "+
				"(p341)", item.Name, item.Consumable.Kind)
		}
		if !janelas[item.Consumable.Scope] {
			t.Errorf("%s tem janela %q, e a ficha só sabe escrever imediato, 1 cena "+
				"e 1 dia", item.Name, item.Consumable.Scope)
		}
		// A JANELA SEGUE A DURAÇÃO da magia, e a exceção é o ÓLEO: ele age
		// sobre um objeto, e não abre janela de efeito em quem o usa — a Tranca
		// Arcana é permanente na porta e instantânea para o frasco.
		if existe && item.Consumable.Scope == "instant" &&
			magia.Duration != "instantanea" && item.Consumable.Kind != "oleo" {
			t.Errorf("%s é imediata e a magia dela dura %q: ou a janela está "+
				"errada, ou falta dizer por que ela não segue a magia",
				item.Name, magia.Duration)
		}
	}
	if medidas == 0 {
		t.Fatal("nenhuma poção no catálogo: este guarda não mediu nada")
	}
	t.Logf("poções medidas: %d", medidas)
}

// mapOf decodifica um recurso guardado como OBJETO por id.
func mapOf[T any](t *testing.T, resource string) map[string]T {
	t.Helper()
	raw, err := files.ReadFile("data/" + resource + ".json")
	if err != nil {
		t.Fatalf("ler %s.json: %v", resource, err)
	}
	var byID map[string]T
	if err := json.Unmarshal(raw, &byID); err != nil {
		t.Fatalf("%s.json: %v", resource, err)
	}
	return byID
}

// listOf decodifica um recurso embutido numa lista da forma pedida.
//
// Como o `itemsOfCategory`, ela existe porque decodificar o arquivo inteiro
// numa struct só quebra na primeira entrada de outra espécie — e aqui a forma
// pedida é SEMPRE parcial, com os campos que a pergunta usa.
func listOf[T any](t *testing.T, resource string) []T {
	t.Helper()
	raw, err := files.ReadFile("data/" + resource + ".json")
	if err != nil {
		t.Fatalf("ler %s.json: %v", resource, err)
	}
	var lista []T
	if err := json.Unmarshal(raw, &lista); err != nil {
		t.Fatalf("%s.json: %v", resource, err)
	}
	return lista
}
