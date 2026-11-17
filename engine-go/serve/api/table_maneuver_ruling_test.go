package api

import (
	"html"
	"net/http"
	"strings"
	"testing"
)

// A MANOBRA TEM A DECISÃO DO MESTRE (ALE-659): Sucesso ou Falha, no lugar do ✓ que só dizia "aceito". Os dois
// encerram a manobra num toque (ela fica na carta só para o jogador ler) — ela não rola dano —, e a Falha mantém a ação gasta, como o erro do golpe. É a
// decisão que diz se o que depende da manobra abre: a derrubada do Lutador de Chão só vem do agarrar que venceu.
func TestTheGameMasterRulesTheManeuverSuccessOrFailure(t *testing.T) {
	f, goblin := attackOnTurn(t)
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/manobra/derrubar", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a manobra deu %d", rec.Code)
	}
	if pendingOf(t, f) == nil {
		t.Fatal("a manobra não virou provisório")
	}

	// A CARTA DO MESTRE oferece as duas decisões, e não o ✓.
	page := html.UnescapeString(f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String())
	card := page[strings.Index(page, `id="table-attack"`):]
	card = card[:strings.Index(card, `id="gm-attack-details"`)]
	for _, want := range []string{"julgar/sucesso", "julgar/falha", ">Sucesso<", ">Falha<"} {
		if !strings.Contains(card, want) {
			t.Errorf("a carta da manobra não oferece %q ao mestre", want)
		}
	}
	if strings.Contains(card, "Confirmar e aplicar") {
		t.Error("a carta da manobra ainda tem o ✓, que não diz se ela venceu")
	}

	if refusal := gmRefusal(t, f, "/ataque/confirmar"); !strings.Contains(refusal, "Sucesso ou Falha") {
		t.Errorf("o ✓ numa manobra não foi recusado pela decisão que ela pede: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/julgar/acertou"); !strings.Contains(refusal, "Sucesso ou Falha") {
		t.Errorf("o acertou numa manobra não foi recusado pela decisão que ela pede: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/julgar/falha"); refusal != "" {
		t.Fatalf("a Falha da manobra foi recusada: %q", refusal)
	}
	// ENCERRADA, e na mesa para o jogador ler (ver `table_maneuver_ruling_read_test.go`).
	if pa := pendingOf(t, f); pa == nil || !pa.Settled() {
		t.Errorf("a Falha não encerrou a manobra: %+v", pa)
	}
	if sceneOf(t, f).StandardLeft {
		t.Error("a Falha devolveu a ação padrão, e ela fica gasta como no erro do golpe")
	}

	g, enemy := attackOnTurn(t)
	if rec := g.requests(t, g.player, http.MethodPost, g.tableUrl()+"/iniciativa/"+enemy+"/manobra/derrubar", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor a manobra deu %d", rec.Code)
	}
	if refusal := gmRefusal(t, g, "/ataque/julgar/sucesso"); refusal != "" {
		t.Fatalf("o Sucesso da manobra foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, g); pa == nil || !pa.Settled() {
		t.Errorf("o Sucesso não encerrou a manobra: %+v", pa)
	}
}

// O GOLPE NÃO TEM SUCESSO NEM FALHA: a decisão dele é errou, acertou ou crítico, porque é ela que diz o dano.
func TestTheStrikeIsNotRuledSuccessOrFailure(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	if refusal := gmRefusal(t, f, "/ataque/julgar/sucesso"); !strings.Contains(refusal, "errou, acertou ou crítico") {
		t.Errorf("o Sucesso num golpe não foi recusado pela decisão que ele pede: %q", refusal)
	}
}
