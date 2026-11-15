package api

import (
	"fmt"
	"strings"
	"testing"
)

// O ATAQUE PAGO QUE NÃO GASTA A VEZ (ALE-651): o Ataque Extra (p66) e o Golpe Relâmpago (p76). Ele é o
// mesmo extra da arma natural — abre depois da resposta do mestre ao golpe de agredir, não gasta ação, e cada
// fonte conta os próprios usos —, com o PM do poder.
//
// INTEGRAÇÃO porque o que quebra é a COMPOSIÇÃO: o gesto sai das Ações, a carta diz o PM, a trava da mesa
// confere a abertura e a conta, e o PM sai da ficha.

func paidExtra(t *testing.T, f sceneFixture, id int64, weapon int, power, target, signals string) string {
	t.Helper()
	body := fmt.Sprintf(`{"turn_target":%q%s}`, target, signals)
	return sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/ataque-pago/%d/%s?embutida=1", id, weapon, power), body))
}

const (
	extraAttack    = "class.guerreiro.ataque-extra"
	lightningStrik = "class.lutador.golpe-relampago"
)

// O ATAQUE EXTRA: 2 PM, uma vez por rodada, só depois de agredir e da resposta do mestre.
func TestTheExtraAttackPaysTwoManaAfterTheStrike(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Guerreiro", 6)

	if refusal := paidExtra(t, f, id, 0, extraAttack, goblin, ""); !strings.Contains(refusal, "depois de agredir") {
		t.Errorf("o Ataque Extra antes de agredir não foi recusado pela ordem: %q", refusal)
	}
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("agredir foi recusado: %q", refusal)
	}
	gmChecks(t, f)

	before := poolsOf(t, f.s, id).MpCurrent
	if refusal := paidExtra(t, f, id, 0, extraAttack, goblin, ""); refusal != "" {
		t.Fatalf("o Ataque Extra depois de agredir foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, f); pa == nil || pa.ExtraOf != extraAttack {
		t.Fatalf("o Ataque Extra não virou o extra do poder: %+v", pa)
	}
	if after := poolsOf(t, f.s, id).MpCurrent; after != before-2 {
		t.Errorf("o Ataque Extra deixou %d PM, want %d − 2", after, before)
	}
	gmChecks(t, f)

	if refusal := paidExtra(t, f, id, 0, extraAttack, goblin, ""); !strings.Contains(refusal, "uma vez por rodada") {
		t.Errorf("o segundo Ataque Extra da vez não foi recusado pela regra: %q", refusal)
	}
}

// O GOLPE RELÂMPAGO é DESARMADO: depois de agredir com o machado ele é recusado, e depois de agredir desarmado
// ele sai, por 1 PM.
func TestTheLightningStrikeFollowsAnUnarmedStrike(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Lutador", 1)
	unarmed := naturalIndex(t, f, "Ataque desarmado")

	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("agredir com o machado foi recusado: %q", refusal)
	}
	gmChecks(t, f)
	if refusal := paidExtra(t, f, id, unarmed, lightningStrik, goblin, ""); !strings.Contains(refusal, "desarmado") {
		t.Errorf("o Golpe Relâmpago depois do machado não foi recusado por pedir o agredir desarmado: %q", refusal)
	}

	g, other, enemy := barbarianOnTurn(t)
	seedClasse(t, g.s, other, "Lutador", 1)
	if refusal := rollsFromTheActions(t, g, other, fmt.Sprintf("atacar/%d", unarmed), enemy); refusal != "" {
		t.Fatalf("agredir desarmado foi recusado: %q", refusal)
	}
	gmChecks(t, g)
	before := poolsOf(t, g.s, other).MpCurrent
	if refusal := paidExtra(t, g, other, unarmed, lightningStrik, enemy, ""); refusal != "" {
		t.Fatalf("o Golpe Relâmpago depois do agredir desarmado foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, g); pa == nil || pa.ExtraOf != lightningStrik {
		t.Fatalf("o Golpe Relâmpago não virou o extra do poder: %+v", pa)
	}
	if after := poolsOf(t, g.s, other).MpCurrent; after != before-1 {
		t.Errorf("o Golpe Relâmpago deixou %d PM, want %d − 1", after, before)
	}
}

// AS AÇÕES OFERECEM O ATAQUE PAGO, e a linha espera a abertura como a do extra da arma natural.
func TestTheActionsOfferThePaidExtraStrikes(t *testing.T) {
	f, id, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Guerreiro", 6)
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	for _, want := range []string{"Ataque Extra · Machado de batalha", "/acoes/ataque-pago/0/" + extraAttack, "2 PM"} {
		if !strings.Contains(surface, want) {
			t.Errorf("as Ações do guerreiro de 6º nível não dizem %q", want)
		}
	}

	g, lutador, _ := barbarianOnTurn(t)
	seedClasse(t, g.s, lutador, "Lutador", 1)
	surface = superficieDeAcoes(t, acoesDaMesa(t, g))
	for _, want := range []string{"Golpe Relâmpago · Ataque desarmado", "depois de agredir desarmado"} {
		if !strings.Contains(surface, want) {
			t.Errorf("as Ações do lutador não dizem %q", want)
		}
	}
	if strings.Contains(surface, "Golpe Relâmpago · Machado de batalha") {
		t.Error("o Golpe Relâmpago apareceu no machado, e ele é desarmado (p76)")
	}
}

// A espera do agredir desarmado é sinal TIPADO, e a mesa o declara: sem a declaração ele nasce texto vazio, e
// o golpe chegaria sem o alvo (ver o caso das Ações).
func TestTheLightningStrikeSignalsAreDeclaredByTheTable(t *testing.T) {
	f, lutador, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, lutador, "Lutador", 1)
	expectTypedSignalsDeclared(t, acoesDaMesa(t, f), "$extra_unarmed")
}
