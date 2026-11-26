package api

import (
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O PERIGO NA MESA, de ponta a ponta (ver GLOSSARY, **perigo**): a página Perigos do mestre dispara a armadilha e
// expõe ao ambiente, a faixa do disparo é só dele, e o que dura vira a marca que rola na vez de quem está nele.

func TestTheMasterFiresTheTrapAndOnlyHeSeesTheBand(t *testing.T) {
	f := newSceneFixture(t)
	f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	page := f.tableUrl() + "/perigos"

	if got := f.requests(t, f.player, http.MethodGet, page, "").Code; got != http.StatusForbidden {
		t.Errorf("o jogador abriu a página Perigos com %d; want 403", got)
	}
	body := f.requests(t, f.gm, http.MethodGet, page, "").Body.String()
	for _, want := range []string{`aria-label="Disparar Fosso Camuflado"`, `aria-label="Expor a Ácido · imersão total"`} {
		if !strings.Contains(body, want) {
			t.Errorf("a página Perigos do mestre não tem %s", want)
		}
	}
	if table := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(table, `href="`+page+`"`) {
		t.Error("o cabeçalho da mesa do mestre não leva à página Perigos")
	}

	fire := page + "/fosso-camuflado/disparar"
	f.requests(t, f.player, http.MethodPost, fire, "")
	if fired := stateOf(t, f.s.sessions, f.sessionID).LastHazard; fired != nil {
		t.Fatalf("o jogador disparou %s; quem dispara é o mestre", fired.Hazard)
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, fire, `{}`).Body.String()); refusal != "" {
		t.Fatalf("disparar o fosso foi recusado: %q", refusal)
	}
	for _, where := range []string{f.tableUrl(), page} {
		body := f.requests(t, f.gm, http.MethodGet, where, "").Body.String()
		if !strings.Contains(body, `aria-label="Perigo disparado"`) || !strings.Contains(body, "Reflexos evita · CD 20") {
			t.Errorf("%s: o mestre não vê a faixa do fosso com a resistência e a CD", where)
		}
	}
	if body := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String(); strings.Contains(body, "Fosso Camuflado") {
		t.Error("a mesa do jogador mostra o fosso disparado: a CD e o dano são da ficha do perigo")
	}

	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, page+"/nao-existe/disparar", `{}`).Body.String()); !strings.Contains(refusal, "não está no catálogo") {
		t.Errorf("disparar um perigo que não existe disse %q", refusal)
	}
}

func TestTheGasMarksTheGoblinAndRollsOnItsTurn(t *testing.T) {
	f := newSceneFixture(t)
	goblin := f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/perigos/gas-venenoso/disparar", `{}`).Body.String()); refusal != "" {
		t.Fatalf("disparar o gás foi recusado: %q", refusal)
	}
	if body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(body, "Dura ") {
		t.Error("a faixa do gás não diz quantas rodadas ele dura")
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/marcar", "")
	if marks := entryOf(t, stateOf(t, f.s.sessions, f.sessionID), goblin).Marks; len(marks) != 1 || marks[0].Source != "Gás Venenoso" {
		t.Fatalf("o Goblin marcado com o gás tem %+v", marks)
	}
	if body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(body, `aria-label="Rolar Gás Venenoso na vez de Goblin"`) {
		t.Fatal("na vez do Goblin, a mesa do mestre não oferece o Rolar do gás")
	}
	f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/iniciativa/"+goblin+"/na-vez/0/rolar", `{"due_hand_0":5}`)
	cast := stateOf(t, f.s.sessions, f.sessionID).LastSpellCast
	if cast == nil || cast.Kind != live.RollOfTheMark || cast.Head() != "Gás Venenoso · início do turno de Goblin" ||
		cast.Parts[0].Total != 5 || cast.Parts[0].Kind != "perda de vida por veneno" {
		t.Fatalf("a faixa da vez diz %+v; want os 5 da mesa no 1d12 do gás", cast)
	}
}

func TestTheMasterExposesToTheAcidAndTheMarkStaysUntilHeTakesIt(t *testing.T) {
	f := newSceneFixture(t)
	f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/perigos/acido/expor/exposicao", `{}`).Body.String()); refusal != "" {
		t.Fatalf("expor ao ácido foi recusado: %q", refusal)
	}
	mark := stateOf(t, f.s.sessions, f.sessionID).Markable
	if mark == nil || mark.Source != "Ácido · exposição" || mark.Counted || mark.Parts[0].Dice != "1d6" {
		t.Fatalf("a exposição ao ácido deixou a marca %+v; want 1d6 por rodada, até o mestre tirar", mark)
	}
}
