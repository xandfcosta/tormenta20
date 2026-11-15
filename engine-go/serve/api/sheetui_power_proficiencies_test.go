package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// A PROFICIÊNCIA QUE O PODER DÁ NO PAINEL (ALE-649, decisão do dono). A categoria inteira aparece marcada e
// travada, dizendo de onde vem — o Pistoleiro dá as armas de fogo (p48) —, e a parte de uma categoria vira
// uma nota na linha dela ("Armeiro: as corpo a corpo"): o Armeiro sabe as armas marciais corpo a corpo (p68), e não as marciais todas.

func proficiencyPanel(t *testing.T, f sceneFixture, id int64) string {
	t.Helper()
	return f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=proficiencies", id), "").Body.String()
}

func TestThePowerProficiencyShowsInThePanel(t *testing.T) {
	f := newSceneFixture(t)
	gunner := seedCharacterAtLevel(t, f.s, f.player, "Pistoleira", "Bucaneiro", 5, 0, 0)
	saveHand(t, f, gunner, `["armas-simples"]`)
	seedPowerChoices(t, f.s, gunner, `["class.bucaneiro.pistoleiro"]`, `{}`)
	row := proficiencyRow(t, proficiencyPanel(t, f, gunner), "Armas de fogo")
	if !strings.Contains(row, "vem de Pistoleiro") {
		t.Errorf("a linha das armas de fogo não diz que vem do Pistoleiro:\n%s", row)
	}
	if strings.Contains(row, `role="switch"`) {
		t.Errorf("a linha das armas de fogo se desmarca na mão, e ela sai com o poder:\n%s", row)
	}

	smith := seedCharacterAtLevel(t, f.s, f.player, "Armeira", "Inventor", 5, 0, 0)
	saveHand(t, f, smith, `["armas-simples"]`)
	seedPowerChoices(t, f.s, smith, `["class.inventor.armeiro"]`, `{}`)
	if row := proficiencyRow(t, proficiencyPanel(t, f, smith), "Armas marciais"); !strings.Contains(row, "Armeiro: as corpo a corpo") {
		t.Errorf("a linha das armas marciais não diz a parte que o Armeiro dá:\n%s", row)
	}
}
