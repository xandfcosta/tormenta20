package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// A PROPOSTA PAGA AO NASCER, e o cancelamento do mestre devolve (decisão do dono). Cobrada
// no ✓, a mesma ação e o mesmo PM podiam ser gastos em outra coisa enquanto o mestre
// decidia, e a resposta dele travava.

func sceneOf(t *testing.T, f sceneFixture) *live.Scene {
	t.Helper()
	return stateOf(t, f.s.sessions, f.sessionID).Scene
}

func proposesOnTheQueue(t *testing.T, f sceneFixture, target string) {
	t.Helper()
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/iniciativa/"+target+"/atacar", "")
	if pendingOf(t, f) == nil {
		t.Fatal("o ataque não virou proposta")
	}
}

// replacesTheProposal troca a proposta por outra semeada, pelo que a mesa tem: o mestre
// cancela, e ela nasce de novo pagando. O `RerollAttack` não serve: ele exige um rolar de
// novo a mais.
func replacesTheProposal(t *testing.T, f sceneFixture, attack live.PendingAttack) {
	t.Helper()
	store := f.s.sessions
	if _, err := store.CancelAttack(t.Context(), f.sessionID, live.Attacker{UserID: f.gm, Role: "gm"}); err != nil {
		t.Fatalf("tirar a proposta da mesa: %v", err)
	}
	if _, err := store.ProposeAttack(t.Context(), f.sessionID, attack); err != nil {
		t.Fatalf("semear a proposta: %v", err)
	}
}

func gmRefusal(t *testing.T, f sceneFixture, path string) string {
	t.Helper()
	return tableRefusal(t, f.requests(t, f.gm, http.MethodPost, f.tableUrl()+path, "").Body.String())
}

func TestTheProposalSpendsTheActionWhenItIsBorn(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	if sceneOf(t, f).StandardLeft {
		t.Error("a proposta nasceu sem gastar a ação padrão")
	}
	if !sceneOf(t, f).MovementLeft {
		t.Error("atacar não toca na ação de movimento")
	}
}

func TestTheGMCancellingGivesTheActionBack(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	gmCancels(t, f)
	if !sceneOf(t, f).StandardLeft {
		t.Error("o golpe cancelado não aconteceu, e a ação padrão não voltou")
	}
}

func TestTheCancelledChargeGivesTheWholeTurnBack(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	if refusal := charges(t, f, barbaro, "investida", goblin); refusal != "" {
		t.Fatalf("a investida foi recusada: %q", refusal)
	}
	if scene := sceneOf(t, f); scene.StandardLeft || scene.MovementLeft {
		t.Fatalf("a investida é completa (p235), e sobrou padrão %v, movimento %v", scene.StandardLeft, scene.MovementLeft)
	}
	gmCancels(t, f)
	if scene := sceneOf(t, f); !scene.StandardLeft || !scene.MovementLeft {
		t.Errorf("a investida cancelada devolve o turno, e veio padrão %v, movimento %v", scene.StandardLeft, scene.MovementLeft)
	}
}

// O cancelamento devolve o que a proposta pagou, e não o turno inteiro.
func TestTheRefundKeepsWhatWasSpentAfterTheProposal(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	if _, err := f.s.sessions.SpendAction(context.Background(), f.sessionID, engine.ActionMovement); err != nil {
		t.Fatalf("gastar o movimento: %v", err)
	}
	gmCancels(t, f)
	scene := sceneOf(t, f)
	if !scene.StandardLeft {
		t.Error("a padrão da proposta não voltou")
	}
	if scene.MovementLeft {
		t.Error("o cancelamento devolveu o movimento que a pessoa gastou depois de propor")
	}
}

// Quem está na vez agora, na mesma rodada, não ganha a ação que outra pessoa pagou.
func TestTheActionStaysSpentWhenTheTurnHasPassed(t *testing.T) {
	f, goblin := attackOnTurn(t)
	store := f.s.sessions
	last := "goblin-salteador"
	if _, err := store.AddInitiativeEntry(context.Background(), f.sessionID, live.InitiativeEntry{
		Label: "Goblin lento", Initiative: 1, Type: "npc", MonsterID: &last,
	}); err != nil {
		t.Fatalf("pôr o último da fila: %v", err)
	}
	proposesOnTheQueue(t, f, goblin)
	round := stateOf(t, store, f.sessionID).Round
	if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("girar a vez: %v", err)
	}
	state := stateOf(t, store, f.sessionID)
	if state.Round != round || state.Initiative[state.TurnIndex].Label != "Goblin lento" {
		t.Fatalf("a bancada pede a vez do Goblin lento na rodada %d, e está na de %s, rodada %d",
			round, state.Initiative[state.TurnIndex].Label, state.Round)
	}
	if _, err := store.SpendAction(context.Background(), f.sessionID, engine.ActionStandard); err != nil {
		t.Fatalf("gastar a padrão de quem está na vez: %v", err)
	}
	gmCancels(t, f)
	if sceneOf(t, f).StandardLeft {
		t.Error("o cancelamento devolveu a padrão a quem não propôs")
	}
}

// A vez volta para quem propôs, mas a ação paga era de outra rodada.
func TestTheActionStaysSpentInTheNextRound(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	store := f.s.sessions
	for range 2 {
		if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	if pendingOf(t, f) == nil {
		t.Fatal("girar a vez não é resposta do mestre, e a proposta sumiu")
	}
	if _, err := store.SpendAction(context.Background(), f.sessionID, engine.ActionStandard); err != nil {
		t.Fatalf("gastar a padrão da rodada nova: %v", err)
	}
	gmCancels(t, f)
	if sceneOf(t, f).StandardLeft {
		t.Error("o cancelamento devolveu, nesta rodada, a padrão paga na rodada anterior")
	}
}

// O erro é um desfecho: fechar a leitura dele não desfaz o golpe.
func TestTheMissKeepsTheActionSpent(t *testing.T) {
	for name, closes := range map[string]func(f sceneFixture){
		"jogador": func(f sceneFixture) {
			f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
		},
		"mestre": func(f sceneFixture) { gmCancels(t, f) },
	} {
		t.Run(name, func(t *testing.T) {
			f, goblin := attackOnTurn(t)
			proposesOnTheQueue(t, f, goblin)
			gmRules(t, f, "errou")
			closes(f)
			if pendingOf(t, f) != nil {
				t.Fatal("o erro não saiu da mesa")
			}
			if sceneOf(t, f).StandardLeft {
				t.Error("fechar o erro devolveu a ação do golpe que aconteceu")
			}
		})
	}
}

// As posturas depois de propor passam pelas rotas de verdade, e o que não couber é
// recusado ali: a resposta do mestre não trava.
func knightProposesAndAssumesStances(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f, knight, _, goblin := knightOnTurn(t)
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	if refusal := rollsFromTheActions(t, f, knight, "atacar/0", goblin); refusal != "" {
		t.Fatalf("propor foi recusado: %q", refusal)
	}
	if refusal := assumes(t, f, knight, "postura-muralha"); refusal != "" {
		t.Fatalf("a primeira postura cabe no movimento, e veio %q", refusal)
	}
	if refusal := assumes(t, f, knight, "postura-torre"); !strings.Contains(refusal, "não sobrou ação") {
		t.Fatalf("a padrão já foi paga, e a segunda postura devia ser recusada; veio %q", refusal)
	}
	return f, knight
}

func TestStancesAfterTheProposalDoNotStopTheMiss(t *testing.T) {
	f, _ := knightProposesAndAssumesStances(t)
	if refusal := gmRefusal(t, f, "/ataque/julgar/errou"); refusal != "" {
		t.Fatalf("o \"errou\" foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Ruling != live.RulingMiss {
		t.Errorf("o erro não ficou na mesa: %+v", pa)
	}
}

func TestStancesAfterTheProposalDoNotStopTheCheck(t *testing.T) {
	f, _ := knightProposesAndAssumesStances(t)
	if refusal := gmRefusal(t, f, "/ataque/julgar/acertou"); refusal != "" {
		t.Fatalf("o \"acertou\" foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("o dano foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓ foi recusado: %q", refusal)
	}
	if pendingOf(t, f) != nil {
		t.Error("o ✓ não tirou o golpe da mesa")
	}
}

// boostingBarbarian é o guerreiro com o Ataque Especial e o Sortudo, com `mana` PM.
func boostingBarbarian(t *testing.T, mana int64) (sceneFixture, int64, string) {
	t.Helper()
	f, barbaro, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, barbaro, "Guerreiro", 5)
	seedPowerChoices(t, f.s, barbaro, `["sortudo"]`, "{}")
	spent := poolsOf(t, f.s, barbaro).MpMax - mana
	if _, err := f.s.db.ExecContext(context.Background(),
		"INSERT OR REPLACE INTO character_damage (characterId, hpDamage, mpSpent) VALUES (?, 0, ?)", barbaro, spent); err != nil {
		t.Fatalf("gastar o PM da bancada: %v", err)
	}
	if got := mpOf(t, f.s, barbaro); got != mana {
		t.Fatalf("a bancada pediu %d PM, e a ficha tem %d", mana, got)
	}
	return f, barbaro, goblin
}

// proposesWithTheBoost propõe com o Ataque Especial de 2 PM, pela rota da barra.
func proposesWithTheBoost(t *testing.T, f sceneFixture, barbaro int64, goblin string) {
	t.Helper()
	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":0,"boost_power":"class.guerreiro.ataque-especial-4","boost_steps":1,"boost_split":"attack"}`, goblin)
	if refusal := sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/atacar/0?embutida=1", barbaro), chosen)); refusal != "" {
		t.Fatalf("propor com o impulso foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.BoostPM != 2 {
		t.Fatalf("a proposta devia levar 2 PM de impulso: %+v", pa)
	}
}

func TestTheBoostIsPaidOnceWhenTheProposalIsBorn(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 5)
	proposesWithTheBoost(t, f, barbaro, goblin)
	if got := mpOf(t, f.s, barbaro); got != 3 {
		t.Fatalf("depois de propor com 2 PM de impulso, a ficha tem %d PM, e são 3", got)
	}
	entry := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Furioso")
	if line := stateOf(t, f.s.sessions, f.sessionID).Initiative[live.FindEntryIndex(stateOf(t, f.s.sessions, f.sessionID), entry)]; line.MpCurrent == nil || *line.MpCurrent != 3 {
		t.Errorf("a linha da fila não espelhou o PM pago: %v", line.MpCurrent)
	}
	gmRules(t, f, "acertou")
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("o dano foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓ foi recusado: %q", refusal)
	}
	if got := mpOf(t, f.s, barbaro); got != 3 {
		t.Errorf("o ✓ cobrou o impulso de novo: a ficha tem %d PM, e são 3", got)
	}
}

func TestTheGMCancellingGivesTheBoostBack(t *testing.T) {
	for name, before := range map[string]func(sceneFixture){
		"antes da decisão":  func(sceneFixture) {},
		"depois do acertou": func(f sceneFixture) { gmRules(t, f, "acertou") },
	} {
		t.Run(name, func(t *testing.T) {
			f, barbaro, goblin := boostingBarbarian(t, 5)
			proposesWithTheBoost(t, f, barbaro, goblin)
			before(f)
			gmCancels(t, f)
			if got := mpOf(t, f.s, barbaro); got != 5 {
				t.Errorf("o golpe cancelado não custa, e a ficha ficou com %d PM de 5", got)
			}
		})
	}
}

func TestTheMissKeepsTheBoostPaid(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 5)
	proposesWithTheBoost(t, f, barbaro, goblin)
	gmRules(t, f, "errou")
	gmCancels(t, f)
	if got := mpOf(t, f.s, barbaro); got != 3 {
		t.Errorf("o golpe que errou aconteceu, e a ficha ficou com %d PM; são 3", got)
	}
}

// O mesmo PM não paga o impulso e o rolar de novo: com 3 PM, o impulso de 2 deixa 1, e o
// Sortudo custa 3.
func TestTheBoostAndTheRerollCannotShareTheMana(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 3)
	proposesWithTheBoost(t, f, barbaro, goblin)
	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":10}`)
	if len(got.Rerolls) != 0 {
		t.Errorf("o Sortudo passou sem PM para ele: %+v", got.Rerolls)
	}
	if mana := mpOf(t, f.s, barbaro); mana != 1 {
		t.Errorf("a ficha ficou com %d PM, e sobra 1 do impulso", mana)
	}
}

func TestTheGMCancellingGivesTheRerollBack(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 6)
	proposesWithTheBoost(t, f, barbaro, goblin)
	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":10}`)
	if len(got.Rerolls) != 1 {
		t.Fatalf("o Sortudo não rolou de novo: %+v", got.Rerolls)
	}
	if mana := mpOf(t, f.s, barbaro); mana != 1 {
		t.Fatalf("depois do impulso e do Sortudo sobra 1 PM, e a ficha tem %d", mana)
	}
	gmCancels(t, f)
	if mana := mpOf(t, f.s, barbaro); mana != 6 {
		t.Errorf("o golpe cancelado não custa, e a ficha ficou com %d PM de 6", mana)
	}
}

// A conferência e a cobrança do rolar de novo moram sob a trava. Conferidas antes, dois
// toques com PM para um só listariam dois, e o cancelamento devolveria o que nunca saiu.
func TestARerollTheSheetCanNoLongerPayIsRefused(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 3)
	proposesOnTheQueue(t, f, goblin)
	stale := *pendingOf(t, f)
	stale.Roll = 19
	stale.Rerolls = []live.DieReroll{{Name: "Sortudo", PM: 4}}
	if _, err := f.s.sessions.RerollAttack(context.Background(), f.sessionID, stale); err == nil {
		t.Error("o rolar de novo de 4 PM passou numa ficha com 3")
	}
	if pa := pendingOf(t, f); len(pa.Rerolls) != 0 {
		t.Errorf("o rolar de novo recusado ficou na proposta: %+v", pa.Rerolls)
	}
	if mana := mpOf(t, f.s, barbaro); mana != 3 {
		t.Errorf("a ficha ficou com %d PM de 3", mana)
	}
}

func TestARerollFromAnOldReadIsRefused(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 9)
	proposesOnTheQueue(t, f, goblin)
	first := *pendingOf(t, f)
	first.Rerolls = []live.DieReroll{{Name: "Sortudo", PM: 3}}
	if _, err := f.s.sessions.RerollAttack(context.Background(), f.sessionID, first); err != nil {
		t.Fatalf("o primeiro rolar de novo: %v", err)
	}
	// A segunda aba leu a proposta antes do primeiro, e chega com a mesma lista.
	if _, err := f.s.sessions.RerollAttack(context.Background(), f.sessionID, first); err == nil {
		t.Error("o segundo pedido, feito sobre a leitura antiga, passou")
	}
	if mana := mpOf(t, f.s, barbaro); mana != 6 {
		t.Errorf("foram 3 PM de um rolar de novo só, e a ficha ficou com %d de 9", mana)
	}
}

// Reiniciar o combate apaga a proposta sem resposta, e ela não aconteceu.
func TestRestartingTheCombatGivesTheOpenProposalBack(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 5)
	proposesWithTheBoost(t, f, barbaro, goblin)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/combate/reiniciar", ""); rec.Code != http.StatusOK {
		t.Fatalf("reiniciar o combate deu %d", rec.Code)
	}
	if pendingOf(t, f) != nil {
		t.Fatal("reiniciar o combate não tirou a proposta da mesa")
	}
	if got := mpOf(t, f.s, barbaro); got != 5 {
		t.Errorf("o golpe apagado não aconteceu, e a ficha ficou com %d PM de 5", got)
	}
}

func TestRestartingTheCombatKeepsTheMissPaid(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 5)
	proposesWithTheBoost(t, f, barbaro, goblin)
	gmRules(t, f, "errou")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/combate/reiniciar", ""); rec.Code != http.StatusOK {
		t.Fatalf("reiniciar o combate deu %d", rec.Code)
	}
	if got := mpOf(t, f.s, barbaro); got != 3 {
		t.Errorf("o golpe que errou aconteceu, e a ficha ficou com %d PM; são 3", got)
	}
}

// Sem a ficha não há para onde devolver, e isso não pode travar a resposta do mestre.
func TestCancellingAfterTheSheetIsGoneStillAnswers(t *testing.T) {
	f, barbaro, goblin := boostingBarbarian(t, 5)
	proposesWithTheBoost(t, f, barbaro, goblin)
	if _, err := f.s.db.ExecContext(context.Background(), "DELETE FROM characters WHERE id = ?", barbaro); err != nil {
		t.Fatalf("apagar a ficha: %v", err)
	}
	if refusal := gmRefusal(t, f, "/ataque/cancelar"); refusal != "" {
		t.Errorf("o cancelamento foi recusado: %q", refusal)
	}
	if pendingOf(t, f) != nil {
		t.Error("a proposta da ficha apagada ficou na mesa")
	}
}

// Uma cena nova pode repetir a rodada e a vez, e o turno já é outro.
func TestTheActionStaysSpentInTheNextScene(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	store := f.s.sessions
	round := stateOf(t, store, f.sessionID).Round
	if _, err := store.StartScene(context.Background(), f.sessionID, live.SceneAction); err != nil {
		t.Fatalf("abrir a cena nova: %v", err)
	}
	for range 2 {
		if _, err := store.NextTurn(context.Background(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	if state := stateOf(t, store, f.sessionID); state.Round != round || state.PendingAttack == nil {
		t.Fatalf("a bancada pede a rodada %d com a proposta aberta, e veio a rodada %d, proposta %v",
			round, state.Round, state.PendingAttack != nil)
	}
	if _, err := store.SpendAction(context.Background(), f.sessionID, engine.ActionStandard); err != nil {
		t.Fatalf("gastar a padrão da cena nova: %v", err)
	}
	gmCancels(t, f)
	if sceneOf(t, f).StandardLeft {
		t.Error("o cancelamento devolveu, na cena nova, a padrão paga na anterior")
	}
}

// bloodEffects conta as linhas do bônus da Sangue dos Inimigos. O bônus só existe nelas.
func bloodEffects(t *testing.T, f sceneFixture, barbaro int64) int {
	t.Helper()
	var lines int
	if err := f.s.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM active_effects WHERE characterId = ? AND catalogId = 'class.barbaro.sangue-dos-inimigos'", barbaro).Scan(&lines); err != nil {
		t.Fatalf("contar os efeitos: %v", err)
	}
	return lines
}

// criticalOnTheTable propõe pela rota de verdade e leva o crítico até antes do ✓.
func criticalOnTheTable(t *testing.T, f sceneFixture, barbaro int64, goblin string) {
	t.Helper()
	if refusal := rollsFromTheActions(t, f, barbaro, "atacar/0", goblin); refusal != "" {
		t.Fatalf("propor foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/julgar/critico"); refusal != "" {
		t.Fatalf("o crítico foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("o dano foi recusado: %q", refusal)
	}
}

func barbarianWithTheBloodPower(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, barbaro, goblin := barbarianOnTurn(t)
	if refusal := powerCommand(t, f, barbaro, "escolhe/class.barbaro.sangue-dos-inimigos", ""); refusal != "" {
		t.Fatalf("escolher a Sangue dos Inimigos: %s", refusal)
	}
	return f, barbaro, goblin
}

func TestTheCheckRaisesTheBonusOfTheStanceAtTheProposal(t *testing.T) {
	f, barbaro, goblin := barbarianWithTheBloodPower(t)
	if refusal := powerCommand(t, f, barbaro, "postura/furia/entra", ""); refusal != "" {
		t.Fatalf("entrar em fúria: %s", refusal)
	}
	criticalOnTheTable(t, f, barbaro, goblin)
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓ foi recusado: %q", refusal)
	}
	if bloodEffects(t, f, barbaro) != 1 {
		t.Error("o crítico proposto em fúria não subiu a Sangue dos Inimigos")
	}
}

func TestARageStartedAfterTheProposalDoesNotRaiseTheBonus(t *testing.T) {
	f, barbaro, goblin := barbarianWithTheBloodPower(t)
	criticalOnTheTable(t, f, barbaro, goblin)
	if refusal := powerCommand(t, f, barbaro, "postura/furia/entra", ""); refusal != "" {
		t.Fatalf("entrar em fúria depois de propor: %s", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓ foi recusado: %q", refusal)
	}
	if lines := bloodEffects(t, f, barbaro); lines != 0 {
		t.Errorf("a fúria ligada depois de propor subiu o bônus: %d linhas", lines)
	}
}

// A PROPOSTA É UM RETRATO do personagem (decisão do dono): quem propôs em fúria acumula,
// mesmo que a fúria tenha acabado antes do ✓.
func TestARageEndedBeforeTheCheckStillRaisesTheBonus(t *testing.T) {
	f, barbaro, goblin := barbarianWithTheBloodPower(t)
	if refusal := powerCommand(t, f, barbaro, "postura/furia/entra", ""); refusal != "" {
		t.Fatalf("entrar em fúria: %s", refusal)
	}
	criticalOnTheTable(t, f, barbaro, goblin)
	endsTheRage(t, f, barbaro)
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); refusal != "" {
		t.Fatalf("o ✓ foi recusado: %q", refusal)
	}
	if lines := bloodEffects(t, f, barbaro); lines != 1 {
		t.Errorf("o crítico foi proposto em fúria, e o bônus não subiu: %d linhas", lines)
	}
}
