package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// A INVESTIDA (p235) E A INVESTIDA IMPRUDENTE (p42), ALE-499.
//
//	"Você avança até o dobro de seu deslocamento [...] e, no fim do movimento,
//	faz um ataque corpo a corpo. Você recebe +2 no teste de ataque, mas sofre –2
//	na Defesa até o seu próximo turno, porque sua guarda fica aberta."
//
//	"Quando faz uma investida, você pode aumentar sua penalidade na Defesa pela
//	investida para –5 para receber um bônus de +1d12 na rolagem de dano deste
//	ataque."
//
// O app APLICA a penalidade e a expira quando a vez de quem investiu volta
// (decisão do dono): o ataque de um inimigo contra ele compara com a Defesa da
// ficha, e uma penalidade só anunciada faria a mesa mostrar a conta errada. A
// linha reta, o dobro do deslocamento e o terreno são do mestre.

func charges(t *testing.T, f sceneFixture, id int64, kind, target string) string {
	t.Helper()
	return rollsFromTheActions(t, f, id, kind+"/0", target)
}

func TestTheChargeAddsTwoToTheAttackAndOpensTheGuardUntilTheChargersNextTurn(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	before := defenceOnTheSheet(t, f, barbaro)

	if refusal := charges(t, f, barbaro, "investida", goblin); refusal != "" {
		t.Fatalf("a investida foi recusada: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil {
		t.Fatal("a investida não deixou provisório")
	}
	if !strings.Contains(strings.Join(pending.Situations, ","), "investida") {
		t.Errorf("o provisório não diz que foi investida: situações %v", pending.Situations)
	}
	// ANTES DE CONFIRMAR, a guarda ainda está fechada: o golpe cancelado não
	// aconteceu.
	if got := defenceOnTheSheet(t, f, barbaro); got != before {
		t.Errorf("a Defesa caiu na PROPOSTA (%d → %d); a investida só acontece quando o mestre confirma", before, got)
	}

	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if got := defenceOnTheSheet(t, f, barbaro); got != before-2 {
		t.Errorf("Defesa depois da investida: %d, want %d (–2, p235)", got, before-2)
	}
	// A INVESTIDA É AÇÃO COMPLETA: não sobra a padrão nem a de movimento.
	if scene := stateOf(t, f.s.sessions, f.sessionID).Scene; scene.StandardLeft || scene.MovementLeft {
		t.Errorf("a investida deixou ação no turno (padrão %v, movimento %v); ela é completa (p235)",
			scene.StandardLeft, scene.MovementLeft)
	}

	// A VEZ DO GOBLIN: a guarda continua aberta — é nela que a penalidade vale.
	if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("girar a vez: %v", err)
	}
	if got := defenceOnTheSheet(t, f, barbaro); got != before-2 {
		t.Errorf("na vez do goblin a Defesa voltou a %d; o –2 vale até o PRÓXIMO TURNO do bárbaro", got)
	}
	// A VEZ DO BÁRBARO VOLTA, e a penalidade cai.
	if _, err := f.s.sessions.NextTurn(context.Background(), f.sessionID); err != nil {
		t.Fatalf("girar a vez: %v", err)
	}
	if got := defenceOnTheSheet(t, f, barbaro); got != before {
		t.Errorf("com a vez de volta a Defesa é %d, want %d", got, before)
	}
}

func TestTheRecklessChargeTradesFiveDefenceForOneD12(t *testing.T) {
	f, barbaro, goblin := barbarianOnTurn(t)
	before := defenceOnTheSheet(t, f, barbaro)

	// QUEM NÃO TEM O PODER não faz a Imprudente.
	if refusal := charges(t, f, barbaro, "investida-imprudente", goblin); !strings.Contains(refusal, "Investida Imprudente") {
		t.Fatalf("sem o poder, a Imprudente tinha de ser recusada pelo nome, e veio %q", refusal)
	}
	if refusal := powerCommand(t, f, barbaro, "escolhe/class.barbaro.investida-imprudente", ""); refusal != "" {
		t.Fatalf("escolher a Investida Imprudente: %q", refusal)
	}

	if refusal := charges(t, f, barbaro, "investida-imprudente", goblin); refusal != "" {
		t.Fatalf("a Imprudente foi recusada: %q", refusal)
	}
	pending := stateOf(t, f.s.sessions, f.sessionID).PendingAttack
	if pending == nil || !strings.Contains(strings.Join(pending.Situations, ","), "imprudente") {
		t.Fatalf("o provisório não diz que foi a Imprudente: %+v", pending)
	}
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/ataque/confirmar", ""); rec.Code != http.StatusOK {
		t.Fatalf("confirmar deu %d", rec.Code)
	}
	if got := defenceOnTheSheet(t, f, barbaro); got != before-5 {
		t.Errorf("Defesa depois da Imprudente: %d, want %d (–5, p42)", got, before-5)
	}
}
