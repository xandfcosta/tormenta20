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

// O rascunho joga como a cópia da campanha: ser rascunho muda o nome, e não o que a ficha aceita.
func TestADraftPlaysLikeACopy(t *testing.T) {
	f, id := draftSheet(t)
	before := poolsOf(t, f.s, id)
	for _, path := range []string{"/vitais/pv/-1", "/vitais/pm/-1", "/efeitos/condicao/caido"} {
		if refusal := sheetPost(t, f, id, path, ""); refusal != "" {
			t.Errorf("%s no rascunho foi recusado: %q", path, refusal)
		}
	}
	after := poolsOf(t, f.s, id)
	if after.HpCurrent != before.HpCurrent-1 || after.MpCurrent != before.MpCurrent-1 {
		t.Errorf("o rascunho devia gastar 1 PV e 1 PM: de %+v para %+v", before, after)
	}
	for _, path := range []string{
		"/pericias/rolar/Iniciativa",
		"/efeitos/aplica/armadura-arcana",
		"/efeitos/encerra/1",
		"/magias/conjura/luz",
		"/itens/1/usa",
		"/poderes/usa/furia",
		"/poderes/postura/furia/entra",
	} {
		if refusal := sheetPost(t, f, id, path, ""); strings.Contains(refusal, "rascunho") {
			t.Errorf("%s foi recusado por ser rascunho: %q", path, refusal)
		}
	}
}

// A tela oferece os mesmos gestos da sessão, e só a etiqueta diz que é rascunho.
func TestTheDraftSheetOffersTheGesturesOfASession(t *testing.T) {
	f, id := draftSheet(t)
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=expertises", id), "").Body.String()
	for _, want := range []string{"/vitais/pv/", "?tab=conditionals", ">rascunho<"} {
		if !strings.Contains(page, want) {
			t.Errorf("o rascunho não traz %q", want)
		}
	}
	if strings.Contains(page, "entre com ele numa campanha") {
		t.Error("o rascunho ainda manda entrar numa campanha para jogar")
	}
	effects := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=conditionals", id), "").Body.String()
	if !strings.Contains(effects, "condition-picker") {
		t.Error("a aba Efeitos do rascunho não abriu")
	}
	copied := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=expertises", f.charID), "").Body.String()
	if strings.Contains(copied, ">rascunho<") {
		t.Error("a ficha de campanha se diz rascunho")
	}
}

// Usar poder no rascunho cobra o PM de verdade, e a postura entra paga, como numa sessão.
func TestADraftUsesPowersAndPaysForThem(t *testing.T) {
	f := newSceneFixture(t)
	id := seedCharacterAtLevel(t, f.s, f.player, "Rascunho de bárbaro", "Bárbaro", 5, 0, 0)
	choiceCom(t, f, id, `["class.barbaro.brado-assustador"]`, `[]`)

	before := pm(t, f, id)
	if refusal := powerCommand(t, f, id, "usa/class.barbaro.brado-assustador", ""); refusal != "" {
		t.Fatalf("usar o poder no rascunho foi recusado: %q", refusal)
	}
	if spent := before - pm(t, f, id); spent != 1 {
		t.Errorf("o poder cobrou %d PM, quer 1", spent)
	}
	before = pm(t, f, id)
	if refusal := powerCommand(t, f, id, "postura/furia/entra", `{"stance_degrees":0}`); refusal != "" {
		t.Fatalf("entrar na Fúria no rascunho foi recusado: %q", refusal)
	}
	if pm(t, f, id) >= before {
		t.Error("entrar na Fúria não cobrou PM")
	}
}
