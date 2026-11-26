package api

import (
	"strings"
	"testing"
)

// O CARTÃO FLUTUANTE (ver GLOSSARY, **cartão flutuante**): a ação em que o mestre escolhe é um cartão que se
// arrasta pela alça e se recolhe pela seta.

// sectionOf é o `<section id="…">` da página, do id ao fecho; vazio quando a página não o tem. Os cartões não
// aninham `<section>`, e o primeiro fecho é o dele.
func sectionOf(page, id string) string {
	start := strings.Index(page, `<section id="`+id+`"`)
	if start < 0 {
		return ""
	}
	end := strings.Index(page[start:], "</section>")
	if end < 0 {
		return page[start:]
	}
	return page[start : start+end]
}

// requireFloatingCard cobra do cartão a alça visível e a seta de recolher. Que o recolhido sobrevive ao remendo é
// do navegador, e quem prova é o `float-dock.spec.ts`.
func requireFloatingCard(t *testing.T, card, what string) {
	t.Helper()
	if card == "" {
		t.Fatalf("a página não tem o cartão %s", what)
	}
	for _, want := range []string{`data-float`, `data-float-grip`, `data-float-fold`} {
		if !strings.Contains(card, want) {
			t.Errorf("o cartão %s não tem %s", what, want)
		}
	}
}
