package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"t20engine/plataforma"
	"testing"

	"github.com/go-chi/chi/v5"
	"t20engine/db"
	"t20engine/db/sqlcgen"
)

// inviteFixture spins the real schema up on a throwaway file DB and returns a
// server plus a campaign holding `token` as its invite. A fake would have to
// impersonate sqlc's generated queries, and the behaviour under test is
// precisely "what does a lookup MISS look like" — so the real query answers.
func inviteFixture(t *testing.T, token string) (*Server, http.Handler) {
	t.Helper()
	sqlDB, err := db.Open(filepath.Join(t.TempDir(), "invites_test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	queries := sqlcgen.New(sqlDB)
	user, err := queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email: "gm@test.local", Passwordhash: "x", Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	campaign, err := queries.CreateCampaign(ctx, sqlcgen.CreateCampaignParams{
		Ownerid: user.ID, Name: "Mesa do Beco", Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if _, err := queries.SetInviteToken(ctx, sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: token, Valid: true}, UpdatedAt: plataforma.NowISO(), ID: campaign.ID,
	}); err != nil {
		t.Fatalf("set invite token: %v", err)
	}

	s := &Server{db: sqlDB, queries: queries}
	r := chi.NewRouter()
	r.Get("/invites/{token}", s.handleResolveInvite)
	return s, r
}

func getInvite(t *testing.T, h http.Handler, token string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/invites/"+token, nil))
	return rec
}

func TestResolveInviteReturnsPreview(t *testing.T) {
	_, h := inviteFixture(t, "token-vivo")

	rec := getInvite(t, h, "token-vivo")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v (body %q)", err, rec.Body.String())
	}
	// camelCase or the join form reads an undefined campaignId (ALE-18).
	if body["campaignName"] != "Mesa do Beco" {
		t.Errorf("campaignName = %v, want %q", body["campaignName"], "Mesa do Beco")
	}
	if _, ok := body["campaignId"].(float64); !ok {
		t.Errorf("campaignId ausente ou não numérico: %v", body["campaignId"])
	}
}

// The regression this file exists for: an unknown token used to answer 200 with
// a `null` body, so a dead invite reached the client as a SUCCESS carrying no
// campaign — the join screen could not tell it apart from one still loading
// (ALE-80).
func TestResolveInviteRejectsUnknownToken(t *testing.T) {
	_, h := inviteFixture(t, "token-vivo")

	rec := getInvite(t, h, "token-que-nao-existe")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 (body %q)", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body == "null\n" || body == "null" {
		t.Error("um convite inexistente ainda responde `null` em vez de um erro")
	}
}

// A rotated invite invalidates the previous link — the old token must read as
// gone, not as an empty success.
func TestResolveInviteRejectsRotatedToken(t *testing.T) {
	s, h := inviteFixture(t, "token-antigo")
	if _, err := s.queries.SetInviteToken(context.Background(), sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: "token-novo", Valid: true}, UpdatedAt: plataforma.NowISO(), ID: 1,
	}); err != nil {
		t.Fatalf("rotate: %v", err)
	}

	if rec := getInvite(t, h, "token-antigo"); rec.Code != http.StatusNotFound {
		t.Errorf("token rotacionado: status = %d, want 404", rec.Code)
	}
	if rec := getInvite(t, h, "token-novo"); rec.Code != http.StatusOK {
		t.Errorf("token novo: status = %d, want 200", rec.Code)
	}
}
