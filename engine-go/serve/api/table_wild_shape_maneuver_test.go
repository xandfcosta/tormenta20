package api

import (
	"testing"

	"t20engine/app/combat"
	"t20engine/domain/live"
)

// O TAMANHO DA FORMA chega à manobra da Mesa. A Forma Sorrateira normal é Pequena (p63), e Pequeno dá −2
// nas manobras (Tab. 1-21, p106). A ficha já mostrava o −2; a Mesa montava os efeitos do combatente sem a
// forma, e a manobra saía com o número do druida em pé.
func TestTheWildShapeSizeReachesTheManeuverAtTheTable(t *testing.T) {
	f, id := wildDruid(t)
	roster := combat.NewRoster(f.s.queries, f.s.catalogs)
	entry := live.InitiativeEntry{ID: "druida", Label: "Druida da forma", CharacterID: &id}

	standing, err := roster.Of(t.Context(), 0, entry)
	if err != nil {
		t.Fatalf("montar o combatente em pé: %v", err)
	}
	shape := `{"wild_form":"sorrateira","wild_tier":"normal","wild_damage":"perfuracao","wild_move":""}`
	if refusal := powerCommand(t, f, id, "forma-selvagem", shape); refusal != "" {
		t.Fatalf("a transformação foi recusada: %q", refusal)
	}
	shaped, err := roster.Of(t.Context(), 0, entry)
	if err != nil {
		t.Fatalf("montar o combatente na forma: %v", err)
	}

	for _, side := range []struct {
		name          string
		before, after map[string]int
	}{
		{"atacando", standing.ManeuverOffense, shaped.ManeuverOffense},
		{"resistindo", standing.ManeuverDefense, shaped.ManeuverDefense},
	} {
		if got := side.after["derrubar"] - side.before["derrubar"]; got != -2 {
			t.Errorf("%s, a Forma Sorrateira (Pequena) muda o Derrubar em %d na Mesa, e a Tab. 1-21 dá −2", side.name, got)
		}
	}
}
