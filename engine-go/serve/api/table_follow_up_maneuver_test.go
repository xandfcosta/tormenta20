package api

import (
	"fmt"
	"html"
	"net/http"
	"strings"
	"testing"
)

// A MANOBRA EM SEGUIDA (ALE-651): o Lutador de Chão (p77) e a Mão na Boca (p74). O jogador DECLARA a ação
// completa antes de rolar (decisão do dono): o interruptor vai no gesto que a abre, a carta a mostra ao
// mestre, e a decisão dele abre a manobra livre na mesma vez, contra o mesmo alvo. Sem a declaração, nada abre.
//
// INTEGRAÇÃO porque o que quebra é a COMPOSIÇÃO: a declaração sai das Ações, a decisão abre a vez, a trava da
// mesa confere a abertura sem cobrar ação, e o PM sai da ficha.

const (
	groundFighter = "class.lutador.lutador-de-chao"
	handOverMouth = "class.ladino.mao-na-boca"
)

// declaresFollowUp manda o gesto com a manobra em seguida declarada.
func declaresFollowUp(t *testing.T, f sceneFixture, id int64, path, target string) string {
	t.Helper()
	body := fmt.Sprintf(`{"turn_target":%q,"follow_up":true}`, target)
	return sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/%s?embutida=1", id, path), body))
}

// followsUp rola a manobra em seguida que o poder abriu.
func followsUp(t *testing.T, f sceneFixture, id int64, maneuver, power, target string) string {
	t.Helper()
	return rollsFromTheActions(t, f, id, "manobra/"+maneuver+"/em-seguida/"+power, target)
}

// O LUTADOR DE CHÃO: declarado no agarrar, o Sucesso abre a derrubada livre por 1 PM, uma por agarrar.
// Sem a declaração, a decisão não abre nada.
func TestTheGroundFighterTripsForOneManaAfterTheDeclaredGrapple(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Lutador", 1)
	seedPowerChoices(t, f.s, id, `["`+groundFighter+`"]`, "{}")
	if refusal := rollsFromTheActions(t, f, id, "manobra/agarrar", goblin); refusal != "" {
		t.Fatalf("agarrar sem declarar foi recusado: %q", refusal)
	}
	if refusal := gmRefusal(t, f, "/ataque/julgar/sucesso"); refusal != "" {
		t.Fatalf("o Sucesso do agarrar foi recusado: %q", refusal)
	}
	if refusal := followsUp(t, f, id, "derrubar", groundFighter, goblin); !strings.Contains(refusal, "declar") {
		t.Errorf("a derrubada sem o agarrar declarado não foi recusada pela declaração: %q", refusal)
	}

	g, other, enemy := barbarianOnTurn(t)
	seedClasse(t, g.s, other, "Lutador", 1)
	seedPowerChoices(t, g.s, other, `["`+groundFighter+`"]`, "{}")
	if refusal := declaresFollowUp(t, g, other, "manobra/agarrar", enemy); refusal != "" {
		t.Fatalf("o agarrar declarado foi recusado: %q", refusal)
	}
	if pa := pendingOf(t, g); pa == nil || pa.FollowUp == nil || pa.FollowUp.PowerID != groundFighter {
		t.Fatalf("o agarrar não levou a declaração do Lutador de Chão ao mestre: %+v", pa)
	}
	// A CARTA DO MESTRE diz a declaração NA FACE, perto da decisão: é o que ele lê antes de responder, e não um detalhe.
	page := html.UnescapeString(g.requests(t, g.gm, http.MethodGet, g.tableUrl(), "").Body.String())
	face := page[strings.Index(page, `id="table-attack"`):]
	face = face[:strings.Index(face, `id="gm-attack-details"`)]
	if !strings.Contains(face, "Declarou derrubar em seguida (Lutador de Chão, 1 PM)") {
		t.Error("a face da carta do agarrar não diz ao mestre que a derrubada em seguida foi declarada")
	}
	if refusal := followsUp(t, g, other, "derrubar", groundFighter, enemy); refusal == "" {
		t.Error("a derrubada saiu antes da decisão do mestre sobre o agarrar")
	}
	if refusal := gmRefusal(t, g, "/ataque/julgar/sucesso"); refusal != "" {
		t.Fatalf("o Sucesso do agarrar declarado foi recusado: %q", refusal)
	}

	before := poolsOf(t, g.s, other).MpCurrent
	if refusal := followsUp(t, g, other, "derrubar", groundFighter, enemy); refusal != "" {
		t.Fatalf("a derrubada do Lutador de Chão depois do Sucesso foi recusada: %q", refusal)
	}
	pa := pendingOf(t, g)
	if pa == nil || pa.Maneuver == nil || pa.Maneuver.Kind != "derrubar" || pa.FollowUpOf == nil || pa.FollowUpOf.PowerID != groundFighter {
		t.Fatalf("a manobra em seguida não virou a derrubada do Lutador de Chão: %+v", pa)
	}
	if after := poolsOf(t, g.s, other).MpCurrent; after != before-1 {
		t.Errorf("a derrubada do Lutador de Chão deixou %d PM, want %d − 1", after, before)
	}
	// O ✕ DO MESTRE DESFAZ a derrubada: o PM volta, e ela continua aberta.
	if refusal := gmRefusal(t, g, "/ataque/cancelar"); refusal != "" {
		t.Fatalf("o ✕ da derrubada foi recusado: %q", refusal)
	}
	if after := poolsOf(t, g.s, other).MpCurrent; after != before {
		t.Errorf("o ✕ da derrubada deixou %d PM, want os %d de antes", after, before)
	}
	if refusal := followsUp(t, g, other, "derrubar", groundFighter, enemy); refusal != "" {
		t.Fatalf("a derrubada desfeita pelo ✕ não voltou a ficar aberta: %q", refusal)
	}
	if refusal := gmRefusal(t, g, "/ataque/julgar/sucesso"); refusal != "" {
		t.Fatalf("o Sucesso da derrubada foi recusado: %q", refusal)
	}
	if refusal := followsUp(t, g, other, "derrubar", groundFighter, enemy); !strings.Contains(refusal, "declar") {
		t.Errorf("a segunda derrubada do mesmo agarrar não foi recusada: %q", refusal)
	}
}

// A MÃO NA BOCA: declarada num golpe com o Ataque Furtivo ligado, o acerto que o mestre confirma abre o
// agarrar livre, sem PM. Sem o Ataque Furtivo, a declaração é recusada.
func TestTheHandOverMouthGrapplesAfterTheDeclaredSneakHit(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Ladino", 1)
	seedPowerChoices(t, f.s, id, `["`+handOverMouth+`"]`, "{}")
	if refusal := declaresFollowUp(t, f, id, "atacar/0", goblin); !strings.Contains(refusal, "ataque furtivo") {
		t.Errorf("a Mão na Boca sem o Ataque Furtivo ligado não foi recusada por ele: %q", refusal)
	}

	turnSwitchOn(t, f.s, id, "ataque-furtivo")
	if refusal := declaresFollowUp(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe furtivo com a Mão na Boca declarada foi recusado: %q", refusal)
	}
	gmChecks(t, f)

	before := poolsOf(t, f.s, id).MpCurrent
	if refusal := followsUp(t, f, id, "agarrar", handOverMouth, goblin); refusal != "" {
		t.Fatalf("o agarrar da Mão na Boca depois do acerto foi recusado: %q", refusal)
	}
	pa := pendingOf(t, f)
	if pa == nil || pa.Maneuver == nil || pa.Maneuver.Kind != "agarrar" || pa.FollowUpOf == nil || pa.FollowUpOf.PowerID != handOverMouth {
		t.Fatalf("a manobra em seguida não virou o agarrar da Mão na Boca: %+v", pa)
	}
	if after := poolsOf(t, f.s, id).MpCurrent; after != before {
		t.Errorf("o agarrar da Mão na Boca cobrou PM: %d, want %d", after, before)
	}
}

// AS AÇÕES OFERECEM A DECLARAÇÃO E A MANOBRA EM SEGUIDA. O Agarrar do lutador abre a folha com o interruptor
// do Lutador de Chão; o Atacar do ladino só abre a da Mão na Boca com o Ataque Furtivo ligado, que é quando
// ela vale. A linha da manobra livre espera a abertura.
func TestTheActionsOfferTheFollowUpManeuvers(t *testing.T) {
	f, lutador, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, lutador, "Lutador", 1)
	seedPowerChoices(t, f.s, lutador, `["`+groundFighter+`"]`, "{}")
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	for _, want := range []string{"derrubar em seguida", "follow_up", "Derrubar · Lutador de Chão", "/acoes/manobra/derrubar/em-seguida/" + groundFighter, "1 PM"} {
		if !strings.Contains(surface, want) {
			t.Errorf("as Ações do Lutador de Chão não dizem %q", want)
		}
	}

	g, ladino, _ := barbarianOnTurn(t)
	seedClasse(t, g.s, ladino, "Ladino", 1)
	seedPowerChoices(t, g.s, ladino, `["`+handOverMouth+`"]`, "{}")
	surface = superficieDeAcoes(t, acoesDaMesa(t, g))
	if !strings.Contains(surface, "Agarrar · Mão na Boca") {
		t.Error("as Ações do ladino não oferecem o agarrar da Mão na Boca")
	}
	if strings.Contains(surface, "agarrar em seguida") {
		t.Error("a folha da Mão na Boca apareceu sem o Ataque Furtivo ligado, e ela só vale com ele")
	}
	turnSwitchOn(t, g.s, ladino, "ataque-furtivo")
	page := acoesDaMesa(t, g)
	if !strings.Contains(html.UnescapeString(page), `aria-label="Rolar Atacar · Machado de batalha"`) {
		t.Error("a folha da Mão na Boca no golpe não se chama pela linha (Atacar · Machado de batalha)")
	}
	if surface = superficieDeAcoes(t, page); !strings.Contains(surface, "agarrar em seguida") {
		t.Error("o Atacar do ladino com o Ataque Furtivo ligado não abre a folha da Mão na Boca")
	}
	// UMA FOLHA POR ARMA, e o desarmado já é a segunda: dois diálogos com o mesmo id abririam o mesmo.
	for _, weapon := range []string{"0", "1"} {
		if id := `id="follow-up-` + weapon + "-" + handOverMouth + `"`; strings.Count(page, id) != 1 {
			t.Errorf("a folha da Mão na Boca da arma %s aparece %d vezes, e é uma", weapon, strings.Count(page, id))
		}
	}
}

// O interruptor da declaração e a abertura da manobra em seguida são sinais TIPADOS, e a mesa os declara.
func TestTheFollowUpSignalsAreDeclaredByTheTable(t *testing.T) {
	f, lutador, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, lutador, "Lutador", 1)
	seedPowerChoices(t, f.s, lutador, `["`+groundFighter+`"]`, "{}")
	expectTypedSignalsDeclared(t, acoesDaMesa(t, f), "$follow_up", "$follow_up_open")
}
