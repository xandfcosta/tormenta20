package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// ROLAR DE NOVO O ATAQUE NA FAIXA DO PROVISÓRIO (p226, ALE-508): só quem rolou,
// antes de o mestre confirmar, e vale o SEGUNDO dado.
//
// O provisório é semeado com um 1 natural, como no `boostedProposal`: o d20 da
// proposta é de verdade, e o que se prende aqui é o que o SEGUNDO faz.

func pendingWithANaturalOne(t *testing.T, f sceneFixture, path string) live.PendingAttack {
	t.Helper()
	if rec := f.requests(t, f.player, http.MethodPost, f.tableUrl()+path, ""); rec.Code != http.StatusOK {
		t.Fatalf("propor %s deu %d", path, rec.Code)
	}
	state := stateOf(t, f.s.sessions, f.sessionID)
	if state.PendingAttack == nil {
		t.Fatalf("%s não virou provisório", path)
	}
	// O 1 NATURAL ERRA (p221): o provisório semeado é a sugestão inteira, e
	// não só o dado.
	seeded := *state.PendingAttack
	seeded.Total += 1 - seeded.Roll
	seeded.Roll = 1
	seeded.Hit, seeded.Critical = false, false
	if seeded.Maneuver != nil {
		seeded.Maneuver.Margin = seeded.Total - seeded.Maneuver.Opposed
	}
	if _, err := f.s.sessions.ProposeAttack(t.Context(), f.sessionID, seeded); err != nil {
		t.Fatalf("semear o 1 natural no provisório: %v", err)
	}
	return seeded
}

func rerollsTheAttack(t *testing.T, f sceneFixture, who int64, key, body string) live.PendingAttack {
	t.Helper()
	f.requests(t, who, http.MethodPost, f.tableUrl()+"/ataque/rolar-de-novo/"+key, body)
	state := stateOf(t, f.s.sessions, f.sessionID)
	if state.PendingAttack == nil {
		t.Fatal("o provisório sumiu depois de rolar de novo")
	}
	return *state.PendingAttack
}

func withPowers(t *testing.T, f sceneFixture, powers, choices string) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET classPowers = ?, powerChoices = ? WHERE id = ?", powers, choices, f.charID,
	); err != nil {
		t.Fatalf("semear os poderes: %v", err)
	}
}

// p130 Sortudo: "Quando faz um teste, você pode gastar 3 PM para rolá-lo
// novamente." O 1 natural erra; o 20 natural do segundo dado acerta e critica.
func TestLuckyRerollsTheAttackAndTheSecondDieStands(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")

	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, `id="table-attack" class="ds-strike"`) {
		t.Fatal("a página da mesa não desenhou o cartão do provisório")
	}
	if !strings.Contains(page, "Rolar de novo · Sortudo · 3 PM") {
		t.Fatal("a faixa de quem rolou não oferece o Sortudo")
	}
	if gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(gm, "Rolar de novo · Sortudo") {
		t.Error("o mestre, que não rolou, vê o botão de rolar de novo")
	}

	before := mpOf(t, f.s, f.charID)
	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":20}`)
	if got.Roll != 20 || !got.Hit || !got.Critical {
		t.Errorf("depois de rolar de novo com 20: d20 %d, acertou %v, crítico %v; o 20 natural acerta e critica (p221, p231)",
			got.Roll, got.Hit, got.Critical)
	}
	if len(got.Dice) != 0 {
		t.Errorf("rolar de novo o teste rolou dano (%v): o dano vem depois da decisão do mestre", got.Dice)
	}
	if len(got.Rerolls) != 1 || got.Rerolls[0].Discarded != 1 || got.Rerolls[0].Name != "Sortudo" {
		t.Errorf("o provisório não guarda o dado que saiu: %+v", got.Rerolls)
	}
	if spent := before - mpOf(t, f.s, f.charID); spent != 3 {
		t.Errorf("o Sortudo cobrou %d PM, e a p130 diz 3", spent)
	}
	if gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(gm, "o d20 1 saiu") {
		t.Error("a faixa não diz ao mestre qual dado saiu")
	}
}

// SÓ QUEM ROLOU rola de novo: o mestre não gasta o PM de ninguém.
func TestOnlyWhoRolledRerollsTheAttack(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")
	if got := rerollsTheAttack(t, f, f.gm, "sortudo", `{"reroll_d20":20}`); got.Roll != 1 {
		t.Errorf("o mestre rolou de novo o ataque do jogador: d20 %d", got.Roll)
	}
}

// ROLAR DE NOVO VALE ANTES DA DECISÃO (p226): *"devem ser usados antes de o
// mestre declarar se você passou ou não no teste"*. Depois do "errou" do
// mestre, o teste já foi julgado.
func TestNoRerollAfterTheGMRules(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/errou", ""); rec.Code != http.StatusOK {
		t.Fatalf("o mestre dizer errou deu %d", rec.Code)
	}
	before := mpOf(t, f.s, f.charID)
	if got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":20}`); got.Roll != 1 || len(got.Rerolls) != 0 {
		t.Errorf("o jogador rolou de novo depois de o mestre decidir: d20 %d, %+v", got.Roll, got.Rerolls)
	}
	if spent := before - mpOf(t, f.s, f.charID); spent != 0 {
		t.Errorf("o rolar de novo recusado cobrou %d PM", spent)
	}
}

// p66 Mestre em Arma: "Escolha uma arma. Com esta arma, […] quando faz um
// teste de ataque você pode gastar 2 PM para rolá-lo novamente."
func TestWeaponMasteryRerollsOnlyWithTheChosenWeapon(t *testing.T) {
	const poder = `["class.guerreiro.mestre-em-arma"]`
	f, goblin := attackOnTurn(t)
	withPowers(t, f, poder, `{"class.guerreiro.mestre-em-arma":["adaga"]}`)
	pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if strings.Contains(page, "Rolar de novo · Mestre em Arma") {
		t.Error("o Mestre em Arma da adaga oferece rolar de novo o golpe da espada longa")
	}

	withPowers(t, f, poder, `{"class.guerreiro.mestre-em-arma":["espada-longa"]}`)
	page = f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "Rolar de novo · Mestre em Arma · 2 PM") {
		t.Fatal("o Mestre em Arma da espada longa não oferece rolar de novo o golpe dela")
	}
	if got := rerollsTheAttack(t, f, f.player, "mestre_em_arma", `{"reroll_d20":20}`); got.Roll != 20 {
		t.Errorf("o Mestre em Arma não rolou de novo: d20 %d", got.Roll)
	}
}

// A MANOBRA rola de novo o d20 de quem TENTOU; o do defensor fica, e a
// diferença é refeita contra ele (p234).
func TestTheManeuverRerollKeepsTheDefendersDie(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	seeded := pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/manobra/derrubar")
	if seeded.Maneuver == nil {
		t.Fatal("a manobra não veio no provisório")
	}
	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":15}`)
	if got.Maneuver == nil || got.Maneuver.OpposedRoll != seeded.Maneuver.OpposedRoll {
		t.Fatalf("o d20 do defensor mudou: %+v, antes %+v", got.Maneuver, seeded.Maneuver)
	}
	if got.Roll != 15 || got.Total != seeded.Total+14 || got.Maneuver.Margin != seeded.Maneuver.Margin+14 {
		t.Errorf("a manobra com 15 em vez de 1: d20 %d, total %d, diferença %d; esperava 15, %d e %d",
			got.Roll, got.Total, got.Maneuver.Margin, seeded.Total+14, seeded.Maneuver.Margin+14)
	}
}

// p48 Sorte de Nimb: "Qualquer resultado 11 ou mais na segunda rolagem será
// considerado um 20 natural" — e o 20 natural acerta e critica (p221, p231).
func TestNimbsLuckMakesTheAttackANaturalTwenty(t *testing.T) {
	f, goblin := attackOnTurn(t)
	seedClasse(t, f.s, f.charID, "Bucaneiro", 20)
	pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")
	got := rerollsTheAttack(t, f, f.player, "sorte_de_nimb", `{"reroll_d20":11}`)
	if got.Roll != 11 || !got.Hit || !got.Critical || len(got.Rerolls) != 1 || !got.Rerolls[0].Natural {
		t.Errorf("Sorte de Nimb com 11: d20 %d, acertou %v, crítico %v, %+v; esperava o 11 valendo 20 natural",
			got.Roll, got.Hit, got.Critical, got.Rerolls)
	}
	if page := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(page, "11 conta como 20 natural") {
		t.Error("a faixa não diz que o 11 conta como 20 natural")
	}
}
