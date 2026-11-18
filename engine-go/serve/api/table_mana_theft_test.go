package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
)

// O ROUBO DE MANA (p74): "Quando você causa dano com um ataque furtivo, para cada 1d6 de
// dano de seu ataque furtivo, você recebe 1 PM temporário e a criatura perde 1 ponto de
// mana (se tiver). Você só pode usar este poder uma vez por cena contra uma mesma
// criatura."
//
// Ele sai na CONFIRMAÇÃO do mestre, como o Panache: é ela que diz que o golpe causou dano.
// Os dados de furtivo são os que a PROPOSTA guardou, com o interruptor ligado. Roubos
// contra criaturas diferentes na mesma cena SOMAM (decisão do dono).

// thiefAtTheTable é a arcanista da bancada com sete níveis de ladino — 4d6 de furtivo —,
// o Truque Mágico e, se pedido, o Roubo de Mana, numa cena de ação com o Goblin.
func thiefAtTheTable(t *testing.T, withThePower bool) (sceneFixture, string) {
	t.Helper()
	f := newSceneFixture(t)
	seedClasse(t, f.s, f.charID, "Ladino", 7)
	powers := `["class.ladino.truque-magico"]`
	if withThePower {
		powers = `["class.ladino.truque-magico","class.ladino.roubo-de-mana"]`
	}
	choiceCom(t, f, f.charID, powers, `[]`)
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	return f, f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
}

// confirmsASneakAttack semeia o golpe com `dice` d6 de furtivo contra `target` e o confirma.
func confirmsASneakAttack(t *testing.T, f sceneFixture, target string, dice int) {
	t.Helper()
	attacker := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Arcanista")
	if _, err := f.s.sessions.ProposeAttack(context.Background(), f.sessionID, live.PendingAttack{
		AttackerEntryID: attacker, TargetEntryID: target, TargetLabel: "alvo",
		Weapon: "Adaga", Roll: 15, Total: 22, Defense: 13,
		Hit: true, RawDamage: 12, Damage: 12, Ruling: live.RulingHit, DamageRolled: true, SneakDice: dice,
	}); err != nil {
		t.Fatalf("semear o golpe furtivo: %v", err)
	}
	rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmar o golpe deu %d", rec.Code)
	}
	if refusal := tableRefusal(t, rec.Body.String()); refusal != "" {
		t.Fatalf("confirmar o golpe foi recusado: %s", refusal)
	}
}

func colleagueAtTheTable(t *testing.T, f sceneFixture) (int64, string) {
	t.Helper()
	colleague := withAColleague(t, f)
	if _, err := f.s.sessions.AddInitiativeEntry(context.Background(), f.sessionID, sheetCombatant("Guerreira", 3, colleague)); err != nil {
		t.Fatalf("pôr a Guerreira na fila: %v", err)
	}
	arrangePools(t, f.s, colleague, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = 6
		return pools, nil
	})
	// A VEZ SAI DA LADINA, pela razão da bancada do bárbaro: o provisório semeado de quem
	// está na vez cobra a ação padrão, e o segundo golpe seria recusado por falta dela.
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	return colleague, entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Guerreira")
}

func TestTheProposalKeepsTheSneakAttackDice(t *testing.T) {
	f, goblin := thiefAtTheTable(t, true)
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", f.charID)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{"conditional":"flag:ataque-furtivo"}`).Body.String()); refusal != "" {
		t.Fatalf("ligar o Ataque Furtivo: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, f.charID, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe: %q", refusal)
	}
	pa := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pa == nil {
		t.Fatal("o golpe não deixou provisório")
	}
	if pa.SneakDice != 4 {
		t.Errorf("o golpe com o furtivo ligado guardou %d d6 de furtivo, e o ladino 7 rola 4d6 (p73)", pa.SneakDice)
	}
}

func TestTheManaTheftGivesTemporaryPMAndTakesTheTargetsPM(t *testing.T) {
	f, _ := thiefAtTheTable(t, true)
	colleague, guerreira := colleagueAtTheTable(t, f)

	confirmsASneakAttack(t, f, guerreira, 4)
	if got := tempMpOf(t, f, f.charID); got != 4 {
		t.Errorf("o Roubo de Mana de 4d6 deu %d PM temporários ao ladino, e o livro dá 4 (p74)", got)
	}
	if got := mpOf(t, f.s, colleague); got != 2 {
		t.Errorf("a Guerreira tinha 6 PM e ficou com %d; o roubo de 4d6 tira 4 (p74)", got)
	}

	// UMA VEZ POR CENA CONTRA UMA MESMA CRIATURA.
	confirmsASneakAttack(t, f, guerreira, 4)
	if got := tempMpOf(t, f, f.charID); got != 4 {
		t.Errorf("o segundo roubo contra a mesma criatura na cena levou a reserva a %d; o livro limita a uma vez por cena", got)
	}
	if got := mpOf(t, f.s, colleague); got != 2 {
		t.Errorf("o segundo roubo contra a mesma criatura tirou PM de novo: %d", got)
	}
}

func TestManaTheftsAgainstDifferentCreaturesAddUp(t *testing.T) {
	f, goblin := thiefAtTheTable(t, true)
	_, guerreira := colleagueAtTheTable(t, f)
	confirmsASneakAttack(t, f, guerreira, 4)
	confirmsASneakAttack(t, f, goblin, 3)
	if got := tempMpOf(t, f, f.charID); got != 7 {
		t.Errorf("roubos de 4 e de 3 contra criaturas diferentes deram %d PM temporários, e somam 7 (decisão do dono)", got)
	}
}

func TestNoManaIsStolenWithoutThePower(t *testing.T) {
	f, _ := thiefAtTheTable(t, false)
	colleague, guerreira := colleagueAtTheTable(t, f)
	confirmsASneakAttack(t, f, guerreira, 4)
	if got := tempMpOf(t, f, f.charID); got != 0 {
		t.Errorf("o ladino sem o Roubo de Mana ganhou %d PM temporários", got)
	}
	if got := mpOf(t, f.s, colleague); got != 6 {
		t.Errorf("a Guerreira perdeu PM para um ladino sem o Roubo de Mana: %d", got)
	}
}
