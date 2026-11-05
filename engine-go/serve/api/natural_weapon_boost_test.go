package api

import (
	"fmt"
	"strings"
	"testing"
)

// O IMPULSO NO ATAQUE EXTRA da arma natural (decisão do dono).

// O impulso soma ao PM do extra (decisão do dono), e o cancelamento devolve os dois.
func TestTheExtraAttackTakesTheBoost(t *testing.T) {
	f, id, goblin := minotaurOnTurn(t)
	seedClasse(t, f.s, id, "Guerreiro", 5)
	horns := naturalIndex(t, f, "Chifres")
	rollsFromTheActions(t, f, id, "atacar/0", goblin)
	gmRules(t, f, "errou")
	before := mpOf(t, f.s, id)
	plain := attackByWeapon(t, f, id)["Chifres"]

	chosen := fmt.Sprintf(`{"turn_target":%q,"boost_weapon":%d,"boost_power":"class.guerreiro.ataque-especial-4","boost_steps":1,"boost_split":"attack"}`, goblin, horns)
	if refusal := sceneRefusal(f.posts(t, f.player, fmt.Sprintf("/personagens/%d/acoes/ataque-extra/%d?embutida=1", id, horns), chosen)); refusal != "" {
		t.Fatalf("o extra com impulso foi recusado: %q", refusal)
	}
	pa := pendingOf(t, f)
	if pa == nil || pa.BoostPowerID == "" || pa.ExtraOf == "" {
		t.Fatalf("o extra devia levar o impulso: %+v", pa)
	}
	if pa.Total-pa.Roll <= plain {
		t.Errorf("o extra rolou %+d, e sem impulso são %+d", pa.Total-pa.Roll, plain)
	}
	if got := mpOf(t, f.s, id); got != before-3 {
		t.Errorf("1 PM do extra e 2 do impulso: %d virou %d", before, got)
	}
	if page := f.requests(t, f.gm, "GET", f.tableUrl(), "").Body.String(); !strings.Contains(page, "Ataque extra · Ataque Especial · 3 PM") {
		t.Error("o cartão não diz o extra, o impulso e o preço")
	}
	gmCancels(t, f)
	if got := mpOf(t, f.s, id); got != before {
		t.Errorf("o extra cancelado devolve os dois: %d virou %d", before, got)
	}
}

// A linha do extra oferece a mesma tira de impulso da linha de ataque.
func TestTheExtraLineOffersTheBoost(t *testing.T) {
	f, id, _ := minotaurOnTurn(t)
	seedClasse(t, f.s, id, "Guerreiro", 5)
	surface := superficieDeAcoes(t, acoesDaMesa(t, f))
	at := strings.Index(surface, "Ataque extra · Chifres")
	if at < 0 {
		t.Fatal("as Ações não oferecem o ataque extra")
	}
	// RECORTADA À LINHA: a de ataque, logo acima, oferece o mesmo impulso.
	line := surface[at:]
	if next := strings.Index(line, "ds-turn-action"); next >= 0 {
		line = line[:next]
	}
	if !strings.Contains(line, "Ataque Especial") {
		t.Error("a linha do extra não oferece o impulso")
	}
}
