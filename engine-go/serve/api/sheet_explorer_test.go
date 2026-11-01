package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// O EXPLORADOR (caçador, p50; ALE-503): um terreno no 3º nível e mais um a cada
// quatro; repetir soma +2; a área de Tormenta só a partir do 11º.

const explorerPath = "escolha/class.cacador.explorador/"

func hunter(t *testing.T, level int64) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	return f, seedCharacterAtLevel(t, f.s, f.player, "Caçadora", "Caçador", level, 0, 0)
}

func TestTheExplorerPicksATerrainPerFourLevels(t *testing.T) {
	f, id := hunter(t, 7)

	if step := choiceStepTitled(t, powerScreen(t, f, id), "Caçador · Explorador"); step.Tag != "0 de 2" {
		t.Errorf("a caçadora de nível 7 sem terreno não tem a pendência de dois: o passo diz %q", step.Tag)
	}
	if refusal := powerCommand(t, f, id, explorerPath+"area-de-tormenta", ""); !strings.Contains(refusal, "11") {
		t.Errorf("a área de Tormenta foi aceita no nível 7, ou a recusa não diz o nível: %q", refusal)
	}
	for range 2 {
		if refusal := powerCommand(t, f, id, explorerPath+"floresta", ""); refusal != "" {
			t.Fatalf("escolher floresta: %q", refusal)
		}
	}
	if refusal := powerCommand(t, f, id, explorerPath+"urbano", ""); !strings.Contains(refusal, "2") {
		t.Errorf("a terceira escolha no nível 7 foi aceita: %q", refusal)
	}
	screen := powerScreen(t, f, id)
	for _, want := range []string{"2 de 2 · nível 7", "Floresta ×2"} {
		if !strings.Contains(screen, want) {
			t.Errorf("a tela não tem %q", want)
		}
	}
	effects := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=conditionals", id), "").Body.String()
	if !strings.Contains(effects, "no terreno: Floresta") {
		t.Error("a aba Efeitos não tem o interruptor da floresta")
	}

	// O segundo terreno veio do 7º nível, e sai com ele (p50).
	rec := f.requests(t, f.player, http.MethodPost,
		fmt.Sprintf("/personagens/%d/nivel/%s/-1", id, url.PathEscape("Caçador")), "")
	if refusal := sceneRefusal(rec.Body.String()); refusal != "" {
		t.Fatalf("descer para o nível 6 foi recusado: %q", refusal)
	}
	if _, picks := storedChoices(t, f.s, id); strings.Count(picks, "floresta") != 1 {
		t.Errorf("no nível 6 fica um terreno, e ficou %s", picks)
	}
}
