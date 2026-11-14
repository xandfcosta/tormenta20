package api

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// O HISTÓRICO DA ABA É DO SERVIDOR (ALE-637). A aba continua trocando só a cena, por morph; o endereço
// quem troca é a resposta (`sse.ReplaceURL`), e o cliente não escreve no histórico. O F5 abre a aba certa,
// e o voltar sai da ficha (decisão do dono).

// datastarTabGet é o `@get` que o toque numa aba manda, e o corpo da resposta.
func datastarTabGet(t *testing.T, f sceneFixture, path string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+f.token(t, f.player))
	req.Header.Set("Datastar-Request", "true")
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	return html.UnescapeString(rec.Body.String())
}

func TestTheTabResponseReplacesTheAddress(t *testing.T) {
	f := newSceneFixture(t)
	body := datastarTabGet(t, f, fmt.Sprintf("/personagens/%d?tab=combat&datastar=%%7B%%7D", f.charID))
	want := fmt.Sprintf(`history.replaceState({}, "", "/personagens/%d?tab=combat")`, f.charID)
	if !strings.Contains(body, want) {
		t.Errorf("a resposta da aba não troca o endereço para a aba pedida; esperava %s", want)
	}
}

func TestTheTabInsideTheTableKeepsTheSessionAddress(t *testing.T) {
	f := newSceneFixture(t)
	body := datastarTabGet(t, f, fmt.Sprintf("/personagens/%d?tab=combat&embutida=1", f.charID))
	if strings.Contains(body, "replaceState") {
		t.Error("a ficha dentro da Mesa trocou o endereço, que é o da sessão")
	}
}

func TestTheTabLinkDoesNotWriteHistory(t *testing.T) {
	f := newSceneFixture(t)
	page := html.UnescapeString(spellScreen(t, f, f.charID))
	if strings.Contains(page, "history.pushState") || strings.Contains(page, "history.replaceState") {
		t.Error("a aba ainda escreve no histórico pelo cliente")
	}
	if !strings.Contains(page, "@get(el.href)") {
		t.Error("a aba deixou de trocar só a cena (@get(el.href))")
	}
}
