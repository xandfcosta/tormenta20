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

// Autorização pelo roteador DE VERDADE, com uma sessão assinada DE VERDADE.
//
// Os outros testes HTTP deste pacote montam o próprio roteador e injetam o
// usuário na mão, então o `requireAuth` nunca executa neles. Estes passam pelo
// `s.Router()`, com o middleware, a tabela de rotas e o handler no caminho: uma
// rota registrada fora do grupo protegido, ou um handler que esquece o ajudante
// de autorização, reprova aqui.

// authed emite um JWT de verdade para o usuário e manda o pedido como ele
// mandaria.
func authed(t *testing.T, s *Server, UserID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), UserID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", UserID, err)
	}
	token, err := s.accountRules().signToken(user)
	if err != nil {
		t.Fatalf("assinar token: %v", err)
	}
	return sendRaw(t, s, method, path, body, "Bearer "+token)
}

// anon manda o pedido sem credencial nenhuma.
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
	path := "/campanhas/" + id64(campaign)

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

	// A regra do `account_middleware.go` existe de propósito: um token que
	// sobrevive ao dono não pode autenticar.
	t.Run("JWT válido de usuário deletado", func(t *testing.T) {
		ghost := seedUser(t, s, "fantasma@t20.local")
		user, err := s.queries.GetUserByID(context.Background(), ghost)
		if err != nil {
			t.Fatalf("carregar usuário: %v", err)
		}
		token, err := s.accountRules().signToken(user)
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

// Toda rota sob o `requireAuth` responde 401 sem credencial — nunca 200, e nunca
// 404, que significaria a rota estar fora do grupo protegido.
func TestProtectedRoutesRejectAnonymous(t *testing.T) {
	s := newTestServer(t)

	// O `/health` fica de fora de propósito: ele é anônimo por desenho, e é o
	// `healthcheck` do compose que bate nele.
	protected := []struct{ method, path string }{
		{http.MethodGet, "/campanhas"},
		{http.MethodPost, "/campanhas"},
		{http.MethodDelete, "/campanhas/1"},
		{http.MethodPost, "/campanhas/1/sessoes"},
		{http.MethodGet, "/personagens"},
		{http.MethodPatch, "/personagens/1/conditions"},
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
// A VARREDURA das escritas de ficha é o `TestNoSheetWriteAcceptsAStranger`, no
// roteador das cenas — mas ela filtra por `/personagens/{id}/` e não alcança esta
// rota, que atende em `/characters`. Uma rota só, então enumerar aqui não é
// remendo: é o conjunto inteiro.
func TestTheSurvivingCharacterWriteRejectsAStranger(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	estranho := seedUser(t, s, "estranho@t20.local")
	ficha := seedCharacter(t, s, dono, "Herói Alheio", 10, 10, 0, 0)

	rec := authed(t, s, estranho, http.MethodPatch, "/personagens/"+id64(ficha)+"/conditions",
		`{"activeConditions":["caido"]}`)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("um estranho recebeu %d ao mexer na ficha dos outros, esperado 403 (%s)",
			rec.Code, rec.Body.String())
	}
}

func id64(v int64) string {
	return strconv.FormatInt(v, 10)
}
