package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// Authorization through the REAL router, with a REAL signed session.
//
// Every other HTTP test in this package mounts its own chi router and injects
// the user by hand, so `requireAuth` had never once executed under test and 53
// of 56 handlers had never seen a request. What protected the app from a player
// calling the API directly was a Playwright spec asserting that a BUTTON is
// absent — the file itself calls that "the UX half".
//
// These go through `s.Router()` so the middleware, the route table and the
// handler are all in the path: a route registered outside the protected group,
// or a handler that forgets its authorization helper, fails here.

// authed issues a real JWT for the user and sends the request as they would.
func authed(t *testing.T, s *Server, UserID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), UserID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", UserID, err)
	}
	token, err := s.signToken(user)
	if err != nil {
		t.Fatalf("assinar token: %v", err)
	}
	return sendRaw(t, s, method, path, body, "Bearer "+token)
}

// anon sends the request with no credentials at all.
func anon(t *testing.T, s *Server, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	return sendRaw(t, s, method, path, "", "")
}

func sendRaw(t *testing.T, s *Server, method, path, body, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	s.Router().ServeHTTP(rec, req)
	return rec
}

func jsonField(t *testing.T, rec *httptest.ResponseRecorder, field string) any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("resposta não é JSON (%s): %v", rec.Body.String(), err)
	}
	return payload[field]
}

func TestRequireAuthRejectsMissingAndBrokenCredentials(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampaign(t, s, owner)
	path := "/campaigns/" + id64(campaign)

	t.Run("sem credencial nenhuma", func(t *testing.T) {
		if rec := anon(t, s, http.MethodGet, path); rec.Code != http.StatusUnauthorized {
			t.Fatalf("esperado 401 sem token, veio %d", rec.Code)
		}
	})

	t.Run("Bearer malformado", func(t *testing.T) {
		rec := sendRaw(t, s, http.MethodGet, path, "", "Bearer não-é-um-jwt")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("esperado 401 com token inválido, veio %d", rec.Code)
		}
	})

	// The rule `middleware.go:29` exists on purpose — a token outliving its user
	// must not authenticate. Nothing covered it before.
	t.Run("JWT válido de usuário deletado", func(t *testing.T) {
		ghost := seedUser(t, s, "fantasma@t20.local")
		user, err := s.queries.GetUserByID(context.Background(), ghost)
		if err != nil {
			t.Fatalf("carregar usuário: %v", err)
		}
		token, err := s.signToken(user)
		if err != nil {
			t.Fatalf("assinar: %v", err)
		}
		if _, err := s.db.ExecContext(context.Background(), "DELETE FROM users WHERE id = ?", ghost); err != nil {
			t.Fatalf("deletar usuário: %v", err)
		}

		rec := sendRaw(t, s, http.MethodGet, path, "", "Bearer "+token)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("esperado 401 para usuário deletado, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})
}

// Every route under `requireAuth` must answer 401 without credentials — never
// 200, and never 404, which would mean the route sits outside the guarded group.
func TestProtectedRoutesRejectAnonymous(t *testing.T) {
	s := newTestServer(t)

	// As SETE que sobraram da ALE-277. A lista era de trinta e seis, e ela
	// encolheu com as rotas — não por alguém ter tirado casos, mas porque o que
	// elas protegiam deixou de existir. O `/health` fica de fora de propósito:
	// ele é anônimo por desenho, e é o `healthcheck` do compose que bate nele.
	protected := []struct{ method, path string }{
		{http.MethodGet, "/campaigns"},
		{http.MethodPost, "/campaigns"},
		{http.MethodDelete, "/campaigns/1"},
		{http.MethodPost, "/campaigns/1/sessions"},
		{http.MethodGet, "/characters"},
		{http.MethodPatch, "/characters/1/conditions"},
	}

	for _, route := range protected {
		rec := anon(t, s, route.method, route.path)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: esperado 401 anônimo, veio %d", route.method, route.path, rec.Code)
		}
	}
}

// A ÚNICA escrita de ficha que sobrou na API responde 403 para um estranho.
//
// Aqui morava um guarda de VARREDURA — o `TestEveryCharacterRouteRejectsAnIntruder`
// lia as rotas de `/characters` do `server.go` e cobrava um 403 de cada uma
// (ALE-186). Ele saiu na ALE-277 com as 24 rotas que mediam, e o sucessor dele
// não é este caso: é o `TestNoSheetWriteAcceptsAStranger`, que faz a MESMA
// varredura no roteador das cenas, que é onde a ficha se escreve hoje.
//
// Este caso existe porque a varredura de lá não alcança a rota que ficou aqui:
// ela filtra por `/personagens/{id}/`, e esta atende em `/characters`. Uma rota
// só, então enumerar não é remendo — é o conjunto inteiro.
func TestTheSurvivingCharacterWriteRejectsAStranger(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	estranho := seedUser(t, s, "estranho@t20.local")
	ficha := seedCharacter(t, s, dono, "Herói Alheio", 10, 10, 0, 0)

	rec := authed(t, s, estranho, http.MethodPatch, "/characters/"+id64(ficha)+"/conditions",
		`{"activeConditions":["caido"]}`)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("um estranho recebeu %d ao mexer na ficha dos outros, esperado 403 (%s)",
			rec.Code, rec.Body.String())
	}
}

// Aqui morava o TestGetCampaignAuthorization, sobre a leitura de UMA campanha. A rota saiu na ALE-277.

// Aqui morava o TestCampaignWritesRejectNonOwner, sobre o PATCH e o convite da campanha. A rota saiu na ALE-277.

// Aqui morava o TestSessionRoutesRejectCrossCampaignAndNonOwner, sobre as rotas de sessão. A rota saiu na ALE-277.

func id64(v int64) string {
	return strconv.FormatInt(v, 10)
}

// Aqui morava o TestCampaignDescriptionBlankIsTheSameEitherWay, sobre a descrição em branco. A rota saiu na ALE-277.
