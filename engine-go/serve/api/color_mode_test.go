package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
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
