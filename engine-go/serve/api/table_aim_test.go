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

// VOLTAR A VEZ desfaz um "Próximo": quem volta encontra o turno como o deixou, com o que já gastou, a mira
// e o ataque a mais. Antes ele recebia o orçamento de quem estava na vez, e era uma ação de graça.
func TestRewindingGivesTheTurnBackAsItWas(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	store := f.s.sessions
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar: %q", refusal)
	}
	before := sceneOf(t, f)
	if before.MovementLeft || !before.StandardLeft || before.AimedAt != goblin {
		t.Fatalf("o controle falhou: %+v", *before)
	}
	if _, err := store.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	if scene := sceneOf(t, f); !scene.MovementLeft || scene.AimedAt != "" {
		t.Fatalf("o controle falhou: quem entrou na vez não tem o turno cheio: %+v", *scene)
	}
	if _, err := store.PreviousTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	after := sceneOf(t, f)
	if after.MovementLeft != before.MovementLeft || after.StandardLeft != before.StandardLeft || after.AimedAt != before.AimedAt {
		t.Errorf("a vez voltou diferente: %+v, e tinha saído como %+v", *after, *before)
	}
	// E o "Próximo" de novo não deixa o gasto passar adiante.
	if _, err := store.NextTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	if scene := sceneOf(t, f); !scene.MovementLeft || !scene.StandardLeft || scene.AimedAt != "" {
		t.Errorf("depois de voltar, o próximo não recebeu o turno cheio: %+v", *scene)
	}
}

// Voltar para antes de qualquer "Próximo" guardado: o turno é cheio, e o que era de quem estava na vez não
// passa para quem entrou nela.
func TestRewindingPastTheHistoryGivesAFullTurn(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar: %q", refusal)
	}
	if _, err := f.s.sessions.PreviousTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.sessions.PreviousTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.sessions.PreviousTurn(t.Context(), f.sessionID); err != nil {
		t.Fatal(err)
	}
	if scene := sceneOf(t, f); scene.AimedAt != "" {
		t.Errorf("a mira de quem estava na vez passou adiante: %+v", *scene)
	}
}

// MIRA APURADA: "quando usa a ação mirar, você recebe +2 em testes de ataque e na margem de ameaça com ataques
// à distância até o fim do turno" (p128). Quem liga é o gesto, e não há interruptor para ganhar o +2 sem mirar.
func TestAimingAppliesTheSharpAim(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	seedPowerChoices(t, f.s, id, `["mira-apurada"]`, "{}")
	if chips := acoesDaMesa(t, f); strings.Contains(chips, "ao mirar") {
		t.Error("a Mira Apurada ainda tem interruptor")
	}
	target := fmt.Sprintf("/personagens/%d/efeitos/situacao", id)
	if refusal := sceneRefusal(f.requests(t, f.player, http.MethodPost, target, `{"conditional":"flag:mira-apurada"}`).Body.String()); refusal == "" {
		t.Error("o interruptor escondido ainda liga pelo endereço")
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o disparo: %q", refusal)
	}
	plain := pendingOf(t, f)
	gmCancels(t, f)

	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o disparo mirado: %q", refusal)
	}
	aimed := pendingOf(t, f)
	// +5 do combate corpo a corpo que a mira anula, +2 da Mira Apurada.
	if got := (aimed.Total - aimed.Roll) - (plain.Total - plain.Roll); got != 7 {
		t.Errorf("mirar mudou o ataque em %+d, e devia ser +7", got)
	}
	if got := plain.CritRange - aimed.CritRange; got != 2 {
		t.Errorf("a margem de ameaça mudou %d, e o livro dá 2", got)
	}
	gmCancels(t, f)

	// "Até o fim do turno".
	for range 2 {
		if _, err := f.s.sessions.NextTurn(t.Context(), f.sessionID); err != nil {
			t.Fatal(err)
		}
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o disparo do turno seguinte: %q", refusal)
	}
	if next := pendingOf(t, f); next.Total-next.Roll != plain.Total-plain.Roll {
		t.Errorf("o bônus durou além do turno: %+d contra %+d", next.Total-next.Roll, plain.Total-plain.Roll)
	}
}

// Quem não tem o poder mira sem ganhar o bônus.
func TestAimingWithoutTheSharpAimOnlyCancelsThePenalty(t *testing.T) {
	f, id, goblin := archerBesideAFight(t)
	before, _ := shoots(t, f, id, goblin)
	gmCancels(t, f)
	if refusal := rollsFromTheActions(t, f, id, "mirar", goblin); refusal != "" {
		t.Fatalf("mirar: %q", refusal)
	}
	if after, _ := shoots(t, f, id, goblin); after != before+5 {
		t.Errorf("o bônus foi de %+d a %+d, e sem o poder são só os 5", before, after)
	}
}
