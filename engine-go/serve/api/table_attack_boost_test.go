package api

import (
	"net/http"
	"strings"
	"testing"
)

// O PM DO IMPULSO SAI NA CONFIRMAÇÃO (ALE-487). O jogador propõe o golpe com o
// Golpe Divino ou o Ataque Especial; o provisório leva o PM, e só quando o
// mestre CONFIRMA a ficha paga (decisão do dono) — o golpe cancelado não custa.
//
// O provisório é semeado com o impulso direto no estado da sessão: o que se
// prende aqui é a COBRANÇA, e a escolha do impulso na proposta é do `Strike`
// (`strike_boost_test.go`), com o motor provando o número.
func boostedProposal(t *testing.T, pm int) (sceneFixture, int64) {
	t.Helper()
	f, goblin := attackOnTurn(t)
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	if state.PendingAttack == nil {
		t.Fatal("o ataque não virou provisório")
	}
	boosted := *state.PendingAttack
	boosted.BoostPowerID, boosted.BoostPM = "class.guerreiro.ataque-especial-4", pm
	if _, err := f.s.sessions.ProposeAttack(t.Context(), f.sessionID, boosted); err != nil {
		t.Fatalf("pôr o impulso no provisório: %v", err)
	}
	return f, mpOf(t, f.s, f.charID)
}

func TestConfirmingTheAttackChargesTheBoost(t *testing.T) {
	f, before := boostedProposal(t, 2)
	if before < 2 {
		t.Fatalf("o personagem do caso tem %d PM — o caso precisa de pelo menos 2", before)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if after := mpOf(t, f.s, f.charID); after != before-2 {
		t.Errorf("PM depois de confirmar o golpe de 2 PM: %d, want %d", after, before-2)
	}
}

func TestCancellingTheAttackChargesNothing(t *testing.T) {
	f, before := boostedProposal(t, 2)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar deu %d", rec.Code)
	}
	if after := mpOf(t, f.s, f.charID); after != before {
		t.Errorf("PM depois de cancelar: %d, want %d — o golpe cancelado não aconteceu", after, before)
	}
}

// A FAIXA DIZ O IMPULSO E O PREÇO antes de o mestre confirmar: confirmar cobra
// PM da ficha, e quem confirma precisa saber quanto.
func TestTheStripSaysTheBoostAndWhatConfirmingCharges(t *testing.T) {
	f, _ := boostedProposal(t, 2)
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a faixa está na página, senão a ausência abaixo não diz nada.
	if !strings.Contains(page, "mesa-ataque-selo") {
		t.Fatal("a página da mesa não desenhou a faixa do provisório")
	}
	if !strings.Contains(page, "Ataque Especial · 2 PM ao confirmar") {
		t.Error("a faixa não diz o impulso do golpe nem o PM que a confirmação cobra")
	}
}
