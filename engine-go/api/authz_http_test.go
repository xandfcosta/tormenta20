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
func authed(t *testing.T, s *Server, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário %d não existe: %v", userID, err)
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

	protected := []struct{ method, path string }{
		{http.MethodGet, "/auth/me"},
		{http.MethodGet, "/campaigns"},
		{http.MethodPost, "/campaigns"},
		{http.MethodGet, "/campaigns/1"},
		{http.MethodPatch, "/campaigns/1"},
		{http.MethodDelete, "/campaigns/1"},
		{http.MethodPost, "/campaigns/1/invite"},
		{http.MethodGet, "/campaigns/1/members"},
		{http.MethodPost, "/campaigns/1/members"},
		{http.MethodPatch, "/campaigns/1/members/1"},
		{http.MethodDelete, "/campaigns/1/members/1"},
		{http.MethodGet, "/campaigns/1/sessions"},
		{http.MethodPost, "/campaigns/1/sessions"},
		{http.MethodGet, "/campaigns/1/sessions/1"},
		{http.MethodPatch, "/campaigns/1/sessions/1"},
		{http.MethodDelete, "/campaigns/1/sessions/1"},
		{http.MethodPost, "/campaigns/1/sessions/1/start"},
		{http.MethodPost, "/campaigns/1/sessions/1/end"},
		{http.MethodPost, "/campaigns/1/sessions/1/clear-tracker"},
		{http.MethodGet, "/characters"},
		{http.MethodPost, "/characters"},
		{http.MethodGet, "/characters/1"},
		{http.MethodGet, "/characters/1/sheet"},
		{http.MethodGet, "/characters/1/campaigns"},
		{http.MethodPatch, "/characters/1/vitals"},
		{http.MethodPost, "/characters/1/damage"},
		{http.MethodPatch, "/characters/1/level"},
		{http.MethodPatch, "/characters/1/classes/level"},
		{http.MethodPatch, "/characters/1/abilities"},
		{http.MethodPatch, "/characters/1/proficiencies"},
		{http.MethodPatch, "/characters/1/conditions"},
		{http.MethodPost, "/characters/1/items"},
		{http.MethodPatch, "/characters/1/items/1"},
		{http.MethodDelete, "/characters/1/items/1"},
		{http.MethodPost, "/characters/1/items/1/consume"},
		{http.MethodPost, "/characters/1/expertises"},
		{http.MethodPatch, "/characters/1/expertises"},
		{http.MethodDelete, "/characters/1/expertises/atletismo"},
		{http.MethodPost, "/characters/1/spells"},
		{http.MethodDelete, "/characters/1/spells/bola-de-fogo"},
		{http.MethodPatch, "/characters/1/spells/bola-de-fogo/prepared"},
		{http.MethodPost, "/characters/1/spells/bola-de-fogo/cast"},
		{http.MethodPost, "/characters/1/active-effects"},
		{http.MethodPatch, "/characters/1/active-effects/1"},
		{http.MethodDelete, "/characters/1/active-effects/1"},
		{http.MethodPost, "/characters/1/end-scene"},
		{http.MethodPost, "/characters/1/end-day"},
	}

	for _, route := range protected {
		rec := anon(t, s, route.method, route.path)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: esperado 401 anônimo, veio %d", route.method, route.path, rec.Code)
		}
	}
}

func TestGetCampaignAuthorization(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	stranger := seedUser(t, s, "estranho@t20.local")
	campaign := seedCampaign(t, s, gm)
	hero := seedCharacter(t, s, player, "Herói", 10, 10, 0, 0)
	seedMember(t, s, campaign, hero, "player")
	path := "/campaigns/" + id64(campaign)

	t.Run("o mestre vê a campanha como gm", func(t *testing.T) {
		rec := authed(t, s, gm, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200 para o dono, veio %d (%s)", rec.Code, rec.Body.String())
		}
		if role := jsonField(t, rec, "role"); role != "gm" {
			t.Fatalf("papel esperado gm, veio %v", role)
		}
	})

	t.Run("o membro vê a campanha como player", func(t *testing.T) {
		rec := authed(t, s, player, http.MethodGet, path, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("esperado 200 para o membro, veio %d (%s)", rec.Code, rec.Body.String())
		}
		if role := jsonField(t, rec, "role"); role != "player" {
			t.Fatalf("papel esperado player, veio %v", role)
		}
	})

	t.Run("um estranho leva 403", func(t *testing.T) {
		if rec := authed(t, s, stranger, http.MethodGet, path, ""); rec.Code != http.StatusForbidden {
			t.Fatalf("esperado 403 para estranho, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("campanha inexistente é 404", func(t *testing.T) {
		if rec := authed(t, s, gm, http.MethodGet, "/campaigns/99999", ""); rec.Code != http.StatusNotFound {
			t.Fatalf("esperado 404, veio %d", rec.Code)
		}
	})
}

// The write block a player must never reach. Each case checks the refusal AND
// that the row is untouched — a 403 that already mutated is not a refusal.
func TestCampaignWritesRejectNonOwner(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampaign(t, s, gm)
	hero := seedCharacter(t, s, player, "Herói", 10, 10, 0, 0)
	seedMember(t, s, campaign, hero, "player")
	path := "/campaigns/" + id64(campaign)

	before, err := s.queries.GetCampaign(context.Background(), campaign)
	if err != nil {
		t.Fatalf("ler campanha: %v", err)
	}

	cases := []struct{ nome, method, path, body string }{
		{"editar", http.MethodPatch, path, `{"name":"Sequestrada"}`},
		{"excluir", http.MethodDelete, path, ""},
		{"gerar convite", http.MethodPost, path + "/invite", ""},
	}

	for _, c := range cases {
		t.Run(c.nome+" como membro é 403", func(t *testing.T) {
			rec := authed(t, s, player, c.method, c.path, c.body)
			if rec.Code != http.StatusForbidden {
				t.Fatalf("esperado 403, veio %d (%s)", rec.Code, rec.Body.String())
			}

			after, err := s.queries.GetCampaign(context.Background(), campaign)
			if err != nil {
				t.Fatalf("a campanha sumiu depois de um 403: %v", err)
			}
			if after.Name != before.Name {
				t.Fatalf("nome mudou apesar do 403: %q → %q", before.Name, after.Name)
			}
			if after.Invitetoken != before.Invitetoken {
				t.Fatalf("token de convite girou apesar do 403")
			}
		})
	}
}

// A session id from ANOTHER campaign must not resolve through a campaign the
// caller does own — the classic nested-route confusion.
func TestSessionRoutesRejectCrossCampaignAndNonOwner(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	other := seedUser(t, s, "outro@t20.local")
	mine := seedCampaign(t, s, gm)
	theirs := seedCampaign(t, s, other)
	foreignSession := seedSession(t, s, theirs)

	t.Run("sessão de outra campanha pela minha é 404", func(t *testing.T) {
		path := "/campaigns/" + id64(mine) + "/sessions/" + id64(foreignSession)
		rec := authed(t, s, gm, http.MethodGet, path, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("esperado 404 para id cruzado, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("iniciar sessão alheia é recusado", func(t *testing.T) {
		path := "/campaigns/" + id64(theirs) + "/sessions/" + id64(foreignSession) + "/start"
		rec := authed(t, s, gm, http.MethodPost, path, "")
		if rec.Code != http.StatusForbidden && rec.Code != http.StatusNotFound {
			t.Fatalf("esperado 403 ou 404 para campanha alheia, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})
}

// id64 formats a seeded row id for a URL — the package's `itoa` takes an int.
func id64(v int64) string {
	return strconv.FormatInt(v, 10)
}

// A descrição de campanha tinha DUAS grafias de "vazio": `handleCreateCampaign`
// usava `trimmedNull` (só-espaços → string vazia) e `handleUpdateCampaign`
// inlineava o trim gravando NULL. Mesma coluna, dois valores — o cliente recebia
// `""` de um caminho e `null` do outro, para a mesma entrada do usuário.
func TestCampaignDescriptionBlankIsTheSameEitherWay(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")

	criada := authed(t, s, gm, http.MethodPost, "/campaigns",
		`{"name":"Mesa Nova","description":"   "}`)
	if criada.Code != http.StatusCreated && criada.Code != http.StatusOK {
		t.Fatalf("criar: esperado 2xx, veio %d (%s)", criada.Code, criada.Body.String())
	}
	aoCriar := jsonField(t, criada, "description")

	id := int64(jsonField(t, criada, "id").(float64))
	editada := authed(t, s, gm, http.MethodPatch, "/campaigns/"+id64(id),
		`{"description":"   "}`)
	if editada.Code != http.StatusOK {
		t.Fatalf("editar: esperado 200, veio %d (%s)", editada.Code, editada.Body.String())
	}
	aoEditar := jsonField(t, editada, "description")

	if aoCriar != aoEditar {
		t.Fatalf("mesma entrada, valores diferentes: criar deu %#v, editar deu %#v", aoCriar, aoEditar)
	}
}
