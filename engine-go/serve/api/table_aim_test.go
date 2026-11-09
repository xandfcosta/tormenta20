package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/board"
)

// MIRAR: "Você mira em um alvo que possa ver, dentro do alcance de sua arma. Isso anula a penalidade
// de –5 em testes de Pontaria realizados neste turno contra aquele alvo caso ele esteja engajado em
// combate corpo a corpo" (p234).

const inMeleeLabel = "alvo em combate corpo a corpo"

// archerBesideAFight é o arqueiro na vez, com o Goblin ao lado de um companheiro no tabuleiro: a
// única forma de o −5 existir é o mapa dizer que ele está em combate.
func archerBesideAFight(t *testing.T) (sceneFixture, int64, string) {
	t.Helper()
	f, id, goblin := styleOnTurn(t, "ataque-poderoso", false, "arco-curto")
	f.seedOpenBoard(t, "stone")
	for _, token := range []board.BoardToken{
		{Label: "Goblin", Kind: "npc", X: 10, Y: 0, Footprint: 1, EntryID: &goblin},
		{Label: "Companheiro", Kind: "character", X: 11, Y: 0, Footprint: 1},
	} {
		if _, err := f.s.tableHost().Boards().AddToken(context.Background(), f.sessionID, defaultTab, token); err != nil {
			t.Fatalf("pôr %q: %v", token.Label, err)
		}
	}
	return f, id, goblin
}

func shoots(t *testing.T, f sceneFixture, id int64, target string) (bonus int, situations string) {
	t.Helper()
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", target); refusal != "" {
		t.Fatalf("o disparo foi recusado: %q", refusal)
	}
	pa := pendingOf(t, f)
	return pa.Total - pa.Roll, strings.Join(pa.Situations, ",")
}

func TestAimingCancelsTheShootingIntoMeleePenalty(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	before, situations := shoots(t, f, id, goblin)
	// O CONTROLE: sem mirar, o mapa cobra o −5.
	if !strings.Contains(situations, inMeleeLabel) {
		t.Fatalf("o controle falhou: o disparo não viu o combate corpo a corpo: %q", situations)
	}
	gmCancels(t, f)

	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar foi recusado: %q", refusal)
	}
	if scene := sceneOf(t, f); scene.MovementLeft || !scene.StandardLeft {
		t.Errorf("mirar é ação de movimento (p234): %+v", *scene)
	}
	after, situations := shoots(t, f, id, goblin)
	if strings.Contains(situations, inMeleeLabel) {
		t.Errorf("mirado, o disparo ainda cobrou o −5: %q", situations)
	}
	if after != before+5 {
		t.Errorf("o bônus foi de %+d a %+d, e mirar devolve os 5", before, after)
	}
}

func TestAimingEndsWithTheTurn(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar foi recusado: %q", refusal)
	}
	for range 2 {
		if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
			t.Fatalf("girar a vez: %v", err)
		}
	}
	if _, situations := shoots(t, f, id, goblin); !strings.Contains(situations, inMeleeLabel) {
		t.Errorf("a mira sobreviveu à virada do turno: %q", situations)
	}
}

func TestAimingNeedsTheMovementAndATarget(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	if refusal := rollsFromTheActions(t, f, id, "mirar", ""); !strings.Contains(refusal, "alvo") {
		t.Errorf("mirar sem alvo devia ser recusado, e veio %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("a primeira mira: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("a segunda mira troca a padrão pelo movimento (p233), e foi recusada: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); !strings.Contains(refusal, "não sobrou ação") {
		t.Errorf("sem ação, mirar devia ser recusado, e veio %q", refusal)
	}
}

// Quem só tem arma de Luta na mão não tem em que mirar: a p234 fala de testes de Pontaria.
func TestTheAimLineIsForWhoShoots(t *testing.T) {
	f, _, _ := archerBesideAFight(t)
	if surface := superficieDeAcoes(t, acoesDaMesa(t, f)); !strings.Contains(surface, "Mirar") {
		t.Error("o arqueiro não tem a linha de Mirar")
	}
	f, _, _ = styleOnTurn(t, "ataque-poderoso", false, "espada-longa")
	if surface := superficieDeAcoes(t, acoesDaMesa(t, f)); strings.Contains(surface, "Mirar") {
		t.Error("a espada tem a linha de Mirar")
	}
}

// Depois de mirar, a barra do alvo diz em quem: é o único rastro do gesto antes do disparo.
func TestTheTargetBarSaysWhoIsAimedAt(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	const says = "Mirando em"
	if strings.Contains(acoesDaMesa(t, f), says) {
		t.Fatal("a barra diz que mira antes do gesto")
	}
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar foi recusado: %q", refusal)
	}
	if !strings.Contains(acoesDaMesa(t, f), says) {
		t.Error("a barra não diz em quem o personagem mira")
	}
}

// "Não dá para agir" vale para mirar como vale para o golpe (p233).
func TestAStunnedCharacterDoesNotAim(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	f.requests(t, f.player, http.MethodPost, fmt.Sprintf("/personagens/%d/efeitos/condicao/atordoado", id), "")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal == "" {
		t.Fatal("o controle falhou: atordoado, o golpe devia ser recusado")
	}
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal == "" {
		t.Error("atordoado, mirar foi aceito")
	}
	if !sceneOf(t, f).MovementLeft {
		t.Error("a recusa gastou o movimento")
	}
}

// Quem arremessa também faz teste de Pontaria, e a p234 fala desses testes.
func TestTheAimLineIsAlsoForWhoThrows(t *testing.T) {
	f, _, _ := styleOnTurn(t, "ataque-poderoso", false, "adaga")
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	if !strings.Contains(surface, "/outro-uso") {
		t.Fatal("o controle falhou: a adaga não tem o arremesso")
	}
	if !strings.Contains(surface, "Mirar") {
		t.Error("a adaga arremessada paga o −5 e não tem a linha de Mirar")
	}
}
