package api

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// A BARRA DE FILTROS fica presa no topo de quem rola (decisão do dono,
// ALE-473): a busca e os filtros de cada lista da ficha paralela moram dentro
// dela, na aba e no diálogo, para a lista rolar por baixo.

// insideFilterBar diz se o nó com o `id` (ou o grupo com o `aria-label`)
// dentro do escopo — um `id`, ou o `aria-label` da aba — mora numa
// `.ds-filter-bar`.
func insideFilterBar(t *testing.T, page, scope, id, groupLabel string) bool {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	within := findNode(doc, "id", scope)
	if within == nil {
		within = findNode(doc, "aria-label", scope)
	}
	target := findNode(within, "id", id)
	if groupLabel != "" {
		target = findNode(within, "aria-label", groupLabel)
	}
	if target == nil {
		t.Fatalf("não achei %q/%q dentro de #%s", id, groupLabel, scope)
	}
	for n := target.Parent; n != nil; n = n.Parent {
		if strings.Contains(" "+attr(n, "class")+" ", " ds-filter-bar ") {
			return true
		}
	}
	return false
}

func findNode(n *html.Node, key, value string) *html.Node {
	if n == nil {
		return nil
	}
	if n.Type == html.ElementNode && attr(n, key) == value {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findNode(c, key, value); found != nil {
			return found
		}
	}
	return nil
}

func TestTheParallelSheetFiltersStayOnTop(t *testing.T) {
	f, barbarian, _ := effectsInPlay(t)
	g, arcanist := grimoireOfTwo(t)
	cases := []struct {
		what             string
		page             string
		scope, id, group string
	}{
		{"a busca de perícia", parallelSheet(t, f, barbarian, "expertises"), "expertise-search", "expertise-search", ""},
		{"a busca de poder", parallelSheet(t, f, barbarian, "abilities"), "power-search", "power-search", ""},
		{"a busca do grimório", parallelSheet(t, g, arcanist, "spells"), "grimoire-search", "grimoire-search", ""},
		{"o círculo do grimório", parallelSheet(t, g, arcanist, "spells"), "Magias", "", "Círculo"},
		{"a busca do Aprender", parallelSheet(t, g, arcanist, "spells"), "spell-learn", "spell-learn-search", ""},
		{"a escola do Aprender", parallelSheet(t, g, arcanist, "spells"), "spell-learn", "", "Escola"},
		{"a busca de condição", parallelSheet(t, f, barbarian, "conditionals"), "condition-picker", "condition-search", ""},
		{"a busca de magia aplicável", parallelSheet(t, f, barbarian, "conditionals"), "buff-picker", "buff-search", ""},
		{"o círculo da magia aplicável", parallelSheet(t, f, barbarian, "conditionals"), "buff-picker", "", "Círculo"},
	}
	for _, c := range cases {
		if !insideFilterBar(t, c.page, c.scope, c.id, c.group) {
			t.Errorf("%s não está na barra de filtros", c.what)
		}
	}
}

// NO DIÁLOGO A BARRA É FIXA e só a lista rola: com a barra dentro do que rola,
// a barra de rolagem descia ao lado dos filtros (decisão do dono, ALE-473).
func TestTheParallelDialogFiltersStayOutsideTheScroll(t *testing.T) {
	f, barbarian, _ := effectsInPlay(t)
	g, arcanist := grimoireOfTwo(t)
	cases := []struct{ what, page, dialog string }{
		{"o Aprender", parallelSheet(t, g, arcanist, "spells"), "spell-learn"},
		{"o escolhedor de condição", parallelSheet(t, f, barbarian, "conditionals"), "condition-picker"},
		{"o escolhedor de magia", parallelSheet(t, f, barbarian, "conditionals"), "buff-picker"},
	}
	for _, c := range cases {
		doc, err := html.Parse(strings.NewReader(c.page))
		if err != nil {
			t.Fatal(err)
		}
		dialog := findNode(doc, "id", c.dialog)
		scroll, bar := findNode(dialog, "class", "ds-dialog-scroll"), findNode(dialog, "class", "ds-filter-bar")
		if scroll == nil || bar == nil {
			t.Errorf("%s: falta a parte que rola (%v) ou a barra (%v)", c.what, scroll != nil, bar != nil)
			continue
		}
		if findNode(scroll, "class", "ds-filter-bar") != nil {
			t.Errorf("%s: a barra de filtros mora dentro da parte que rola", c.what)
		}
		if findNode(scroll, "class", "ds-list") == nil {
			t.Errorf("%s: a lista não mora na parte que rola", c.what)
		}
	}
}
