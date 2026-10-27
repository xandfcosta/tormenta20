package api

import (
	"net/http"
	"strings"
	"testing"
)

// ROLAR DE NOVO O TESTE NA FAIXA (p226, ALE-507): só quem rolou, enquanto o
// teste é o último da mesa, e vale o SEGUNDO dado.

func rerollsOnTheTable(t *testing.T, f sceneFixture, who int64, key, body string) *http.Response {
	t.Helper()
	return f.requests(t, who, http.MethodPost, f.tableUrl()+"/teste/rolar-de-novo/"+key, body).Result()
}

func TestTheEtiquetteRerollsTheDiplomacyTestAndTheSecondDieStands(t *testing.T) {
	f, id := spenderAtTheTable(t, "Cavaleiro", 5, 0)
	choiceCom(t, f, id, `["class.cavaleiro.etiqueta"]`, "[]")
	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":4}`); refusal != "" {
		t.Fatalf("rolar Diplomacia: %q", refusal)
	}
	first := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest

	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Rolar de novo · Etiqueta · 1 PM") {
		t.Fatal("a faixa de quem rolou não oferece a Etiqueta")
	}
	if gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(gm, "Rolar de novo · Etiqueta") {
		t.Error("o mestre, que não rolou, vê o botão de rolar de novo")
	}

	before := pm(t, f, id)
	rerollsOnTheTable(t, f, f.player, "etiqueta", `{"reroll_d20":15}`)
	second := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if second.Roll != 15 || second.Total != first.Total+11 {
		t.Errorf("depois de rolar de novo: d20 %d, total %d; esperava d20 15 e total %d (vale o segundo, p226)",
			second.Roll, second.Total, first.Total+11)
	}
	if len(second.Rerolls) != 1 || second.Rerolls[0].Discarded != 4 || second.Rerolls[0].Name != "Etiqueta" {
		t.Errorf("a faixa não guarda o dado que saiu: %+v", second.Rerolls)
	}
	if spent := before - pm(t, f, id); spent != 1 {
		t.Errorf("a Etiqueta cobrou %d PM, e a p54 diz 1", spent)
	}
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, "o d20 4 saiu") {
		t.Error("a faixa não diz qual dado saiu")
	}
}

// SÓ QUEM ROLOU rola de novo: o mestre não gasta o PM de ninguém.
func TestOnlyWhoRolledRerolls(t *testing.T) {
	f, id := spenderAtTheTable(t, "Cavaleiro", 5, 0)
	choiceCom(t, f, id, `["class.cavaleiro.etiqueta"]`, "[]")
	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":4}`); refusal != "" {
		t.Fatalf("rolar Diplomacia: %q", refusal)
	}
	rerollsOnTheTable(t, f, f.gm, "etiqueta", `{"reroll_d20":15}`)
	if got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest; got.Roll != 4 {
		t.Errorf("o mestre rolou de novo o teste do jogador: d20 %d", got.Roll)
	}
}

// p48 Sorte de Nimb: "Qualquer resultado 11 ou mais na segunda rolagem será
// considerado um 20 natural."
func TestNimbsLuckMakesElevenOrMoreANaturalTwenty(t *testing.T) {
	f, id := spenderAtTheTable(t, "Bucaneiro", 20, 0)
	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":2}`); refusal != "" {
		t.Fatalf("rolar Diplomacia: %q", refusal)
	}
	rerollsOnTheTable(t, f, f.player, "sorte_de_nimb", `{"reroll_d20":13}`)
	got := stateOf(t, f.s.sessions, f.sessionID).LastSkillTest
	if got.Roll != 13 || !got.Natural20 || len(got.Rerolls) != 1 || !got.Rerolls[0].Natural {
		t.Errorf("Sorte de Nimb com 13: d20 %d, 20 natural %v, %+v; esperava o 13 valendo 20 natural", got.Roll, got.Natural20, got.Rerolls)
	}
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, "13 conta como 20 natural") {
		t.Error("a faixa não diz que o 13 conta como 20 natural")
	}
}
