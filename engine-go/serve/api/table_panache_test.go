package api

import (
	"net/http"
	"testing"

	"t20engine/domain/sheet"
)

// O PANACHE (p48): "A partir do 5º nível, sempre que faz um acerto crítico em combate
// ou reduz um inimigo a 0 PV, você recupera 1 PM."
//
// O gatilho é o mesmo do Sangue dos Inimigos (`criticalOrDrop`, p42), e quem o dispara
// é a CONFIRMAÇÃO do mestre: é ele quem diz que foi crítico, e o dano confirmado é o
// que zera o alvo. O efeito é outro: devolver PM ao pool, até o máximo.

// buccaneerOnTurn é a arcanista da bancada com cinco níveis de bucaneiro — o Panache
// vem com o 5º, sem escolha —, numa cena de ação, com 3 PM a menos que o máximo. A vez
// passa ao Goblin pela razão do bárbaro: o provisório semeado não cobra a ação dela.
func buccaneerOnTurn(t *testing.T, withThePower bool) (sceneFixture, string) {
	t.Helper()
	f := newSceneFixture(t)
	if withThePower {
		seedClasse(t, f.s, f.charID, "Bucaneiro", 5)
	}
	if rec := f.requests(t, f.gm, http.MethodPatch, f.tableUrl(), `{"status":"active"}`); rec.Code != http.StatusOK {
		t.Fatalf("iniciar a sessão deu %d", rec.Code)
	}
	f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 2)
	attacker := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Arcanista")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
		t.Fatalf("girar a vez deu %d", rec.Code)
	}
	arrangePools(t, f.s, f.charID, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = pools.MpMax - 3
		return pools, nil
	})
	return f, attacker
}

func TestTheCriticalHitGivesTheBuccaneerOnePM(t *testing.T) {
	f, attacker := buccaneerOnTurn(t, true)
	goblin := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Goblin")
	before := mpOf(t, f.s, f.charID)
	confirmsACriticalProposedWith(t, f, attacker, goblin)
	if got := mpOf(t, f.s, f.charID); got != before+1 {
		t.Errorf("o crítico confirmado levou o PM do bucaneiro de %d a %d, e o Panache devolve 1 (p48)", before, got)
	}
}

func TestDroppingAnEnemyToZeroGivesTheBuccaneerOnePM(t *testing.T) {
	f, attacker := buccaneerOnTurn(t, true)
	bicho := aBeastWithHitPoints(t, f, 4)
	before := mpOf(t, f.s, f.charID)

	// O CONTROLE: o acerto que não derruba não devolve nada — é o que separa
	// "derrubou" de "acertou".
	theGameMasterConfirmsAPlainHit(t, f, attacker, bicho, 1)
	if got := mpOf(t, f.s, f.charID); got != before {
		t.Fatalf("um acerto que não derrubou mexeu no PM do bucaneiro: %d virou %d", before, got)
	}
	theGameMasterConfirmsAPlainHit(t, f, attacker, bicho, 99)
	if got := mpOf(t, f.s, f.charID); got != before+1 {
		t.Errorf("derrubar o inimigo levou o PM de %d a %d, e o Panache devolve 1 (p48)", before, got)
	}
}

func TestThePanacheDoesNotGoPastTheMaximumPM(t *testing.T) {
	f, attacker := buccaneerOnTurn(t, true)
	arrangePools(t, f.s, f.charID, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = pools.MpMax
		return pools, nil
	})
	full := mpOf(t, f.s, f.charID)
	goblin := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Goblin")
	confirmsACriticalProposedWith(t, f, attacker, goblin)
	if got := mpOf(t, f.s, f.charID); got != full {
		t.Errorf("com o PM cheio (%d), o Panache levou a %d: recuperar não passa do máximo", full, got)
	}
}

func TestNoPMComesBackWithoutThePanache(t *testing.T) {
	f, attacker := buccaneerOnTurn(t, false)
	goblin := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Goblin")
	before := mpOf(t, f.s, f.charID)
	confirmsACriticalProposedWith(t, f, attacker, goblin)
	if got := mpOf(t, f.s, f.charID); got != before {
		t.Errorf("a arcanista sem o Panache recuperou PM no crítico: %d virou %d", before, got)
	}
}
