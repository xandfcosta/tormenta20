package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
)

// endScopeRouter mounts only the two scope-expiring routes, injecting `user`
// the way requireAuth would. The domain side of endScene/endDay is covered by
// TestEndSceneEndDay; what this file pins is the HTTP contract the sheet's
// "Encerrar cena/dia" buttons read — the `clearedScopes` delta the client uses
// to drop cached effects without a refetch.
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

func TestEndSceneRouteClearsSceneOnly(t *testing.T) {
	s := newTestServer(t)
	ownerID := seedUser(t, s, "dono@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")
	seedEffect(t, s, char, "buff-b", "day")

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: ownerID}), "/end-scene", char)

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
	ownerID := seedUser(t, s, "dono@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")
	seedEffect(t, s, char, "buff-b", "day")

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: ownerID}), "/end-day", char)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	// Both scopes, so the client drops day effects too — reporting only "day"
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
	ownerID := seedUser(t, s, "dono@t.com")
	strangerID := seedUser(t, s, "x@t.com")
	char := seedCharacter(t, s, ownerID, "PC", 10, 10, 5, 5)
	seedEffect(t, s, char, "buff-a", "scene")

	rec := postEndScope(t, endScopeRouter(s, AuthUser{ID: strangerID}), "/end-scene", char)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if got := effectScopes(t, s, char); len(got) != 1 {
		t.Errorf("effect should survive a rejected end-scene, scopes = %v", got)
	}
}
