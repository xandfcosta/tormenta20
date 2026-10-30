package api

import (
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O TESTE ROLADO NA SESSÃO DO JOGADOR (ALE-473, mockup `paridade.html`): o
// cartão do ataque sem os verbos, com o selo dos naturais (p221) e a conta.
func TestTheParallelSessionShowsTheRolledTest(t *testing.T) {
	f := umaMesaAoVivo(t)
	rolaPericia(t, f, "Iniciativa", 20)
	card := rolledTestCard(t, f)
	text := textOf(card)
	for _, want := range []string{"Teste", "20 natural", "Iniciativa", "d20 20"} {
		if !strings.Contains(text, want) {
			t.Errorf("o cartão do teste rolado não mostra %q: %q", want, text)
		}
	}
	if findNode(card, "data-tone", "good") == nil {
		t.Error("o selo do 20 natural não tem o tom do sucesso")
	}
}

// O ROLAR DE NOVO é de quem rolou (p226): o cartão dele oferece o gasto.
func TestTheRolledTestOffersTheRerollToWhoRolled(t *testing.T) {
	f, id := spenderAtTheTable(t, "Cavaleiro", 5, 0)
	choiceCom(t, f, id, `["class.cavaleiro.etiqueta"]`, "[]")
	if refusal := rollsWith(t, f, id, "Diplomacia", `{"expertise_d20":4}`); refusal != "" {
		t.Fatalf("rolar Diplomacia: %q", refusal)
	}
	if text := textOf(rolledTestCard(t, f)); !strings.Contains(text, "Rolar de novo · Etiqueta · 1 PM") {
		t.Errorf("o cartão de quem rolou não oferece a Etiqueta: %q", text)
	}
}

// rolledTestCard é a região do teste rolado na sessão do jogador.
func rolledTestCard(t *testing.T, f sceneFixture) *html.Node {
	t.Helper()
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl()+"?tempestade=1", "").Body.String()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	card := findNode(doc, "id", "table-skill-test")
	if card == nil || !strings.Contains(attr(card, "class"), "ds-strike") {
		t.Fatal("a sessão do jogador não desenha o cartão do teste rolado")
	}
	return card
}
