package api

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// A ABA MOCHILA DA FICHA PARALELA (ALE-473), pelo mockup aprovado: o que está
// em uso em cima (mãos e vestidos, com o teto e os lugares livres), o guardado
// como lista com a busca sempre à vista, cada item com a folha dele, o
// consumível com o Usar, e um "Adicionar item" só.
func TestTheParallelBagTabShowsWhatIsInUseAndStowed(t *testing.T) {
	f, id := fighterFixture(t)
	itemSemeia(t, f, id, "espada-longa", "Espada longa", "wielded")
	itemSemeia(t, f, id, "armadura-completa", "Armadura completa", "vested")
	balm := itemSemeia(t, f, id, "balsamo-restaurador", "Bálsamo restaurador", "")
	dagger := itemSemeia(t, f, id, "adaga", "Adaga", "")
	page := parallelSheet(t, f, id, "bag")
	if strings.Contains(page, "ainda está na ficha de hoje") {
		t.Fatal("a aba Mochila paralela ainda leva à ficha de hoje")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	tab := findNode(doc, "aria-label", "Mochila")
	if tab == nil {
		t.Fatal("a ficha paralela não desenhou a aba Mochila")
	}
	text := textOf(tab)
	for what, want := range map[string]string{
		"o teto das mãos":            "1 / 2",
		"o teto dos vestidos":        "1 / 4",
		"os lugares livres":          "3 lugares livres",
		"a arma na mão":              "Espada longa",
		"a armadura vestida":         "Armadura completa",
		"o guardado":                 "Adaga",
		"o único gesto de adicionar": "Adicionar item",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("a aba não mostra %s (%q)", what, want)
		}
	}
	if strings.Contains(text, "Catálogo") || strings.Contains(text, "Custom") {
		t.Error("a aba ainda separa Catálogo e Custom")
	}
	rendered := renderNode(t, tab)
	balmSheet, daggerSheet := "item-"+strconv.FormatInt(balm, 10), "item-"+strconv.FormatInt(dagger, 10)
	for what, pattern := range map[string]string{
		"o Usar do consumível abrindo a folha dele": `aria-label="Usar Bálsamo restaurador"[^>]*commandfor="` + balmSheet + `"|commandfor="` + balmSheet + `"[^>]*aria-label="Usar Bálsamo restaurador"`,
		"o ⋯ da adaga abrindo a folha dela":         `commandfor="` + daggerSheet + `"`,
		"o adicionar abrindo o diálogo":             `commandfor="item-add"`,
	} {
		if !regexp.MustCompile(pattern).MatchString(rendered) {
			t.Errorf("a aba não tem %s", what)
		}
	}
	if !insideFilterBar(t, page, "Mochila", "bag-search", "") {
		t.Error("a busca do guardado não está na barra de filtros")
	}

	// A FOLHA DO ITEM é um diálogo do kit, e o lugar de equipar grava e fecha.
	sheet := findNode(doc, "id", daggerSheet)
	if sheet == nil || sheet.Data != "dialog" {
		t.Fatal("a adaga não tem a folha no diálogo do kit")
	}
	if !regexp.MustCompile(`command="close"[^>]*data-on:click="[^"]*/itens/` + strconv.FormatInt(dagger, 10) + `/equipa/[^"]*tempestade=1`).MatchString(renderNode(t, sheet)) {
		t.Error("a folha da adaga não empunha e fecha no mesmo toque")
	}
}
