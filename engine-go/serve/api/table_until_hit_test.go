package api

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"t20engine/domain/live"
)

// O ATÉ ACERTAR (p76): "Se você errar um ataque desarmado, recebe um bônus cumulativo de +2
// em testes de ataque e rolagens de dano desarmado contra o mesmo oponente. Os bônus terminam
// quando você acertar um ataque ou no fim da cena, o que acontecer primeiro."
//
// Quem diz que errou é o MESTRE. O bônus entra só contra o oponente marcado (decisão do dono):
// a carta do golpe seguinte contra ele soma +2 por erro, e o acerto apaga a marca.
//
// O d20 é fixo, e a medida é o TOTAL de dois golpes iguais: a diferença entre eles é o bônus.

func fighterUntilHit(t *testing.T) (sceneFixture, int64, string, int) {
	t.Helper()
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Lutador", 1)
	seedPowerChoices(t, f.s, id, `["class.lutador.ate-acertar"]`, "{}")
	return f, id, goblin, naturalIndex(t, f, "Ataque desarmado")
}

// unarmedTotal propõe o golpe desarmado contra `target` com o d20 fixo e devolve o total.
func unarmedTotal(t *testing.T, f sceneFixture, id int64, unarmed int, target string) int {
	t.Helper()
	f.s.fixedD20 = 10
	defer func() { f.s.fixedD20 = 0 }()
	if refusal := rollsFromTheActions(t, f, id, fmt.Sprintf("atacar/%d", unarmed), target); refusal != "" {
		t.Fatalf("agredir desarmado foi recusado: %q", refusal)
	}
	pa := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pa == nil {
		t.Fatal("o golpe desarmado não deixou provisório")
	}
	return pa.Total
}

// theGameMasterSaysMissedAndTheTurnComesBack: o mestre diz "errou", e a vez gira até
// voltar ao lutador, com a ação padrão de novo.
func theGameMasterSaysMissedAndTheTurnComesBack(t *testing.T, f sceneFixture) {
	t.Helper()
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/julgar/errou", ""); rec.Code != http.StatusOK {
		t.Fatalf("o mestre dizer errou deu %d", rec.Code)
	}
	turnComesBackToTheFighter(t, f)
}

// turnComesBackToTheFighter gira a vez até ela voltar ao Furioso, com a ação padrão de novo.
func turnComesBackToTheFighter(t *testing.T, f sceneFixture) {
	t.Helper()
	for range 6 {
		if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/proxima-vez", ""); rec.Code != http.StatusOK {
			t.Fatalf("girar a vez deu %d", rec.Code)
		}
		st := stateOf(t, f.s.sessions, f.sessionID)
		if st.TurnIndex >= 0 && st.Initiative[st.TurnIndex].Label == "Furioso" {
			return
		}
	}
	t.Fatal("a vez não voltou ao Furioso")
}

func TestEachUnarmedMissAddsTwoAgainstTheSameOpponent(t *testing.T) {
	f, id, goblin, unarmed := fighterUntilHit(t)
	first := unarmedTotal(t, f, id, unarmed, goblin)
	theGameMasterSaysMissedAndTheTurnComesBack(t, f)

	if got := unarmedTotal(t, f, id, unarmed, goblin); got != first+2 {
		t.Fatalf("depois de um erro desarmado, o golpe contra o mesmo Goblin deu %d, e o Até Acertar soma +2 a %d (p76)", got, first)
	}
	theGameMasterSaysMissedAndTheTurnComesBack(t, f)
	if got := unarmedTotal(t, f, id, unarmed, goblin); got != first+4 {
		t.Errorf("depois de dois erros, o golpe deu %d, e o bônus cumulativo leva a %d (p76)", got, first+4)
	}
}

func TestTheUntilHitBonusIsOnlyAgainstThatOpponent(t *testing.T) {
	f, id, goblin, unarmed := fighterUntilHit(t)
	monster := "goblin-salteador"
	if _, err := f.s.sessions.AddInitiativeEntry(context.Background(), f.sessionID, live.InitiativeEntry{
		Label: "Outro Goblin", Initiative: 1, Type: "npc", MonsterID: &monster,
	}); err != nil {
		t.Fatalf("pôr o outro Goblin na fila: %v", err)
	}
	bicho := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Outro Goblin")
	first := unarmedTotal(t, f, id, unarmed, bicho)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar o golpe deu %d", rec.Code)
	}
	unarmedTotal(t, f, id, unarmed, goblin)
	theGameMasterSaysMissedAndTheTurnComesBack(t, f)
	if got := unarmedTotal(t, f, id, unarmed, bicho); got != first {
		t.Errorf("o erro contra o Goblin mudou o golpe contra o outro Goblin de %d para %d; o bônus é contra o mesmo oponente (p76)", first, got)
	}
}

func TestHittingEndsTheUntilHitBonus(t *testing.T) {
	f, id, goblin, unarmed := fighterUntilHit(t)
	first := unarmedTotal(t, f, id, unarmed, goblin)
	theGameMasterSaysMissedAndTheTurnComesBack(t, f)

	attacker := entryLabeled(t, stateOf(t, f.s.sessions, f.sessionID), "Furioso")
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/cancelar", ""); rec.Code != http.StatusOK {
		t.Fatalf("cancelar deu %d", rec.Code)
	}
	theGameMasterConfirmsAPlainHit(t, f, attacker, goblin, 1)
	turnComesBackToTheFighter(t, f)
	if got := unarmedTotal(t, f, id, unarmed, goblin); got != first {
		t.Errorf("depois de acertar, o golpe contra o Goblin deu %d; o bônus termina no acerto e o total volta a %d (p76)", got, first)
	}
}

func TestNoUntilHitBonusWithoutThePower(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Lutador", 1)
	unarmed := naturalIndex(t, f, "Ataque desarmado")
	first := unarmedTotal(t, f, id, unarmed, goblin)
	theGameMasterSaysMissedAndTheTurnComesBack(t, f)
	if got := unarmedTotal(t, f, id, unarmed, goblin); got != first {
		t.Errorf("o lutador sem o Até Acertar ganhou bônus depois de errar: %d virou %d", first, got)
	}
}
