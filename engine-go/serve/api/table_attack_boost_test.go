package api

import (
	"net/http"
	"strings"
	"testing"
)

// A FAIXA DIZ O IMPULSO E O PREÇO. O PM sai na proposta, e quem responde precisa saber
// quanto o cancelamento devolve. O provisório é semeado com o impulso direto no estado da
// sessão: o personagem do caso não tem o poder, e o pagamento é medido em
// `table_attack_payment_test.go`.
func TestTheStripSaysTheBoostAndItsCost(t *testing.T) {
	f, goblin := attackOnTurn(t)
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/atacar", ""); rec.Code != http.StatusOK {
		t.Fatalf("propor o ataque deu %d", rec.Code)
	}
	boosted := *stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	boosted.BoostPowerID, boosted.BoostPM = "class.guerreiro.ataque-especial-4", 2
	replacesTheProposal(t, f, boosted)
	page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	// O CONTROLE: a faixa está na página, senão a ausência abaixo não diz nada.
	if !strings.Contains(page, `id="table-attack"`) {
		t.Fatal("a página da mesa não desenhou a faixa do provisório")
	}
	if !strings.Contains(page, "Ataque Especial · 2 PM") {
		t.Error("a faixa não diz o impulso do golpe nem o PM dele")
	}
}
