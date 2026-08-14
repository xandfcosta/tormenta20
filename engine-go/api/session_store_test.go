package api

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

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

// O PV do rastreador É o PV da ficha (ALE-122). O mestre batia -5 na iniciativa
// e a mesma tela mostrava 52/95 ali e 57/95 no card do grupo, porque o socket
// escrevia num blob e só a ficha era lida. Substitui os testes da flag
// `WS_VITALS_WRITETHROUGH_LIVE`, que protegiam um espelho opcional — e que
// estava desligado em produção.
func TestTrackerVitalsAreTheCharactersVitals(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacter(t, s, gm, "A", 20, 30, 5, 10) // hp 20/30, mp 5/10
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.load(ctx, sid); err != nil {
		t.Fatalf("load: %v", err)
	}
	hp, hpm, mp, mpm := int64(20), int64(30), int64(5), int64(10)
	e := charEntry("A", 12, charID)
	e.HpCurrent, e.HpMax, e.MpCurrent, e.MpMax = &hp, &hpm, &mp, &mpm
	if _, err := store.addInitiativeEntry(sid, e); err != nil {
		t.Fatalf("add: %v", err)
	}
	entryID := store.getState(sid).Initiative[0].ID

	snap, err := store.deltaVitals(sid, entryID, ptrInt64(-8), ptrInt64(-2))
	if err != nil {
		t.Fatalf("delta: %v", err)
	}

	// Sem espera: a gravação é o caminho, não um espelho assíncrono.
	row, err := s.queries.GetCharacter(ctx, charID)
	if err != nil {
		t.Fatalf("carregar personagem: %v", err)
	}
	if row.Hpcurrent != 12 || row.Mpcurrent != 3 {
		t.Errorf("ficha = %d/%d PV-PM, esperado 12/3", row.Hpcurrent, row.Mpcurrent)
	}
	// E a entrada espelha o que foi gravado — os dois números da tela são um só.
	got := snap.Initiative[0]
	if derefOr(got.HpCurrent, -1) != 12 || derefOr(got.MpCurrent, -1) != 3 {
		t.Errorf("entrada = %d/%d, esperado espelhar a ficha (12/3)",
			derefOr(got.HpCurrent, -1), derefOr(got.MpCurrent, -1))
	}
}

// A pancada da sessão drena PV TEMPORÁRIOS antes dos reais, como a da ficha
// sempre fez — antes o socket cobrava direto dos PV reais de quem estava sob
// Armadura Arcana.
func TestTrackerDamageDrainsTemporaryPoolsFirst(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacter(t, s, gm, "A", 20, 30, 5, 10)
	seedTempHpPool(t, s, charID, 5)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.load(ctx, sid); err != nil {
		t.Fatalf("load: %v", err)
	}
	hp, hpm := int64(20), int64(30)
	e := charEntry("A", 12, charID)
	e.HpCurrent, e.HpMax = &hp, &hpm
	if _, err := store.addInitiativeEntry(sid, e); err != nil {
		t.Fatalf("add: %v", err)
	}
	entryID := store.getState(sid).Initiative[0].ID

	if _, err := store.deltaVitals(sid, entryID, ptrInt64(-8), nil); err != nil {
		t.Fatalf("delta: %v", err)
	}

	// 5 absorvidos pelo pool, 3 nos PV reais.
	row, _ := s.queries.GetCharacter(ctx, charID)
	if row.Hpcurrent != 17 {
		t.Errorf("PV = %d, esperado 17 (o pool de 5 absorveu antes)", row.Hpcurrent)
	}
	rows, _ := s.queries.ListActiveEffectsByCharacter(ctx, charID)
	if len(parseTempHpPools(rows)) != 0 {
		t.Errorf("o pool tinha de ter sido gasto, sobrou %+v", parseTempHpPools(rows))
	}
}

// seedTempHpPool cria um pool de PV temporários como o que uma magia deixa.
func seedTempHpPool(t *testing.T, s *Server, charID int64, amount int) {
	t.Helper()
	mods := fmt.Sprintf(`[{"target":{"k":"tempHp"},"amount":%d,"bonusType":"untyped"}]`, amount)
	if _, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: charID, Catalogid: "armadura-arcana", Scope: "scene",
		Modifiers: mods, Createdat: nowISO(),
	}); err != nil {
		t.Fatalf("semear pool temporário: %v", err)
	}
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
