package api

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"testing"
)

func seedSession(t *testing.T, s *Server, campaignID int64) int64 {
	t.Helper()
	sess, err := s.queries.CreateSession(context.Background(), sqlcgen.CreateSessionParams{
		Campaignid: campaignID, Sessionnumber: 1, Title: sql.NullString{String: "S", Valid: true},
		Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
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

	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("initial Load: %v", err)
	}
	// A cena precisa estar iniciada para o turno andar.
	if _, err := store.StartScene(sid); err != nil {
		t.Fatalf("live.StartScene: %v", err)
	}
	if _, err := store.AddInitiativeEntry(sid, npc("Goblin", 15)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := store.NextTurn(sid); err != nil {
		t.Fatalf("NextTurn: %v", err)
	}
	if Dirty, _ := store.Persist(ctx, sid); Dirty {
		t.Fatalf("Persist should succeed, got Dirty")
	}

	store.Forget(sid) // drop the cache → next Load re-hydrates from the DB
	loaded, err := store.Load(ctx, sid)
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
		RuntimeState: blob, UpdatedAt: dbvalue.NowISO(), ID: sid,
	}); err != nil {
		t.Fatalf("seed blob: %v", err)
	}
	loaded, err := s.sessions.Load(ctx, sid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Initiative) != 1 || loaded.Initiative[0].Label != "Boss" || loaded.Round != 2 {
		t.Errorf("hydrated %+v, want Boss/round 2", loaded)
	}
	// O blob é ANTIGO e não traz `sceneActive`; o zero de um bool é
	// `false`, e sem esta dedução a mesa que parou na rodada 2 reabriria fora de
	// cena e a fila sumiria dos jogadores até o mestre clicar em iniciar. O turno
	// em curso é prova de que a cena estava ligada.
	if !loaded.SceneActive {
		t.Error("sessão reaberta no meio do turno voltou fora de cena — a mesa perde a fila")
	}
}

// A recíproca da dedução acima, e é ela que impede o remendo de virar mentira:
// blob antigo SEM turno em curso não inventa cena nenhuma. Uma sessão que
// terminou o combate na semana passada reabre fora de cena, que é o certo.
func TestABlobWithoutATurnInventsNoScene(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	blob := `{"initiative":[{"id":"x","label":"Boss","initiative":9,"type":"npc"}],"round":0,"turnIndex":-1}`
	if err := s.queries.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: blob, UpdatedAt: dbvalue.NowISO(), ID: sid,
	}); err != nil {
		t.Fatalf("seed blob: %v", err)
	}
	loaded, err := s.sessions.Load(ctx, sid)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SceneActive {
		t.Error("blob sem turno acordou em cena — a fila iria para a mesa sem o mestre mandar")
	}
}

func TestStoreRefreshCharacterMaxes(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 3, 0)
	pocoReal := bookPools(t, s, "Guerreiro", 1)
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// A entrada carrega máximos VELHOS e um atual vivo que o refresh não pode
	// tocar.
	e := sheetCombatant("A", 12, charID)
	stale, cur := int64(1), int64(4)
	e.HpMax, e.HpCurrent = &stale, &cur
	if _, err := store.AddInitiativeEntry(sid, e); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got := store.RefreshCharacterMaxes(ctx, sid)
	entry := got.Initiative[0]
	if entry.HpMax == nil || *entry.HpMax != pocoReal.PvMax || entry.MpMax == nil || *entry.MpMax != pocoReal.PmMax {
		t.Errorf("os máximos não foram refrescados: hpMax=%v mpMax=%v, queria os da ficha (%d/%d)",
			entry.HpMax, entry.MpMax, pocoReal.PvMax, pocoReal.PmMax)
	}
	if entry.HpCurrent == nil || *entry.HpCurrent != 4 {
		t.Errorf("hpCurrent should be untouched at 4, got %v", entry.HpCurrent)
	}
}

// O caso que faltava: o máximo ENCOLHE (o mestre baixou o nível, a CON caiu) e o
// atual fica acima dele. Sem aparar, a barra do rastreador mostra 9/5 — mais de
// 100% — enquanto o servidor recusa esse mesmo par em qualquer outro caminho
// (criação e PATCH de vitais). O número que sobra não é "vida a mais": é uma
// ficha que se contradiz na tela.
func TestStoreRefreshClampsCurrentToNewMax(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	// O poço REAL é o do guerreiro de nível 1, e a entrada carrega um máximo
	// maior com um atual acima dele — o estado que o refresh existe para aparar.
	charID := seedCharacter(t, s, gm, "Encolheu")
	pocoReal := bookPools(t, s, "Guerreiro", 1)
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// A entrada carrega máximos ANTIGOS (maiores) e um atual acima do novo teto.
	e := sheetCombatant("Encolheu", 12, charID)
	velhoHpMax, atualHp := pocoReal.PvMax*2, pocoReal.PvMax+4
	velhoMpMax, atualMp := pocoReal.PmMax*2, pocoReal.PmMax+4
	e.HpMax, e.HpCurrent = &velhoHpMax, &atualHp
	e.MpMax, e.MpCurrent = &velhoMpMax, &atualMp
	if _, err := store.AddInitiativeEntry(sid, e); err != nil {
		t.Fatalf("Add: %v", err)
	}

	entry := store.RefreshCharacterMaxes(ctx, sid).Initiative[0]

	if entry.HpCurrent == nil || *entry.HpCurrent != pocoReal.PvMax {
		t.Errorf("PV atual=%v, queria %d (aparado no máximo real)", entry.HpCurrent, pocoReal.PvMax)
	}
	if entry.MpCurrent == nil || *entry.MpCurrent != pocoReal.PmMax {
		t.Errorf("PM atual=%v, queria %d (aparado no máximo real)", entry.MpCurrent, pocoReal.PmMax)
	}
}

func TestStoreDirtyOnPersistFailure(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	store := s.sessions
	if _, err := store.AddInitiativeEntry(sid, npc("x", 1)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if d, _ := store.Persist(ctx, sid); d {
		t.Fatalf("first Persist should succeed")
	}
	_ = s.db.Close() // break the DB so the next write fails
	if d, _ := store.Persist(ctx, sid); !d {
		t.Error("Persist after DB Close should report Dirty")
	}
	if !store.SaveFailed(sid) {
		t.Error("SaveFailed should be true after a failed Persist")
	}
}

func TestForgetPreservesDirtyForRecovery(t *testing.T) {
	// O esquecer NÃO pode largar a flag `Dirty`: uma sessão deixada suja ainda
	// precisa emitir `persistence-warning{Dirty:false}` no próximo `Persist` que
	// der certo.
	store := newTestServer(t).sessions
	sid := int64(42)
	store.Mu.Lock()
	store.Dirty[sid] = true // simulate a prior failed Persist (banner shown)
	store.Mu.Unlock()

	store.Forget(sid)

	if !store.SaveFailed(sid) {
		t.Error("Forget cleared the Dirty flag — the Dirty→healthy recovery broadcast would be lost")
	}
}

// O PV do rastreador É o PV da ficha. Escrever num blob à parte e ler da ficha
// faz a MESMA tela mostrar 52/95 na iniciativa e 57/95 no card do grupo.
func TestTrackerVitalsAreTheCharactersVitals(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 3, 10, 4) // hp 20/30, mp 5/10
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	hp, hpm, mp, mpm := int64(20), int64(30), int64(5), int64(10)
	e := sheetCombatant("A", 12, charID)
	e.HpCurrent, e.HpMax, e.MpCurrent, e.MpMax = &hp, &hpm, &mp, &mpm
	if _, err := store.AddInitiativeEntry(sid, e); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entryID := store.GetState(sid).Initiative[0].ID

	snap, err := store.DeltaVitals(sid, entryID, live.PtrInt64(-8), live.PtrInt64(-2))
	if err != nil {
		t.Fatalf("delta: %v", err)
	}

	// Sem espera: a gravação é o caminho, não um espelho assíncrono.
	if poco := poolsOf(t, s, charID); poco.HpCurrent != 12 || poco.MpCurrent != 3 {
		t.Errorf("ficha = %d/%d PV-PM, esperado 12/3", poco.HpCurrent, poco.MpCurrent)
	}
	// E a entrada espelha o que foi gravado — os dois números da tela são um só.
	got := snap.Initiative[0]
	if live.DerefOr(got.HpCurrent, -1) != 12 || live.DerefOr(got.MpCurrent, -1) != 3 {
		t.Errorf("entrada = %d/%d, esperado espelhar a ficha (12/3)",
			live.DerefOr(got.HpCurrent, -1), live.DerefOr(got.MpCurrent, -1))
	}
}

// A pancada da sessão drena PV TEMPORÁRIOS antes dos reais, como a da ficha
// sempre fez: sem isso, quem está sob Armadura Arcana paga direto nos PV reais.
func TestTrackerDamageDrainsTemporaryPoolsFirst(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 3, 10, 4)
	seedTempHpPool(t, s, charID, 5)
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	hp, hpm := int64(20), int64(30)
	e := sheetCombatant("A", 12, charID)
	e.HpCurrent, e.HpMax = &hp, &hpm
	if _, err := store.AddInitiativeEntry(sid, e); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entryID := store.GetState(sid).Initiative[0].ID

	if _, err := store.DeltaVitals(sid, entryID, live.PtrInt64(-8), nil); err != nil {
		t.Fatalf("delta: %v", err)
	}

	// 5 absorvidos pelo pool, 3 nos PV reais.
	if poco := poolsOf(t, s, charID); poco.HpCurrent != 17 {
		t.Errorf("PV = %d, esperado 17 (o pool de 5 absorveu antes)", poco.HpCurrent)
	}
	rows, _ := s.queries.ListActiveEffectsByCharacter(ctx, charID)
	if len(sheet.ParseTempHpPools(rows)) != 0 {
		t.Errorf("o pool tinha de ter sido gasto, sobrou %+v", sheet.ParseTempHpPools(rows))
	}
}

// seedTempHpPool cria um pool de PV temporários como o que uma magia deixa.
func seedTempHpPool(t *testing.T, s *Server, charID int64, amount int) {
	t.Helper()
	mods := fmt.Sprintf(`[{"target":{"k":"tempHp"},"amount":%d,"bonusType":"untyped"}]`, amount)
	if _, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: charID, Catalogid: "armadura-arcana", Scope: "scene",
		Modifiers: mods, Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("semear pool temporário: %v", err)
	}
}

// Mutações concorrentes na mesma sessão não podem correr entre si (rode com
// `-race`) e todas têm de chegar.
func TestStoreConcurrentMutations(t *testing.T) {
	s := newTestServer(t)
	sid := seedSession(t, s, seedCampaign(t, s, seedUser(t, s, "gm@t.com")))
	store := s.sessions

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = store.AddInitiativeEntry(sid, npc("m", n))
		}(i)
	}
	wg.Wait()
	if got := len(store.GetState(sid).Initiative); got != 20 {
		t.Errorf("entries=%d, want 20", got)
	}
}
