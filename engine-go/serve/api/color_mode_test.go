package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"t20engine/infra/db/sqlcgen"
)

// O MODO DE COR, pelo roteador de verdade (ALE-479).
//
// O que o servidor DECIDE é o que se prende aqui: o que o formulário grava, o
// que ele recusa, para onde ele volta, e que a página sai no modo escolhido já
// na resposta. A aparência de cada modo é do e2e, que mede o desenhado.

func postColorMode(t *testing.T, s *Server, mode, back string) *httptest.ResponseRecorder {
	t.Helper()
	form := url.Values{"modo": {mode}, "voltar": {back}}
	req := httptest.NewRequest(http.MethodPost, "/modo", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

func colorCookieOf(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "color_mode" {
			return c
		}
	}
	t.Fatalf("a resposta não escreveu o cookie color_mode; cabeçalhos: %v", rec.Header())
	return nil
}

func TestTheColorModeFormStoresTheChoiceAndGoesBack(t *testing.T) {
	s := newTestServer(t)

	rec := postColorMode(t, s, "escuro", "/sistema")
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, quero 303; corpo: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Location"); got != "/sistema" {
		t.Errorf("voltou para %q, quero /sistema", got)
	}
	if c := colorCookieOf(t, rec); c.Value != "dark" || c.MaxAge <= 0 {
		t.Errorf("cookie = %q (MaxAge %d), quero dark e persistente", c.Value, c.MaxAge)
	}

	if c := colorCookieOf(t, postColorMode(t, s, "claro", "/")); c.Value != "light" {
		t.Errorf("claro gravou %q, quero light", c.Value)
	}
}

// Voltar a seguir o sistema APAGA a escolha, em vez de gravar "sistema": quem
// não escolheu não tem modo.
func TestFollowingTheSystemAgainForgetsTheChoice(t *testing.T) {
	s := newTestServer(t)

	c := colorCookieOf(t, postColorMode(t, s, "sistema", "/"))
	if c.MaxAge >= 0 {
		t.Errorf("MaxAge = %d, quero negativo (apagar o cookie)", c.MaxAge)
	}
}

func TestAnUnknownColorModeIsRefusedNamingTheValue(t *testing.T) {
	s := newTestServer(t)

	rec := postColorMode(t, s, "sepia", "/")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, quero 400", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `"sepia"`) || !strings.Contains(body, `"escuro"`) {
		t.Errorf("a recusa não diz o valor ofensor e os aceitos: %q", body)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Error("uma recusa gravou cookie")
	}
}

// O `voltar` é texto do cliente: o que não é caminho DESTE app volta para a
// raiz, e não para fora.
func TestTheColorModeNeverRedirectsOutsideTheApp(t *testing.T) {
	s := newTestServer(t)

	for _, back := range []string{"//outro.site/x", "https://outro.site", "/\\outro.site", ""} {
		if got := postColorMode(t, s, "claro", back).Header().Get("Location"); got != "/" {
			t.Errorf("voltar=%q levou a %q, quero /", back, got)
		}
	}
}

// A PÁGINA SAI NO MODO ESCOLHIDO já na resposta, e sem escolha ela não diz
// modo nenhum — é o CSS que segue o sistema operacional. Um cookie adulterado
// é tratado como "sem escolha".
func TestThePageComesBackInTheChosenMode(t *testing.T) {
	s, owner := hubFixture(t)
	user, err := s.queries.GetUserByID(context.Background(), owner)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	session, err := s.accountGate().SignSession(user)
	if err != nil {
		t.Fatalf("assinar: %v", err)
	}
	// O hub, que ainda veste o grimório: o modo é da PESSOA e vale para toda
	// cena — a casca o escreve no `<html>` sem perguntar qual sistema veste.
	page := func(mode string) string {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{Name: s.cfg.CookieName, Value: session})
		if mode != "" {
			req.AddCookie(&http.Cookie{Name: "color_mode", Value: mode})
		}
		rec := httptest.NewRecorder()
		s.WebRouter().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("/ com modo %q: status %d", mode, rec.Code)
		}
		return rec.Body.String()
	}

	if body := page("dark"); !strings.Contains(body, `data-mode="dark"`) {
		t.Error("com o cookie dark, o <html> não saiu com data-mode=\"dark\"")
	}
	for _, mode := range []string{"", "roxo"} {
		if body := page(mode); strings.Contains(body, "data-mode=") {
			t.Errorf("com o cookie %q, o <html> afirmou um modo", mode)
		}
	}
}

// A folha viva veste o sistema novo, e o resto do app continua no grimório até
// a vez de cada tela: é a transição POR ESCOPO da ALE-497.
func TestOnlyTheMigratedSceneWearsTheNewSystem(t *testing.T) {
	s, owner := hubFixture(t)

	shell := regexp.MustCompile(`<section[^>]*data-slot="scene-shell"[^>]*>`)
	scopeOf := func(path string) string {
		tag := shell.FindString(pedeHub(t, s, owner, http.MethodGet, path).Body.String())
		if tag == "" {
			t.Fatalf("%s não desenhou a casca", path)
		}
		return tag
	}

	if tag := scopeOf("/sistema"); !strings.Contains(tag, "scene-ds") || strings.Contains(tag, "scene-grimorio") {
		t.Errorf("a folha viva não saiu só no escopo scene-ds: %s", tag)
	}
	if tag := scopeOf("/"); strings.Contains(tag, "scene-ds") || !strings.Contains(tag, "scene-grimorio") {
		t.Errorf("o hub, que ainda não migrou, não saiu no grimório: %s", tag)
	}
}

// O SELETOR DIZ QUAL MODO ESTÁ VALENDO: sem isso a pessoa aperta "Escuro" sem
// saber se já estava nele. Sem escolha, é "Do sistema" que vem marcado.
func TestTheColorModeSwitchMarksTheModeInForce(t *testing.T) {
	s, owner := hubFixture(t)
	pressed := regexp.MustCompile(`<button[^>]*value="([a-z]+)"[^>]*aria-pressed="true"|<button[^>]*aria-pressed="true"[^>]*value="([a-z]+)"`)
	markedWith := func(mode string) []string {
		req := httptest.NewRequest(http.MethodGet, "/sistema", nil)
		session, err := s.accountGate().SignSession(mustUser(t, s, owner))
		if err != nil {
			t.Fatalf("assinar: %v", err)
		}
		req.AddCookie(&http.Cookie{Name: s.cfg.CookieName, Value: session})
		if mode != "" {
			req.AddCookie(&http.Cookie{Name: "color_mode", Value: mode})
		}
		rec := httptest.NewRecorder()
		s.WebRouter().ServeHTTP(rec, req)
		var got []string
		for _, m := range pressed.FindAllStringSubmatch(rec.Body.String(), -1) {
			got = append(got, m[1]+m[2])
		}
		return got
	}

	for cookie, want := range map[string]string{"dark": "escuro", "light": "claro", "": "sistema"} {
		if got := markedWith(cookie); len(got) != 1 || got[0] != want {
			t.Errorf("com o cookie %q, marcados = %v; quero só %q", cookie, got, want)
		}
	}
}

func mustUser(t *testing.T, s *Server, id int64) sqlcgen.User {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return user
}
