package api

import (
	"net/http"
	"strings"
	"testing"

	"t20engine/domain/live"
)

// O TESTE ESTENDIDO na mesa, de ponta a ponta (ver GLOSSARY, **teste estendido**): o mestre abre o contador pelo
// menu, marca cada resultado na faixa, desfaz e encerra; o jogador vê a mesma faixa sem os botões.

func TestTheMasterOpensTheExtendedTestAndOnlyHeMarksIt(t *testing.T) {
	f := newSceneFixture(t)
	f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	base := f.tableUrl() + "/teste-estendido"
	open := `{"ext_name":"Atravessar o Pântano dos Juncos","ext_required":5,"ext_each":false}`

	if body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(body, `id="gm-extended"`) {
		t.Error("a mesa do mestre não tem o diálogo de abrir o teste estendido")
	}
	if got := f.requests(t, f.player, http.MethodPost, base+"/abrir", open).Code; got != http.StatusForbidden {
		t.Errorf("o jogador abriu o teste estendido com %d; want 403", got)
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, base+"/abrir", `{"ext_name":" ","ext_required":5}`).Body.String()); !strings.Contains(refusal, "o que é") {
		t.Errorf("abrir sem nome disse %q", refusal)
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, base+"/abrir", open).Body.String()); refusal != "" {
		t.Fatalf("abrir o teste estendido foi recusado: %q", refusal)
	}

	for _, mark := range []string{"duplo", "falha"} {
		if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, base+"/marcar/"+mark, `{}`).Body.String()); refusal != "" {
			t.Fatalf("marcar %s foi recusado: %q", mark, refusal)
		}
	}
	f.requests(t, f.player, http.MethodPost, base+"/marcar/sucesso", `{}`)
	if got := stateOf(t, f.s.sessions, f.sessionID).ExtendedTest.TallyOf(""); got.Successes != 2 || got.Failures != 1 {
		t.Fatalf("o contador ficou %+v; want os 2 do sucesso que conta dois e 1 falha, sem a marca do jogador", got)
	}

	gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	for _, want := range []string{`aria-label="Teste estendido"`, "Atravessar o Pântano dos Juncos", "2/5", `aria-label="Marcar um sucesso"`} {
		if !strings.Contains(gm, want) {
			t.Errorf("a mesa do mestre não tem %s", want)
		}
	}
	// A FAIXA É A CONTA, e as marcas moram no cartão do mestre (ver GLOSSARY, **cartão flutuante**).
	if strings.Contains(sectionOf(gm, "table-extended"), "Marcar um sucesso") {
		t.Error("a faixa do teste estendido tem os botões de marcar: eles moram no cartão Marcar")
	}
	marks := sectionOf(gm, "table-extended-marks")
	requireFloatingCard(t, marks, "Marcar")
	if !strings.Contains(marks, "Marcar um sucesso") || !strings.Contains(marks, `class="ds-float-count">2/5<`) {
		t.Error("o cartão Marcar não tem o sucesso ou não diz o placar 2/5 na cabeça")
	}
	player := f.requests(t, f.player, http.MethodGet, f.tableUrl(), "").Body.String()
	if sectionOf(player, "table-extended-marks") != "" {
		t.Error("a mesa do jogador tem o cartão Marcar, que é do mestre")
	}
	if !strings.Contains(player, "2/5") || !strings.Contains(player, "Atravessar o Pântano dos Juncos") {
		t.Error("a mesa do jogador não mostra o contador")
	}
	if strings.Contains(player, `aria-label="Marcar um sucesso"`) {
		t.Error("a mesa do jogador tem o botão de marcar: só o mestre marca")
	}

	f.requests(t, f.gm, http.MethodPost, base+"/desfazer", `{}`)
	if got := stateOf(t, f.s.sessions, f.sessionID).ExtendedTest.TallyOf(""); got.Failures != 0 || got.Successes != 2 {
		t.Errorf("desfazer deixou %+v; want a falha tirada", got)
	}
	for range live.ExtendedFailureLimit {
		f.requests(t, f.gm, http.MethodPost, base+"/marcar/falha", `{}`)
	}
	if body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String(); !strings.Contains(body, "Falha total") {
		t.Error("três falhas e a faixa não diz a falha total")
	}
	f.requests(t, f.gm, http.MethodPost, base+"/encerrar", `{}`)
	if st := stateOf(t, f.s.sessions, f.sessionID); st.ExtendedTest != nil {
		t.Errorf("encerrar deixou o teste %q aberto", st.ExtendedTest.Name)
	}
}

func TestEachLineOfTheQueueIsMarkedOnItsOwnLine(t *testing.T) {
	f := newSceneFixture(t)
	goblin := f.actionSceneWithTheGoblinFirst(t, "Arcanista", f.charID, 1)
	base := f.tableUrl() + "/teste-estendido"
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, base+"/abrir", `{"ext_name":"Avalanche · Correr","ext_required":5,"ext_each":true}`).Body.String()); refusal != "" {
		t.Fatalf("abrir a contagem de cada um foi recusado: %q", refusal)
	}
	body := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(body, `aria-label="Sucesso de Goblin"`) {
		t.Fatal("a faixa do mestre não tem o sucesso do Goblin")
	}
	// O BOTÃO DA LINHA posta no caminho da linha: o do grupo cairia na recusa "é de cada um".
	if !strings.Contains(body, base+"/linha/"+goblin+"/marcar/sucesso") {
		t.Error("o sucesso do Goblin não posta na contagem dele")
	}
	if refusal := tableRefusal(t, f.requests(t, f.gm, http.MethodPost, base+"/linha/"+goblin+"/marcar/sucesso", `{}`).Body.String()); refusal != "" {
		t.Fatalf("marcar o sucesso do Goblin foi recusado: %q", refusal)
	}
	if got := stateOf(t, f.s.sessions, f.sessionID).ExtendedTest.TallyOf(goblin); got.Successes != 1 {
		t.Errorf("o Goblin ficou com %+v; want 1 sucesso", got)
	}
}
