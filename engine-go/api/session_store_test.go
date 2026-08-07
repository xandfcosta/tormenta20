package api

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"t20engine/db/sqlcgen"
)

func seedSession(t *testing.T, s *Server, campaignID int64) int64 {
	t.Helper()
	sess, err := s.queries.CreateSession(context.Background(), sqlcgen.CreateSessionParams{
		Campaignid: campaignID, Sessionnumber: 1, Title: sql.NullString{String: "S", Valid: true},
		Createdat: nowISO(), Updatedat: nowISO(),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return sess.ID
}

func TestStorePersistLoadRoundTrip(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	store := s.sessions

	if _, err := store.load(ctx, sid); err != nil {
		t.Fatalf("initial load: %v", err)
	}
	if _, err := store.addInitiativeEntry(sid, npc("Goblin", 15)); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := store.nextTurn(sid); err != nil {
		t.Fatalf("nextTurn: %v", err)
	}
	if dirty, _ := store.persist(ctx, sid); dirty {
		t.Fatalf("persist should succeed, got dirty")
	}

	store.forget(sid) // drop the cache → next load re-hydrates from the DB
	loaded, err := store.load(ctx, sid)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(loaded.Initiative) != 1 || loaded.Initiative[0].Label != "Goblin" {
		t.Errorf("initiative=%+v, want one Goblin", loaded.Initiative)
	}
	if loaded.Round != 1 || loaded.TurnIndex != 0 {
		t.Errorf("round=%d turnIndex=%d, want 1/0", loaded.Round, loaded.TurnIndex)
	}
}

func TestStoreHydrateFromBlob(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	blob := `{"initiative":[{"id":"x","label":"Boss","initiative":9,"type":"npc"}],"round":2,"turnIndex":0}`
	if err := s.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: blob, UpdatedAt: nowISO(), ID: sid,
	}); err != nil {
		t.Fatalf("seed blob: %v", err)
	}
	loaded, err := s.sessions.load(ctx, sid)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(loaded.Initiative) != 1 || loaded.Initiative[0].Label != "Boss" || loaded.Round != 2 {
		t.Errorf("hydrated %+v, want Boss/round 2", loaded)
	}
}

func TestStoreRefreshCharacterMaxes(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	charID := seedCharacter(t, s, gm, "A", 7, 10, 3, 5) // real maxes 10/5
	store := s.sessions
	if _, err := store.load(ctx, sid); err != nil {
		t.Fatalf("load: %v", err)
	}
	// Entry carries STALE maxes + a live current the refresh must not touch.
	e := charEntry("A", 12, charID)
	stale, cur := int64(1), int64(4)
	e.HpMax, e.HpCurrent = &stale, &cur
	if _, err := store.addInitiativeEntry(sid, e); err != nil {
		t.Fatalf("add: %v", err)
	}
	got := store.refreshCharacterMaxes(ctx, sid)
	entry := got.Initiative[0]
	if entry.HpMax == nil || *entry.HpMax != 10 || entry.MpMax == nil || *entry.MpMax != 5 {
		t.Errorf("maxes not refreshed: hpMax=%v mpMax=%v, want 10/5", entry.HpMax, entry.MpMax)
	}
	if entry.HpCurrent == nil || *entry.HpCurrent != 4 {
		t.Errorf("hpCurrent should be untouched at 4, got %v", entry.HpCurrent)
	}
}

func TestStoreDirtyOnPersistFailure(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	store := s.sessions
	if _, err := store.addInitiativeEntry(sid, npc("x", 1)); err != nil {
		t.Fatalf("add: %v", err)
	}
	if d, _ := store.persist(ctx, sid); d {
		t.Fatalf("first persist should succeed")
	}
	_ = s.db.Close() // break the DB so the next write fails
	if d, _ := store.persist(ctx, sid); !d {
		t.Error("persist after DB close should report dirty")
	}
	if !store.isDirty(sid) {
		t.Error("isDirty should be true after a failed persist")
	}
}

func TestForgetPreservesDirtyForRecovery(t *testing.T) {
	// forget (clear-tracker) must NOT drop the dirty flag: a session left dirty still
	// needs to emit persistence-warning{dirty:false} on the next successful persist.
	store := newTestServer(t).sessions
	sid := int64(42)
	store.mu.Lock()
	store.dirty[sid] = true // simulate a prior failed persist (banner shown)
	store.mu.Unlock()

	store.forget(sid)

	if !store.isDirty(sid) {
		t.Error("forget cleared the dirty flag — the dirty→healthy recovery broadcast would be lost")
	}
}

func TestStoreLiveWriteThrough(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacter(t, s, gm, "A", 20, 30, 5, 10) // DB starts hp 20/30, mp 5/10
	sid := seedSession(t, s, seedCampaign(t, s, gm))

	seedEntry := func(store *sessionStore) string {
		if _, err := store.load(ctx, sid); err != nil {
			t.Fatalf("load: %v", err)
		}
		hp, hpm, mp, mpm := int64(20), int64(30), int64(5), int64(10)
		e := charEntry("A", 12, charID)
		e.HpCurrent, e.HpMax, e.MpCurrent, e.MpMax = &hp, &hpm, &mp, &mpm
		if _, err := store.addInitiativeEntry(sid, e); err != nil {
			t.Fatalf("add: %v", err)
		}
		return store.getState(sid).Initiative[0].ID
	}

	t.Run("on: mirrors PV to the character row", func(t *testing.T) {
		store := newSessionStore(s.queries, newUUID, true)
		id := seedEntry(store)
		if _, err := store.deltaVitals(sid, id, ptrInt64(-8), nil); err != nil { // 20-8 → 12
			t.Fatalf("delta: %v", err)
		}
		var got int64 = -1
		for i := 0; i < 40; i++ { // write-through is async; poll the DB row
			if row, err := s.queries.GetCharacter(ctx, charID); err == nil {
				got = row.Hpcurrent
				if got == 12 {
					break
				}
			}
			time.Sleep(25 * time.Millisecond)
		}
		if got != 12 {
			t.Errorf("DB hpCurrent=%d, want 12 (written through)", got)
		}
	})
	t.Run("off: leaves the character row untouched", func(t *testing.T) {
		store := newSessionStore(s.queries, newUUID, false)
		id := seedEntry(store)
		_, _ = store.deltaVitals(sid, id, ptrInt64(-5), nil)
		time.Sleep(150 * time.Millisecond) // give any (unwanted) goroutine time to run
		if row, _ := s.queries.GetCharacter(ctx, charID); row.Hpcurrent != 12 {
			t.Errorf("DB hpCurrent=%d, want 12 unchanged (write-through off; 12 from the prior subtest)", row.Hpcurrent)
		}
	})
}

// Concurrent mutations on one session must not race (run with -race) and must all land.
func TestStoreConcurrentMutations(t *testing.T) {
	s := newTestServer(t)
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	store := s.sessions

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = store.addInitiativeEntry(sid, npc("m", n))
		}(i)
	}
	wg.Wait()
	if got := len(store.getState(sid).Initiative); got != 20 {
		t.Errorf("entries=%d, want 20", got)
	}
}
