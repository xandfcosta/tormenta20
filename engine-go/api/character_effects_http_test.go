package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"t20engine/db/sqlcgen"

	"github.com/go-chi/chi/v5"
)

// AS ROTAS DE ESCOPO DA FICHA (`/end-scene`, `/end-day`).
//
// Decisão do dono (ALE-223): encerrar cena e encerrar dia são do MESTRE e só
// existem DURANTE uma sessão. Isto INVERTE a regra da ALE-216, que recusava com
// mesa em curso e liberava fora dela — os botões saíram da ficha, e o servidor
// passou a pedir exatamente o contrário do que pedia.
//
// Duas coisas se provam aqui e em lugar nenhum mais: a autorização, e a
// PRECISÃO da pergunta — mestre de uma mesa em curso DESTE personagem, não
// mestre de qualquer coisa com uma sessão viva em algum lugar.

// endScopeRouter mounts only the two scope-expiring routes, injecting `user`
// the way requireAuth would. The domain side of endScene/endDay is covered by
// TestEndSceneEndDay; what this file pins is the HTTP contract — who may call,
// and the `clearedScopes` delta the caller uses to drop cached effects.
func endScopeRouter(s *Server, user AuthUser) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(context.WithValue(req.Context(), userCtxKey, user)))
		})
	})
	r.Post("/characters/{id}/end-scene", s.handleEndScene)
	r.Post("/characters/{id}/end-day", s.handleEndDay)
	return r
}

func postEndScope(t *testing.T, h http.Handler, path string, charID int64) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	url := "/characters/" + strconv.FormatInt(charID, 10) + path
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, url, nil))
	return rec
}

func clearedScopes(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	var body struct {
		ClearedScopes []string `json:"clearedScopes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	return body.ClearedScopes
}

// seedLiveSession puts the character at a table with a session RUNNING: a
// campaign it is a member of, holding a session moved to status 'active'.
// Returns the GM's user id, which is the only caller the routes now accept.
func seedLiveSession(t *testing.T, s *Server, gmID, charID int64) int64 {
	t.Helper()
	campaign := seedCampaign(t, s, gmID)
	seedMember(t, s, campaign, charID, "player")
	sid := seedSession(t, s, campaign)
	if _, err := s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		StartedAt: sql.NullString{String: nowISO(), Valid: true}, UpdatedAt: nowISO(), ID: sid,
	}); err != nil {
		t.Fatalf("start session: %v", err)
	}
	return sid
}

func TestEndSceneRouteClearsSceneOnly(t *testing.T) {
	s := newTestServer(t)
	gmID := seedUser(t, s, "gm@t.com")
	ownerID := seedUser(t, s, "dono@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")
	seedEffect(t, s, char, "buff-b", "day")
	seedLiveSession(t, s, gmID, char)

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: gmID}), "/end-scene", char)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if got := clearedScopes(t, rec); len(got) != 1 || got[0] != "scene" {
		t.Errorf("clearedScopes = %v, want [scene]", got)
	}
	if got := effectScopes(t, s, char); len(got) != 1 || got[0] != "day" {
		t.Errorf("remaining scopes = %v, want [day]", got)
	}
}

func TestEndDayRouteClearsBothScopes(t *testing.T) {
	s := newTestServer(t)
	gmID := seedUser(t, s, "gm@t.com")
	ownerID := seedUser(t, s, "dono@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")
	seedEffect(t, s, char, "buff-b", "day")
	seedLiveSession(t, s, gmID, char)

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: gmID}), "/end-day", char)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	// Both scopes, so the caller drops day effects too — reporting only "day"
	// would leave the cleared scene buffs painted on the sheet.
	got := clearedScopes(t, rec)
	if len(got) != 2 || got[0] != "scene" || got[1] != "day" {
		t.Errorf("clearedScopes = %v, want [scene day]", got)
	}
	if got := effectScopes(t, s, char); len(got) != 0 {
		t.Errorf("remaining scopes = %v, want []", got)
	}
}

func TestEndSceneRouteRejectsStranger(t *testing.T) {
	s := newTestServer(t)
	gmID := seedUser(t, s, "gm@t.com")
	ownerID := seedUser(t, s, "dono@t.com")
	strangerID := seedUser(t, s, "x@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")
	seedLiveSession(t, s, gmID, char)

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: strangerID}), "/end-scene", char)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if got := effectScopes(t, s, char); len(got) != 1 {
		t.Errorf("effect should survive a rejected end-scene, scopes = %v", got)
	}
}

// A metade que a ALE-223 inverteu: o DONO da ficha não encerra cena nem dia,
// nem com mesa em curso. Descanso é decisão da mesa, e a mesa é do mestre.
func TestEndScopeRoutesRefuseTheOwnerAtALiveTable(t *testing.T) {
	for _, path := range []string{"/end-scene", "/end-day"} {
		t.Run(path, func(t *testing.T) {
			s := newTestServer(t)
			gmID := seedUser(t, s, "gm@t.com")
			ownerID := seedUser(t, s, "dono@t.com")
			char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
			seedEffect(t, s, char, "buff-a", "scene")
			seedEffect(t, s, char, "buff-b", "day")
			seedLiveSession(t, s, gmID, char)

			rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: ownerID}), path, char)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 (body %q)", rec.Code, rec.Body.String())
			}
			if got := effectScopes(t, s, char); len(got) != 2 {
				t.Errorf("effects should survive a refused %s, scopes = %v", path, got)
			}
		})
	}
}

// A outra metade da inversão: SEM sessão em curso ninguém encerra nada — nem o
// mestre da campanha, nem o dono da ficha. As duas ações só existem enquanto se
// joga, e é isso que "não faz sentido quando se está editando uma ficha"
// significa em código.
func TestEndSceneRouteRefusedWithNoRunningSession(t *testing.T) {
	s := newTestServer(t)
	gmID := seedUser(t, s, "gm@t.com")
	ownerID := seedUser(t, s, "dono@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")
	campaign := seedCampaign(t, s, gmID)
	seedMember(t, s, campaign, char, "player")
	sid := seedSession(t, s, campaign)
	if _, err := s.queries.EndSession(context.Background(), sqlcgen.EndSessionParams{
		EndedAt: sql.NullString{String: nowISO(), Valid: true}, UpdatedAt: nowISO(), ID: sid,
	}); err != nil {
		t.Fatalf("end session: %v", err)
	}
	seedSession(t, s, campaign) // planned, never started

	for quem, id := range map[string]int64{"o mestre": gmID, "o dono da ficha": ownerID} {
		rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: id}), "/end-scene", char)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403 (body %q)", quem, rec.Code, rec.Body.String())
		}
	}
	if got := effectScopes(t, s, char); len(got) != 1 {
		t.Errorf("remaining scopes = %v, want [scene] — nada devia ter sido encerrado", got)
	}
}

// A PRECISÃO da pergunta, que é o que separa uma consulta de duas.
//
// O mestre tem uma campanha com sessão VIVA e outra, com este personagem, sem
// sessão nenhuma. Compor "é mestre deste personagem?" com "este personagem está
// numa mesa viva?" em dois booleanos separados responderia SIM às duas e
// liberaria — e ele estaria encerrando o dia de uma ficha que não está jogando.
func TestEndSceneRouteRefusesGmWhoseLiveSessionIsAnotherCampaign(t *testing.T) {
	s := newTestServer(t)
	gmID := seedUser(t, s, "gm@t.com")
	ownerID := seedUser(t, s, "dono@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")

	// Campanha A: o personagem está nela, e ela NÃO tem sessão rodando.
	quieta := seedCampaign(t, s, gmID)
	seedMember(t, s, quieta, char, "player")
	// Campanha B: o mesmo mestre, sessão rodando, e o personagem NÃO está nela.
	outroPc := seedCharacter(t, s, ownerID, "Outro", 10, 10, 5, 5)
	seedLiveSession(t, s, gmID, outroPc)

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: gmID}), "/end-scene", char)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (body %q)", rec.Code, rec.Body.String())
	}
	if got := effectScopes(t, s, char); len(got) != 1 {
		t.Errorf("remaining scopes = %v, want [scene]", got)
	}
}
