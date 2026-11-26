package master

import (
	"strings"
	"testing"

	"t20engine/serve/web/bookui"
)

// A aba Perigos (p317-319): o cartão de cada tipo diz o que a mesa usa, e o filtro separa os tipos.

func TestTheHazardCardSaysWhatTheTableUses(t *testing.T) {
	body := pedeNaCenaComLivro(t, "/mestre/perigos").Body.String()
	needle := cardOf(t, body, "Agulha Envenenada")
	for _, want := range []string{"Armadilha", "ND 1/4", "Resistência: Reflexos CD 20 evita", "Investigação CD 25 · Ladinagem CD 20", "1d12 perda de vida por veneno"} {
		if !strings.Contains(needle, want) {
			t.Errorf("o cartão da Agulha Envenenada não diz %q: %s", want, needle)
		}
	}
	if !strings.Contains(cardOf(t, body, "Runa de Proteção"), "Mágica") {
		t.Error("o cartão da Runa de Proteção não diz que ela é mágica (o nome em itálico, p317)")
	}
	chill := cardOf(t, body, "Calafrio Diabólico")
	for _, want := range []string{"Doença", "Contato · Fortitude CD 25", "fraco / debilitado / inconsciente / morre"} {
		if !strings.Contains(chill, want) {
			t.Errorf("o cartão do Calafrio Diabólico não diz %q: %s", want, chill)
		}
	}
	// O clima mora no Capítulo 6, e o cartão leva ao livro na página dele.
	if !strings.Contains(cardOf(t, body, "Clima, Calor e Frio"), "Abrir o livro na página 267") {
		t.Error("o cartão do clima não leva à p267")
	}
}

func TestTheHazardFilterSeparatesTheKinds(t *testing.T) {
	for kind, want := range map[string]int{"armadilha": 23, "doenca": 8, "ambiente": 12} {
		v := loadCollection(collectionCriteria{Aba: "perigos", Filters: map[string][]string{"tipo": {kind}}}, bookui.BookAddress{})
		if v.Findings != want {
			t.Errorf("o filtro %q deu %d perigos, e o livro tem %d", kind, v.Findings, want)
		}
	}
}

func TestTheUnifiedSearchFindsTheHazard(t *testing.T) {
	v := loadCollection(collectionCriteria{Term: "variola"}, bookui.BookAddress{})
	for _, g := range v.Groups {
		if g.Label == "Perigos" && len(g.Hazards) > 0 {
			return
		}
	}
	t.Error("a busca por \"variola\" não achou a doença na aba Perigos")
}

// cardOf é o cartão do verbete, do nome ao fim do <article>.
func cardOf(t *testing.T, body, name string) string {
	t.Helper()
	_, card, found := strings.Cut(body, ">"+name+"<")
	if !found {
		t.Fatalf("o cartão de %q não apareceu", name)
	}
	card, _, _ = strings.Cut(card, "</article>")
	return card
}
