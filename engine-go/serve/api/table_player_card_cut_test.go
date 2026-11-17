package api

import (
	"html"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O RECORTE DO JOGADOR LEVA O QUE EXPLICA A CONTA DELE. O `forPlayer` copiava a carta campo
// a campo, e deixava para trás três coisas que são do próprio jogador: a manobra que ele
// declarou, as situações da Tabela 5-3 do lado dele (as do alvo ficam com o mestre, como a
// Defesa; decisão do dono) e o "não letal" do dano da arma dele.

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

func TestThePlayerCardSaysTheDamageIsNonLethal(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	seeded := *pendingOf(t, f)
	seeded.Ruling, seeded.Hit = live.RulingHit, true
	seeded.DamageRolled, seeded.Dice, seeded.Faces = true, []int{6}, 8
	seeded.RawDamage, seeded.Damage, seeded.NonLethal = 9, 9, 9
	replacesTheProposal(t, f, seeded)

	if gm := html.UnescapeString(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()); !strings.Contains(gm, "não letal") {
		t.Fatal("a carta do mestre não diz que o dano é não letal — o caso mediria outra coisa")
	}
	// O "não letal" leva um espaço INQUEBRÁVEL (`nonLethalWords`): a 390px a linha não o parte.
	if page := playerPage(t, f); !strings.Contains(page, "não\u00a0letal") {
		t.Error("a carta do jogador não diz que o dano da arma dele é não letal")
	}
}
