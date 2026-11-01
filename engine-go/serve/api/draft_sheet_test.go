package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// draftSheet é uma ficha da lista pessoal, que não entrou em campanha nenhuma.
func draftSheet(t *testing.T) (sceneFixture, int64) {
	t.Helper()
	f := newSceneFixture(t)
	return f, seedCharacterAtLevel(t, f.s, f.player, "Aprendiz", "Arcanista", 3, 0, 0)
}

// O rascunho só monta: todo gesto de jogar é recusado no servidor, e a recusa diz o caminho.
func TestADraftDoesNotPlay(t *testing.T) {
	f, id := draftSheet(t)
	for _, path := range []string{
		"/vitais/pv/-1",
		"/pericias/rolar/Iniciativa",
		"/acoes/atacar/0",
		"/acoes/manobra/derrubar",
		"/acoes/investida/0",
		"/acoes/investida-imprudente/0",
		"/efeitos/condicao/caido",
		"/efeitos/aplica/armadura-arcana",
		"/efeitos/encerra/1",
		"/efeitos/postura/furia",
		"/efeitos/situacao",
		"/efeitos/situacao/degraus",
		"/magias/conjura/luz",
		"/itens/1/usa",
		"/poderes/usa/furia",
		"/poderes/postura/furia/entra",
	} {
		if refusal := sheetPost(t, f, id, path, ""); !strings.Contains(refusal, "rascunho") {
			t.Errorf("%s no rascunho devia ser recusado apontando a campanha, e a recusa foi %q", path, refusal)
		}
	}
	if pools := poolsOf(t, f.s, id); pools.HpCurrent != pools.HpMax {
		t.Errorf("a recusa gastou PV mesmo assim: %d de %d", pools.HpCurrent, pools.HpMax)
	}
}

// O que monta continua aberto no rascunho, e o que joga continua aberto na cópia.
func TestADraftBuildsAndACopyPlays(t *testing.T) {
	f, id := draftSheet(t)
	if refusal := sheetPost(t, f, id, "/nivel/Arcanista/1", ""); refusal != "" {
		t.Errorf("subir de nível no rascunho foi recusado: %q", refusal)
	}
	if refusal := sheetPost(t, f, f.charID, "/vitais/pv/-1", ""); refusal != "" {
		t.Errorf("gastar PV na ficha de campanha foi recusado: %q", refusal)
	}
}

// A tela não oferece o que o servidor recusaria.
func TestTheDraftSheetOffersOnlyWhatBuilds(t *testing.T) {
	f, id := draftSheet(t)
	page := func(tab string, who int64) string {
		return f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=%s", who, tab), "").Body.String()
	}
	playOnly := map[string]string{
		"expertises": "/pericias/rolar/",
		"combat":     "/vitais/pv/",
		"bag":        "/vitais/pm/",
	}
	for tab, gesture := range playOnly {
		if strings.Contains(page(tab, id), gesture) {
			t.Errorf("a aba %s do rascunho oferece %s", tab, gesture)
		}
	}
	draft := page("expertises", id)
	for _, want := range []string{"rascunho", "entre com ele numa campanha"} {
		if !strings.Contains(draft, want) {
			t.Errorf("o rascunho não traz %q", want)
		}
	}
	if strings.Contains(draft, "?tab=conditionals") {
		t.Error("o rascunho ainda tem a aba Efeitos")
	}
	if !strings.Contains(page("conditionals", id), "Perícias") {
		t.Error("o endereço da aba Efeitos no rascunho devia cair numa aba que existe")
	}

	copied := page("expertises", f.charID)
	for _, want := range []string{"/pericias/rolar/", "/vitais/pv/", "?tab=conditionals"} {
		if !strings.Contains(copied, want) {
			t.Errorf("a ficha de campanha perdeu %q: o controle não mede nada", want)
		}
	}
	if strings.Contains(copied, "entre com ele numa campanha") {
		t.Error("a ficha de campanha diz que é rascunho")
	}
}
