package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// A ABA PROFICIÊNCIAS DA FICHA PARALELA (ALE-473), pelo mockup aprovado: as
// duas famílias como listas; a que tem origem fica travada, com a origem
// escrita, e a outra tem o interruptor; e o Restaurar padrão em cima.
func TestTheParallelProficienciesTabLocksWhatHasAnOrigin(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Lia", "Guerreiro", 3, 0, 0)
	// A semente não grava proficiência; o Restaurar padrão grava o que a
	// classe concede, e só a linha que o personagem TEM fica travada.
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/proficiencias/padrao?tab=proficiencies&tempestade=1", id), "").Body.String()); refusal != "" {
		t.Fatalf("restaurar o padrão: %s", refusal)
	}
	page := parallelSheet(t, f, id, "proficiencies")
	if strings.Contains(page, "ainda está na ficha de hoje") {
		t.Fatal("a aba Proficiências paralela ainda leva à ficha de hoje")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	tab := findNode(doc, "aria-label", "Proficiências")
	if tab == nil {
		t.Fatal("a ficha paralela não desenhou a aba Proficiências")
	}
	text, rendered := textOf(tab), renderNode(t, tab)
	for _, want := range []string{"Armas", "Armaduras & Escudos", "Armas marciais", "Restaurar padrão"} {
		if !strings.Contains(text, want) {
			t.Errorf("a aba não mostra %q", want)
		}
	}
	// O guerreiro tem as marciais pela classe: travada, sem interruptor.
	if regexp.MustCompile(`role="switch"[^>]*aria-label="Armas marciais"|aria-label="Armas marciais"[^>]*role="switch"`).MatchString(rendered) {
		t.Error("a linha das armas marciais, que vem da classe, tem interruptor")
	}
	if !strings.Contains(text, "Guerreiro") {
		t.Error("a linha travada não diz de onde vem")
	}
	// As exóticas não vêm de nada: o interruptor alterna, na ficha paralela.
	if !regexp.MustCompile(`<input[^>]*role="switch"[^>]*aria-label="Armas exóticas"[^>]*data-on:change="[^"]*/proficiencias/alterna/[^"]*tempestade=1`).MatchString(rendered) {
		t.Error("as armas exóticas não têm o interruptor que alterna")
	}
}
