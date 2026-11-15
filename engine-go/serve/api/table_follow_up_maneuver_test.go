package api

import (
	"html"
	"net/http"
	"strings"
	"testing"
)

// A MANOBRA EM SEGUIDA (ALE-651, ALE-659): o Lutador de Chão (p77) e a Mão na Boca (p74). Ela abre pela
// DECISÃO do mestre sobre o gesto que vem antes — o Sucesso no agarrar, o acerto do golpe com o Ataque Furtivo
// ligado —, na mesma vez e contra o mesmo alvo, e o jogador decide depois se a faz (decisão do dono). Nada é
// rolado antes de valer a condição de que depende.
//
// INTEGRAÇÃO porque o que quebra é a COMPOSIÇÃO: o gesto sai das Ações, a decisão abre a vez, a trava da mesa
// confere a abertura sem cobrar ação, e o PM sai da ficha.

const (
	groundFighter = "class.lutador.lutador-de-chao"
	handOverMouth = "class.ladino.mao-na-boca"
)

// followsUp rola a manobra em seguida que o poder abriu.
func followsUp(t *testing.T, f sceneFixture, id int64, maneuver, power, target string) string {
	t.Helper()
	return rollsFromTheActions(t, f, id, "manobra/"+maneuver+"/em-seguida/"+power, target)
}

// O LUTADOR DE CHÃO: o agarrar com Sucesso abre a derrubada livre por 1 PM, uma por agarrar. A Falha não abre.
func TestTheGroundFighterTripsForOneManaAfterAGrappleRuledSuccess(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Lutador", 1)
	seedPowerChoices(t, f.s, id, `["`+groundFighter+`"]`, "{}")
	if refusal := rollsFromTheActions(t, f, id, "manobra/agarrar", goblin); refusal != "" {
		t.Fatalf("agarrar foi recusado: %q", refusal)
	}
	if refusal := followsUp(t, f, id, "derrubar", groundFighter, goblin); refusal == "" {
		t.Error("a derrubada saiu antes da decisão do mestre sobre o agarrar")
	}
	if refusal := gmRefusal(t, f, "/ataque/julgar/falha"); refusal != "" {
		t.Fatalf("a Falha do agarrar foi recusada: %q", refusal)
	}
	if refusal := followsUp(t, f, id, "derrubar", groundFighter, goblin); !strings.Contains(refusal, "Sucesso") {
		t.Errorf("a derrubada depois de um agarrar com Falha não foi recusada pelo Sucesso que ela pede: %q", refusal)
	}

	g, other, enemy := barbarianOnTurn(t)
	seedClasse(t, g.s, other, "Lutador", 1)
	seedPowerChoices(t, g.s, other, `["`+groundFighter+`"]`, "{}")
	if refusal := rollsFromTheActions(t, g, other, "manobra/agarrar", enemy); refusal != "" {
		t.Fatalf("agarrar foi recusado: %q", refusal)
	}
	// A CARTA DO MESTRE diz, na face, o que o Sucesso abre: é a consequência da decisão que ele vai tomar.
	page := html.UnescapeString(g.requests(t, g.gm, http.MethodGet, g.tableUrl(), "").Body.String())
	face := page[strings.Index(page, `id="table-attack"`):]
	face = face[:strings.Index(face, `id="gm-attack-details"`)]
	if !strings.Contains(face, "O Sucesso abre derrubar em seguida (Lutador de Chão, 1 PM)") {
		t.Error("a face da carta do agarrar não diz ao mestre o que o Sucesso abre")
	}
	if refusal := gmRefusal(t, g, "/ataque/julgar/sucesso"); refusal != "" {
		t.Fatalf("o Sucesso do agarrar foi recusado: %q", refusal)
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
	if refusal := followsUp(t, g, other, "derrubar", groundFighter, enemy); !strings.Contains(refusal, "Sucesso") {
		t.Errorf("a segunda derrubada do mesmo agarrar não foi recusada: %q", refusal)
	}
}

// A MÃO NA BOCA: o acerto de um golpe com o Ataque Furtivo LIGADO abre o agarrar livre, sem PM. O erro não abre,
// e o golpe sem o Ataque Furtivo também não.
func TestTheHandOverMouthGrapplesAfterASneakHit(t *testing.T) {
	f, id, goblin := barbarianOnTurn(t)
	seedClasse(t, f.s, id, "Ladino", 1)
	seedPowerChoices(t, f.s, id, `["`+handOverMouth+`"]`, "{}")
	if refusal := rollsFromTheActions(t, f, id, "atacar/0", goblin); refusal != "" {
		t.Fatalf("o golpe sem o Ataque Furtivo foi recusado: %q", refusal)
	}
	gmChecks(t, f)
	if refusal := followsUp(t, f, id, "agarrar", handOverMouth, goblin); refusal == "" {
		t.Error("o acerto sem o Ataque Furtivo ligado abriu o agarrar da Mão na Boca")
	}

	g, other, enemy := barbarianOnTurn(t)
	seedClasse(t, g.s, other, "Ladino", 1)
	seedPowerChoices(t, g.s, other, `["`+handOverMouth+`"]`, "{}")
	turnSwitchOn(t, g.s, other, "ataque-furtivo")
	if refusal := rollsFromTheActions(t, g, other, "atacar/0", enemy); refusal != "" {
		t.Fatalf("o golpe furtivo foi recusado: %q", refusal)
	}
	gmRules(t, g, "errou")
	if refusal := followsUp(t, g, other, "agarrar", handOverMouth, enemy); refusal == "" {
		t.Error("o erro do golpe furtivo abriu o agarrar da Mão na Boca")
	}

	h, rogue, foe := barbarianOnTurn(t)
	seedClasse(t, h.s, rogue, "Ladino", 1)
	seedPowerChoices(t, h.s, rogue, `["`+handOverMouth+`"]`, "{}")
	turnSwitchOn(t, h.s, rogue, "ataque-furtivo")
	if refusal := rollsFromTheActions(t, h, rogue, "atacar/0", foe); refusal != "" {
		t.Fatalf("o golpe furtivo foi recusado: %q", refusal)
	}
	gmChecks(t, h)
	before := poolsOf(t, h.s, rogue).MpCurrent
	if refusal := followsUp(t, h, rogue, "agarrar", handOverMouth, foe); refusal != "" {
		t.Fatalf("o agarrar da Mão na Boca depois do acerto furtivo foi recusado: %q", refusal)
	}
	pa := pendingOf(t, h)
	if pa == nil || pa.Maneuver == nil || pa.Maneuver.Kind != "agarrar" || pa.FollowUpOf == nil || pa.FollowUpOf.PowerID != handOverMouth {
		t.Fatalf("a manobra em seguida não virou o agarrar da Mão na Boca: %+v", pa)
	}
	if after := poolsOf(t, h.s, rogue).MpCurrent; after != before {
		t.Errorf("o agarrar da Mão na Boca cobrou PM: %d, want %d", after, before)
	}
}

// AS AÇÕES OFERECEM A MANOBRA EM SEGUIDA, e a linha diz de que decisão ela depende. Não há mais declaração antes
// de rolar (decisão do dono): o Agarrar e o Atacar rolam direto.
func TestTheActionsOfferTheFollowUpManeuvers(t *testing.T) {
	f, lutador, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, lutador, "Lutador", 1)
	seedPowerChoices(t, f.s, lutador, `["`+groundFighter+`"]`, "{}")
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	for _, want := range []string{"Derrubar · Lutador de Chão", "depois do Sucesso no agarrar", "/acoes/manobra/derrubar/em-seguida/" + groundFighter, "1 PM"} {
		if !strings.Contains(surface, want) {
			t.Errorf("as Ações do Lutador de Chão não dizem %q", want)
		}
	}
	if strings.Contains(surface, "follow_up =") || strings.Contains(surface, "em seguida, como ação livre") {
		t.Error("as Ações ainda pedem a declaração da manobra em seguida, e ela saiu")
	}

	g, ladino, _ := barbarianOnTurn(t)
	seedClasse(t, g.s, ladino, "Ladino", 1)
	seedPowerChoices(t, g.s, ladino, `["`+handOverMouth+`"]`, "{}")
	surface = superficieDeAcoes(t, acoesDaMesa(t, g))
	for _, want := range []string{"Agarrar · Mão na Boca", "depois de acertar com o ataque furtivo"} {
		if !strings.Contains(surface, want) {
			t.Errorf("as Ações da Mão na Boca não dizem %q", want)
		}
	}
}

// A abertura da manobra em seguida é um sinal TIPADO, e a mesa o declara.
func TestTheFollowUpSignalIsDeclaredByTheTable(t *testing.T) {
	f, lutador, _ := barbarianOnTurn(t)
	seedClasse(t, f.s, lutador, "Lutador", 1)
	seedPowerChoices(t, f.s, lutador, `["`+groundFighter+`"]`, "{}")
	expectTypedSignalsDeclared(t, acoesDaMesa(t, f), "$follow_up_open")
}
