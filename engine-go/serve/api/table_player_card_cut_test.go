package api

import (
	"html"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O RECORTE DO JOGADOR LEVA O QUE EXPLICA A CONTA DELE. O `forPlayer` copiava a carta campo
// a campo, e deixava para trás duas coisas que são do próprio jogador: a manobra que ele
// declarou, e as situações da Tabela 5-3 do lado dele (as do alvo ficam com o mestre, como a
// Defesa; decisão do dono).

func playerPage(t *testing.T, f sceneFixture) string {
	t.Helper()
	return html.UnescapeString(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String())
}

func TestThePlayerCardKeepsTheDeclaredManeuverAndHisSituations(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	seeded := *pendingOf(t, f)
	seeded.Situations = []string{"flanqueando", "cobertura leve"}
	seeded.AttackerSituations = []string{"flanqueando"}
	seeded.FollowUps = []live.FollowUp{{Name: "Lutador de Chão", Maneuver: "derrubar", On: live.RulingHit}}
	replacesTheProposal(t, f, seeded)

	// O CONTROLE: o mestre lê as duas situações e a manobra declarada.
	gm := html.UnescapeString(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	for _, want := range []string{"flanqueando", "cobertura leve", "abre derrubar em seguida"} {
		if !strings.Contains(gm, want) {
			t.Fatalf("a carta do mestre não diz %q — o caso mediria outra coisa", want)
		}
	}

	page := playerPage(t, f)
	if !strings.Contains(page, "abre derrubar em seguida") {
		t.Error("a carta do jogador não diz a manobra que ele mesmo declarou")
	}
	if !strings.Contains(page, "flanqueando") {
		t.Error("a carta do jogador não diz a situação do lado dele, que explica o total")
	}
	if strings.Contains(page, "cobertura leve") {
		t.Error("a carta do jogador diz a cobertura do alvo, que é do mestre")
	}
}
