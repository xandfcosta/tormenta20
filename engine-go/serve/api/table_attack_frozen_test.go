package api

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"t20engine/domain/live"
)

// A PROPOSTA DE ATAQUE É IMUTÁVEL, e só existe outra depois da resposta do mestre
// (decisão do dono). O que muda na ficha depois de propor não alcança o golpe.

func gmCancels(t *testing.T, f sceneFixture) {
	t.Helper()
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
}

func TestANewAttackWaitsForTheGMsAnswer(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	if refusal := attacks(t, f, goblin, ""); refusal != "" {
		t.Fatalf("o primeiro golpe foi recusado: %q", refusal)
	}
	if refusal := attacks(t, f, goblin, ""); !strings.Contains(refusal, "mestre") {
		t.Errorf("sem resposta do mestre, o golpe novo devia ser recusado, e veio %q", refusal)
	}

	gmRules(t, f, "acertou")
	if refusal := attacks(t, f, goblin, ""); !strings.Contains(refusal, "mestre") {
		t.Errorf("com o acerto esperando o dano e o ✓, o golpe novo devia ser recusado, e veio %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.Ruling != live.RulingHit {
		t.Fatalf("o golpe recusado mexeu no acerto: %+v", pa)
	}

	gmCancels(t, f)
	if refusal := attacks(t, f, goblin, ""); refusal != "" {
		t.Errorf("o cancelamento do mestre é resposta, e o golpe novo foi recusado: %q", refusal)
	}
}

// O caminho da fila da Mesa passa pela mesma regra, e o mestre também responde a ela.
func TestTheQueueGestureAlsoWaitsForTheAnswer(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	attack := f.tableUrl() + "/iniciativa/" + goblin + "/atacar"
	f.requests(t, f.player, http.MethodPost, attack, "")
	if pendingOf(t, f) == nil {
		t.Fatal("o golpe pela fila não virou proposta")
	}
	for name, who := range map[string]int64{"jogador": f.player, "mestre": f.gm} {
		body := f.requests(t, who, http.MethodPost, attack, "").Body.String()
		if refusal := tableRefusal(t, body); !strings.Contains(refusal, "mestre") {
			t.Errorf("%s: o golpe novo pela fila devia ser recusado, e veio %q", name, refusal)
		}
	}
}

// Desistir sem resposta seria o d20 novo de graça: propõe, lê o dado, cancela e rola.
func TestThePlayerCannotTakeBackTheProposal(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	attacks(t, f, goblin, "")

	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	if pendingOf(t, f) == nil {
		t.Fatal("o jogador retirou a própria proposta antes da resposta do mestre")
	}
	gmRules(t, f, "acertou")
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	if pendingOf(t, f) == nil {
		t.Fatal("o jogador retirou o acerto que o mestre decidiu")
	}
	gmCancels(t, f)
	if pendingOf(t, f) != nil {
		t.Error("o mestre não conseguiu cancelar")
	}
}

func TestThePlayerClosesTheMiss(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	attacks(t, f, goblin, "")
	gmRules(t, f, "errou")

	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/cancelar", "")
	if pendingOf(t, f) != nil {
		t.Error("o erro encerrou o golpe, e o jogador não conseguiu fechar a leitura")
	}
}

// damageAfter propõe, roda a mudança, e devolve o que a decisão manda rolar.
func damageAfter(t *testing.T, change func(sceneFixture)) *live.DamageToRoll {
	t.Helper()
	f, goblin := doseAttackOnTurn(t)
	if refusal := attacks(t, f, goblin, ""); refusal != "" {
		t.Fatalf("o golpe foi recusado: %q", refusal)
	}
	change(f)
	gmRules(t, f, "acertou")
	pa := pendingOf(t, f)
	if pa == nil || pa.DamageToRoll == nil {
		t.Fatalf("a decisão não guardou o dano: %+v", pa)
	}
	return pa.DamageToRoll
}

func setStrength(t *testing.T, f sceneFixture, strength int) {
	t.Helper()
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET strength = ? WHERE id = ?", strength, f.charID); err != nil {
		t.Fatalf("mudar a Força: %v", err)
	}
}

func TestChangingTheSheetAfterTheProposalDoesNotChangeTheDamage(t *testing.T) {
	proposed := damageAfter(t, func(sceneFixture) {})
	changed := damageAfter(t, func(f sceneFixture) { setStrength(t, f, 15) })
	if changed.Bonus != proposed.Bonus {
		t.Errorf("a Força mudou depois da proposta, e o bônus do dano foi de %d para %d", proposed.Bonus, changed.Bonus)
	}

	// CONTROLE: antes da proposta, a mesma mudança mexe no dano. Sem isto o caso
	// passaria com uma Força que não entra na conta.
	f, goblin := doseAttackOnTurn(t)
	setStrength(t, f, 15)
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	if pa := pendingOf(t, f); pa.DamageToRoll.Bonus == proposed.Bonus {
		t.Fatalf("a Força não muda o bônus (%d): o caso não mede nada", proposed.Bonus)
	}
}

func TestTheRolledDamageUsesTheProposedNumbers(t *testing.T) {
	proposed := damageAfter(t, func(sceneFixture) {})

	f, goblin := doseAttackOnTurn(t)
	attacks(t, f, goblin, "")
	gmRules(t, f, "acertou")
	setStrength(t, f, 15)
	f.requests(t, f.player, http.MethodPost, f.tableUrl()+"/ataque/dano/na-mao", `{"damage_die_1":5}`)
	pa := pendingOf(t, f)
	if pa.RawDamage != 5+proposed.Bonus {
		t.Errorf("o dano saiu %d, e o proposto é 5 + %d", pa.RawDamage, proposed.Bonus)
	}
}

func TestPuttingTheWeaponAwayDoesNotChangeTheProposal(t *testing.T) {
	toRoll := damageAfter(t, func(f sceneFixture) {
		if _, err := f.s.db.ExecContext(context.Background(),
			"UPDATE character_items SET equipped = NULL WHERE characterId = ?", f.charID); err != nil {
			t.Fatalf("guardar a arma: %v", err)
		}
	})
	if len(toRoll.Dice) != 1 || toRoll.Dice[0].Count != 1 || toRoll.Dice[0].Faces != 8 {
		t.Errorf("a espada longa proposta pede 1d8, e a decisão mandou rolar %+v", toRoll.Dice)
	}
}

// A decisão e o dano seguem com o alvo proposto. O ✓ não tem em quem pousar, diz isso, e a
// resposta que sobra é o cancelamento.
func TestTheTargetLeavingTheQueueStopsOnlyTheCheck(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	attacks(t, f, goblin, "")
	if _, err := f.s.sessions.RemoveInitiativeEntry(context.Background(), f.sessionID, goblin); err != nil {
		t.Fatalf("tirar o alvo da fila: %v", err)
	}
	gmRules(t, f, "acertou")
	if pa := pendingOf(t, f); pa == nil || pa.Ruling != live.RulingHit {
		t.Fatalf("sem o alvo na fila, o mestre não conseguiu decidir: %+v", pa)
	}
	if refusal := gmRefusal(t, f, "/ataque/dano"); refusal != "" {
		t.Fatalf("sem o alvo na fila, o dano foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/confirmar"); !strings.Contains(refusal, "saiu da fila") {
		t.Errorf("o ✓ devia dizer que o alvo saiu da fila, e veio %q", refusal)
	}
	gmCancels(t, f)
	if pendingOf(t, f) != nil {
		t.Error("o mestre não conseguiu cancelar o golpe sem alvo")
	}
}

// A proposta de antes desta regra não tem o golpe congelado: a decisão pede o
// cancelamento, em vez de remontar o golpe pela ficha de agora.
func TestAProposalWithoutTheFrozenStrikeAsksForTheCancel(t *testing.T) {
	f, goblin := attackOnTurn(t)
	proposesOnTheQueue(t, f, goblin)
	old := *pendingOf(t, f)
	old.Frozen = nil
	replacesTheProposal(t, f, old)
	if refusal := gmRefusal(t, f, "/ataque/julgar/acertou"); !strings.Contains(refusal, "cancela") {
		t.Fatalf("a decisão devia pedir o cancelamento, e veio %q", refusal)
	}
	gmCancels(t, f)
	if pendingOf(t, f) != nil {
		t.Error("o mestre não conseguiu cancelar a proposta antiga")
	}
}

func TestTheTableTellsTheActionsAnAttackIsOpen(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	if strings.Contains(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), "attack_open: true") {
		t.Fatal("sem proposta, a mesa diz que há ataque em aberto")
	}
	attacks(t, f, goblin, "")
	page := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(page, "attack_open: true") {
		t.Error("com a proposta na mesa, a barra de alvo não avisa")
	}
	if !strings.Contains(page, "Há um ataque em andamento") {
		t.Error("a barra de alvo não diz por que o golpe apagou")
	}
	if !strings.Contains(page, "$attack_open") {
		t.Error("o botão de rolar não espera a resposta do mestre")
	}
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	if card := findNode(doc, "id", "table-attack"); card == nil || strings.Contains(textOf(card), "Cancelar") {
		t.Error("a proposta sem resposta ainda oferece Cancelar ao jogador")
	}
}

// O "rolar de novo" pago é a única troca que a proposta aceita, e só o dado muda.
func TestTheRerollChangesOnlyTheDie(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	seeded := pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")
	setStrength(t, f, 15)

	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":9}`)
	if got.Roll != 9 {
		t.Fatalf("o d20 novo devia ser 9, e veio %d", got.Roll)
	}
	if got.Total-got.Roll != seeded.Total-seeded.Roll {
		t.Errorf("a Força mudou depois da proposta, e o bônus foi de %+d para %+d",
			seeded.Total-seeded.Roll, got.Total-got.Roll)
	}
}

// Na manobra, os bônus saem dos totais guardados (p234), e a ficha de agora não entra.
func TestTheManeuverRerollChangesOnlyTheDie(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	seeded := pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/manobra/derrubar")
	setStrength(t, f, 15)

	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":9}`)
	if got.Roll != 9 {
		t.Fatalf("o d20 novo devia ser 9, e veio %d", got.Roll)
	}
	if got.Total-got.Roll != seeded.Total-seeded.Roll {
		t.Errorf("a Força mudou depois da proposta, e o bônus foi de %+d para %+d",
			seeded.Total-seeded.Roll, got.Total-got.Roll)
	}
	if got.Maneuver.Opposed != seeded.Maneuver.Opposed {
		t.Errorf("o lado de quem se defende mudou: %d virou %d", seeded.Maneuver.Opposed, got.Maneuver.Opposed)
	}
}

// Rolar de novo não pode levar a dose nem o golpe congelado embora.
func TestTheRerollKeepsTheDose(t *testing.T) {
	f, goblin := attackOnTurn(t)
	withPowers(t, f, `["sortudo"]`, "{}")
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	pendingWithANaturalOne(t, f, "/iniciativa/"+goblin+"/atacar")

	got := rerollsTheAttack(t, f, f.player, "sortudo", `{"reroll_d20":9}`)
	if got.Dose == nil || got.Dose.CatalogID != "acido" || len(got.Frozen) == 0 {
		t.Errorf("rolar de novo perdeu parte da proposta: dose %+v", got.Dose)
	}
}

// O erro encerra o golpe: o cartão fica para o jogador ler, e nada mais espera o mestre.
func TestTheMissClosesTheOpenAttack(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	attacks(t, f, goblin, "")
	gmRules(t, f, "errou")
	if strings.Contains(f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(), "attack_open: true") {
		t.Error("depois do erro, a mesa segue dizendo que há ataque em aberto")
	}
}
