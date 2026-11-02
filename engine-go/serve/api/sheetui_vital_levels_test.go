package api

import (
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// A conta de PV e PM mora num diálogo próprio, aberto pela linha do diálogo do valor
// (decisão do dono): o campo fica sempre colado no número que ele muda.
func TestTheVitalDialogOpensTheLevelByLevelAccount(t *testing.T) {
	f, id := arcanistaDeNivel(t, 3)
	page := html.UnescapeString(f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String())

	dialog := func(dialogID string) string {
		t.Helper()
		found := regexp.MustCompile(`(?s)<dialog[^>]*id="` + dialogID + `".*?</dialog>`).FindString(page)
		if found == "" {
			t.Fatalf("a ficha não desenhou o diálogo %q", dialogID)
		}
		return found
	}
	for _, which := range []string{"pv", "pm"} {
		if !strings.Contains(dialog("vital-"+which), `commandfor="vital-`+which+`-levels"`) {
			t.Errorf("o diálogo de %s não abre a conta", which)
		}
	}

	// Arcanista 3, Con 2: 8 + 2 no 1º nível, e 2 + 2 nos outros (p37).
	pv := dialog("vital-pv-levels")
	for _, want := range []string{"Como chega em 18", "Nível 1 · Arcanista", "8 da classe + 2 de Constituição", "Nível 3 · Arcanista", "2 da classe + 2 de Constituição"} {
		if !strings.Contains(pv, want) {
			t.Errorf("a conta de PV não diz %q:\n%s", want, pv)
		}
	}
	// Int 4, pelo caminho do bruxo: a Inteligência entra uma vez, no 1º nível.
	pm := dialog("vital-pm-levels")
	for _, want := range []string{"6 da classe + 4 de Inteligência", "Nível 2 · Arcanista"} {
		if !strings.Contains(pm, want) {
			t.Errorf("a conta de PM não diz %q:\n%s", want, pm)
		}
	}
}
