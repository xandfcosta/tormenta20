package api

import (
	"html"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O RECORTE DO JOGADOR LEVA O QUE É DELE. O `forPlayer` copiava a carta campo a campo, e
// deixava para trás a manobra que ele mesmo declarou.

func playerPage(t *testing.T, f sceneFixture) string {
	t.Helper()
	return html.UnescapeString(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())
}

func TestThePlayerCardKeepsTheDeclaredManeuver(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	seeded := *pendingOf(t, f)
	seeded.FollowUps = []live.FollowUp{{Name: "Lutador de Chão", Maneuver: "derrubar", On: live.RulingHit}}
	replacesTheProposal(t, f, seeded)

	// O CONTROLE: o mestre lê a manobra declarada.
	gm := html.UnescapeString(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	for _, want := range []string{"abre derrubar em seguida"} {
		if !strings.Contains(gm, want) {
			t.Fatalf("a carta do mestre não diz %q — o caso mediria outra coisa", want)
		}
	}

	page := playerPage(t, f)
	if !strings.Contains(page, "abre derrubar em seguida") {
		t.Error("a carta do jogador não diz a manobra que ele mesmo declarou")
	}
}
