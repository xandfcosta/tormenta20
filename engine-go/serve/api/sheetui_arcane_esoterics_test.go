package api

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"testing"
)

// OS ESOTÉRICOS ARCANOS SÓ VALEM NAS MAGIAS ARCANAS (p160): a varinha arcana dá "+1 na CD para resistir a
// suas magias arcanas", e o cajado arcano, o mesmo e +1 no limite de PM. Na mão de um druida, que conjura
// magia divina, eles não mudam número nenhum.

var spellDCOnTheSheet = regexp.MustCompile(`CD <b>(\d+)</b>`)

func spellDCOf(t *testing.T, f sceneFixture, id int64) int {
	t.Helper()
	found := spellDCOnTheSheet.FindStringSubmatch(spellScreen(t, f, id))
	if found == nil {
		t.Fatal("a aba Magias não escreveu a CD")
	}
	dc, _ := strconv.Atoi(found[1])
	return dc
}

func TestTheArcaneWandRaisesOnlyArcaneDCs(t *testing.T) {
	for _, c := range []struct {
		class string
		gain  int
	}{
		{"Arcanista", 1},
		{"Druida", 0},
	} {
		t.Run(c.class, func(t *testing.T) {
			f := newSceneFixture(t)
			id := seedCharacterAtLevel(t, f.s, f.player, "Conjurador "+c.class, c.class, 5, 0, 0)
			before := spellDCOf(t, f, id)
			itemSemeia(t, f, id, "varinha-arcana", "Varinha arcana", "wielded")
			if got := spellDCOf(t, f, id) - before; got != c.gain {
				t.Errorf("a varinha arcana mudou a CD do %s em %+d, e o livro dá %+d", c.class, got, c.gain)
			}
			combat := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=combat", id), "").Body.String()
			if c.gain == 0 && regexp.MustCompile(`Varinha arcana[^<]*</[^>]+>\s*<[^>]+>\+1`).MatchString(combat) {
				t.Errorf("a caixa CD Magia do %s ainda soma a varinha", c.class)
			}
		})
	}
}
