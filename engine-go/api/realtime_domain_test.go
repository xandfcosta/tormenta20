package api

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"t20engine/db"
	"t20engine/db/sqlcgen"
)

// newTestServer spins a migrated temp SQLite + a catalog-less Server. The domain
// helpers under test (authz + combatant resolution) never touch the engine, so a nil
// catalog snapshot is fine — this is the seam the WS gateway will reuse.
func newTestServer(t *testing.T) *Server {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return NewServer(Config{}, database, nil)
}

func seedUser(t *testing.T, s *Server, email string) int64 {
	t.Helper()
	u, err := s.queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		Email: email, Passwordhash: "x", Createdat: nowISO(), Updatedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed user %q: %v", email, err)
	}
	return u.ID
}

func seedCampaign(t *testing.T, s *Server, ownerID int64) int64 {
	t.Helper()
	c, err := s.queries.CreateCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: ownerID, Name: "Mesa", Createdat: nowISO(), Updatedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	return c.ID
}

// seedCharacter inserts a minimal valid character (JSON columns defaulted) with the given
// owner + vitals, returning its id.
func seedCharacter(t *testing.T, s *Server, ownerID int64, name string, hpCur, hpMax, mpCur, mpMax int64) int64 {
	t.Helper()
	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: "Soldado", Level: 1,
		HpMax: hpMax, HpCurrent: hpCur, MpMax: mpMax, MpCurrent: mpCur,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: nowISO(), UpdatedAt: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed character %q: %v", name, err)
	}
	return id
}

func seedMember(t *testing.T, s *Server, campaignID, characterID int64, role string) {
	t.Helper()
	if _, err := s.queries.CreateMember(context.Background(), sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: characterID, Role: role, Addedat: nowISO(),
	}); err != nil {
		t.Fatalf("seed member: %v", err)
	}
}

func TestResolveRole(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	player := seedUser(t, s, "p@t.com")
	stranger := seedUser(t, s, "x@t.com")
	campaignID := seedCampaign(t, s, gm)
	pc := seedCharacter(t, s, player, "PC", 10, 10, 5, 5)
	seedMember(t, s, campaignID, pc, "player")

	cases := []struct {
		name       string
		userID     int64
		wantRole   string
		wantStatus int
	}{
		{"owner is gm", gm, "gm", 200},
		{"member is player", player, "player", 200},
		{"stranger forbidden", stranger, "", 403},
	}
	for _, c := range cases {
		role, status, err := s.resolveRole(ctx, c.userID, campaignID)
		if role != c.wantRole || status != c.wantStatus {
			t.Errorf("%s: role=%q status=%d err=%v, want role=%q status=%d", c.name, role, status, err, c.wantRole, c.wantStatus)
		}
	}
	if _, status, _ := s.resolveRole(ctx, gm, 999999); status != 404 {
		t.Errorf("missing campaign: status=%d, want 404", status)
	}
}

func TestResolveCombatant(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	player := seedUser(t, s, "p@t.com")
	stranger := seedUser(t, s, "x@t.com")
	campaignID := seedCampaign(t, s, gm)
	pc := seedCharacter(t, s, player, "Herói", 7, 12, 3, 8)
	seedMember(t, s, campaignID, pc, "player")
	loose := seedCharacter(t, s, player, "Solto", 5, 5, 0, 0) // not a member

	t.Run("owner resolves with vitals", func(t *testing.T) {
		got, status, err := s.resolveCombatant(ctx, player, campaignID, pc)
		if err != nil || status != 200 {
			t.Fatalf("status=%d err=%v", status, err)
		}
		want := combatant{characterID: pc, name: "Herói", hpCurrent: 7, hpMax: 12, mpCurrent: 3, mpMax: 8}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
	t.Run("gm resolves another player's pc", func(t *testing.T) {
		if _, status, err := s.resolveCombatant(ctx, gm, campaignID, pc); status != 200 || err != nil {
			t.Errorf("gm should resolve: status=%d err=%v", status, err)
		}
	})
	t.Run("stranger forbidden", func(t *testing.T) {
		if _, status, _ := s.resolveCombatant(ctx, stranger, campaignID, pc); status != 403 {
			t.Errorf("status=%d, want 403", status)
		}
	})
	t.Run("non-member character is bad request", func(t *testing.T) {
		if _, status, _ := s.resolveCombatant(ctx, player, campaignID, loose); status != 400 {
			t.Errorf("status=%d, want 400", status)
		}
	})
	t.Run("missing character 404", func(t *testing.T) {
		if _, status, _ := s.resolveCombatant(ctx, gm, campaignID, 999999); status != 404 {
			t.Errorf("status=%d, want 404", status)
		}
	})
}

func TestSessionForCaller(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	stranger := seedUser(t, s, "x@t.com")
	campaignID := seedCampaign(t, s, gm)
	sess, err := s.queries.CreateSession(ctx, sqlcgen.CreateSessionParams{
		Campaignid: campaignID, Sessionnumber: 1, Title: sql.NullString{String: "S1", Valid: true},
		Createdat: nowISO(), Updatedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	t.Run("gm gets session + role", func(t *testing.T) {
		got, role, status, err := s.sessionForCaller(ctx, gm, campaignID, sess.ID)
		if err != nil || status != 200 || role != "gm" || got.ID != sess.ID {
			t.Errorf("status=%d role=%q id=%d err=%v", status, role, got.ID, err)
		}
	})
	t.Run("stranger forbidden before session load", func(t *testing.T) {
		if _, _, status, _ := s.sessionForCaller(ctx, stranger, campaignID, sess.ID); status != 403 {
			t.Errorf("status=%d, want 403", status)
		}
	})
	t.Run("missing session 404", func(t *testing.T) {
		if _, _, status, _ := s.sessionForCaller(ctx, gm, campaignID, 999999); status != 404 {
			t.Errorf("status=%d, want 404", status)
		}
	})
}

func TestListMemberHelpers(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	p1 := seedUser(t, s, "p1@t.com")
	p2 := seedUser(t, s, "p2@t.com")
	campaignID := seedCampaign(t, s, gm)
	pcA := seedCharacter(t, s, p1, "A", 10, 10, 4, 4)
	pcB := seedCharacter(t, s, p2, "B", 6, 8, 2, 2)
	npc := seedCharacter(t, s, gm, "NPC", 20, 20, 0, 0)
	seedMember(t, s, campaignID, pcA, "player")
	seedMember(t, s, campaignID, pcB, "player")
	seedMember(t, s, campaignID, npc, "gm")

	players, err := s.listPlayerCombatants(ctx, campaignID)
	if err != nil || len(players) != 2 {
		t.Fatalf("players=%d err=%v, want 2", len(players), err)
	}
	if players[0].name != "A" || players[0].hpMax != 10 || players[1].name != "B" {
		t.Errorf("unexpected players: %+v", players)
	}

	ids, err := s.listMemberCharacterIds(ctx, campaignID)
	if err != nil || len(ids) != 3 {
		t.Fatalf("ids=%v err=%v, want 3 (players + gm entry)", ids, err)
	}
}
