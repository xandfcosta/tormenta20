package api

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"t20engine/app/session"
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
	if _, err := store.StartScene(sid, live.SceneAction); err != nil {
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
	// Aqui morava a afirmação de que um blob SEM cena reabre em cena quando há
	// turno em curso — a dedução que existia para os blobs de antes da cena
	// tipada. Ela saiu com a retrocompatibilidade: os dados desta casa são de
	// desenvolvimento, e manter o remendo era carregar uma regra para um blob
	// que ninguém tem (ALE-365).
}

// UM BLOB SEM CENA NÃO INVENTA UMA. A sessão que terminou o combate na semana
// passada reabre fora de cena — e fora de cena a fila não vai para a mesa, que
// é a trava do `RedactForPlayers`. O caso é barato e prende a ponta que
// importa: nada acorda em cena sem o mestre mandar.
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
	if loaded.InScene() {
		t.Error("blob sem turno acordou em cena — a fila iria para a mesa sem o mestre mandar")
	}
}

// O REFRESH TIRA OS DOIS POÇOS DA FICHA — máximo e atual.
//
// # O que ele deixou de fazer
//
// Refrescar só os tetos. O atual era declarado intocável, e o resultado é que a
// fila só andava quando o gesto passava POR ELA: sete sítios mudam o poço de um
// personagem e cinco são da FICHA — os botões ±PV, a dose, a conjuração, a
// postura e o descanso pedido pelo jogador. Nenhum deles chegava à linha, e o
// mestre escolhia alvo por um número de antes (ALE-358).
//
// # A entrada nunca foi a autoridade
//
// Com personagem atrás, o `DeltaVitals` e o `PatchVitals` escrevem na FICHA e a
// linha espelha — está escrito no corpo dos dois. Sobrescrever a linha não
// apaga decisão nenhuma: ela é o espelho.
//
// # Os dois sentidos, e o NPC
//
// O caso percorre stale ALTO e stale BAIXO, porque um refresh que só aparasse
// para baixo passaria no primeiro e deixaria o segundo parado — que é
// exatamente o defeito de onde ele veio. E afirma que o NPC NÃO é tocado: ali
// não há ficha atrás, e o rastreador é o registro.
func TestStoreRefreshTakesBothPoolsFromTheSheet(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	charID := seedCharacterAtLevel(t, s, gm, "A", "Guerreiro", 1, 3, 0)
	naFicha := poolsOf(t, s, charID)
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}

	casos := []struct {
		nome         string
		hpMax, hpCur int64
	}{
		{"stale ALTO (a linha acha que ele tem mais do que tem)",
			naFicha.HpMax * 2, naFicha.HpCurrent + 9},
		{"stale BAIXO (a linha acha que ele apanhou mais do que apanhou)",
			1, 1},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			emptyTheQueue(t, store, sid)
			e := sheetCombatant("A", 12, charID)
			hpMax, hpCur := caso.hpMax, caso.hpCur
			e.HpMax, e.HpCurrent = &hpMax, &hpCur
			if _, err := store.AddInitiativeEntry(sid, e); err != nil {
				t.Fatalf("Add: %v", err)
			}
			// O NPC entra JUNTO: a fronteira só é medida quando os dois estão na
			// mesma fila passando pelo mesmo refresh.
			capanga := live.InitiativeEntry{Label: "Goblin", Initiative: 9, Type: "npc"}
			pvDoCapanga, maxDoCapanga := int64(4), int64(11)
			capanga.HpCurrent, capanga.HpMax = &pvDoCapanga, &maxDoCapanga
			if _, err := store.AddInitiativeEntry(sid, capanga); err != nil {
				t.Fatalf("Add npc: %v", err)
			}

			depois := store.RefreshCharacterVitals(ctx, sid)

			linha := rowLabelled(t, depois, "A")
			if live.DerefOr(linha.HpMax, -1) != naFicha.HpMax ||
				live.DerefOr(linha.HpCurrent, -1) != naFicha.HpCurrent {
				t.Errorf("a linha ficou em %d/%d e a ficha está em %d/%d",
					live.DerefOr(linha.HpCurrent, -1), live.DerefOr(linha.HpMax, -1),
					naFicha.HpCurrent, naFicha.HpMax)
			}
			if live.DerefOr(linha.MpCurrent, -1) != naFicha.MpCurrent {
				t.Errorf("o PM da linha ficou em %d e a ficha está em %d",
					live.DerefOr(linha.MpCurrent, -1), naFicha.MpCurrent)
			}

			doCapanga := rowLabelled(t, depois, "Goblin")
			if live.DerefOr(doCapanga.HpCurrent, -1) != 4 || live.DerefOr(doCapanga.HpMax, -1) != 11 {
				t.Errorf("o NPC saiu em %d/%d e devia estar intocado em 4/11 — "+
					"não há ficha atrás dele, e ali o rastreador É o registro",
					live.DerefOr(doCapanga.HpCurrent, -1), live.DerefOr(doCapanga.HpMax, -1))
			}
		})
	}
}

// emptyTheQueue esvazia a iniciativa entre os casos, para o segundo não medir a
// linha que o primeiro deixou.
func emptyTheQueue(t *testing.T, store *session.Store, sid int64) {
	t.Helper()
	for _, e := range store.GetState(sid).Initiative {
		if _, err := store.RemoveInitiativeEntry(sid, e.ID); err != nil {
			t.Fatalf("limpar a fila: %v", err)
		}
	}
}

// rowLabelled acha a entrada pelo rótulo e FALHA se ela sumiu: entrada ausente e
// entrada intocada se parecem quando a asserção lê um zero.
func rowLabelled(t *testing.T, st *live.SessionRuntimeState, rotulo string) live.InitiativeEntry {
	t.Helper()
	for _, e := range st.Initiative {
		if e.Label == rotulo {
			return e
		}
	}
	t.Fatalf("a linha %q não está na fila", rotulo)
	return live.InitiativeEntry{}
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

// seedSustained liga um efeito que cobra mana por turno, como Velocidade deixa.
func seedSustained(t *testing.T, s *Server, charID int64, spellID string) {
	t.Helper()
	if _, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: charID, Catalogid: spellID, Scope: "sustained",
		Modifiers: "[]", Createdat: dbvalue.NowISO(),
	}); err != nil {
		t.Fatalf("semear sustentada: %v", err)
	}
}

// ENTRAR NA VEZ PAGA 1 PM POR SUSTENTADA (T20 p227), e a que não for paga CAI.
//
// O caso é de INTEGRAÇÃO porque o que ele prende é a COSTURA: a decisão mora no
// `engine.PaySustained`, os efeitos moram na ficha, o mana mora nos dois (ficha
// e fila, espelhados), e quem junta os três é o avanço do turno.
func TestEnteringYourTurnPaysForEachSustainedAbility(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Arcanista", 3, 10, 4)
	seedSustained(t, s, charID, "velocidade")
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := store.StartScene(sid, live.SceneAction); err != nil {
		t.Fatalf("começar a cena: %v", err)
	}
	if _, err := store.AddInitiativeEntry(sid, sheetCombatant("A", 12, charID)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	antes := poolsOf(t, s, charID).MpCurrent

	depois, err := store.NextTurn(sid)
	if err != nil {
		t.Fatalf("entrar na vez: %v", err)
	}

	if agora := poolsOf(t, s, charID).MpCurrent; agora != antes-1 {
		t.Errorf("a sustentada cobra 1 PM da FICHA: era %d e ficou %d", antes, agora)
	}
	extrato := depois.Scene.Upkeep
	if extrato == nil {
		t.Fatal("a faixa não tem o que dizer: a manutenção não deixou extrato")
	}
	if extrato.Cost != 1 || len(extrato.Paid) != 1 || extrato.Paid[0] != "Velocidade" {
		t.Errorf("o extrato diz %+v, quero 1 PM pago por Velocidade", extrato)
	}
	// E a FILA espelha o mana da ficha: os dois números da tela são um só.
	if mp := live.DerefOr(depois.Initiative[0].MpCurrent, -1); mp != antes-1 {
		t.Errorf("a fila mostra %d PM e a ficha tem %d", mp, antes-1)
	}
}

// SEM MANA A SUSTENTADA CAI, e ela some da ficha: "se não o fizer, a habilidade
// termina" (p227). Um efeito de pé devendo PM seria a pior das duas saídas.
func TestWithoutManaTheSustainedAbilityEnds(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	charID := seedCharacterAtLevel(t, s, gm, "A", "Arcanista", 3, 10, 4)
	seedSustained(t, s, charID, "velocidade")
	sid := seedSession(t, s, seedCampaign(t, s, gm))
	store := s.sessions
	if _, err := store.Load(ctx, sid); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := store.StartScene(sid, live.SceneAction); err != nil {
		t.Fatalf("começar a cena: %v", err)
	}
	if _, err := store.AddInitiativeEntry(sid, sheetCombatant("A", 12, charID)); err != nil {
		t.Fatalf("Add: %v", err)
	}
	entryID := store.GetState(sid).Initiative[0].ID
	// Zera o mana pelo caminho de verdade, e o CONTROLE vem junto: sem isto o
	// teste mediria uma ficha cheia e passaria verde sobre nada.
	if _, err := store.DeltaVitals(sid, entryID, nil, live.PtrInt64(-99)); err != nil {
		t.Fatalf("zerar o mana: %v", err)
	}
	if mp := poolsOf(t, s, charID).MpCurrent; mp != 0 {
		t.Fatalf("o controle falhou: a ficha ficou com %d PM em vez de 0", mp)
	}

	depois, err := store.NextTurn(sid)
	if err != nil {
		t.Fatalf("entrar na vez: %v", err)
	}

	extrato := depois.Scene.Upkeep
	if extrato == nil || len(extrato.Dropped) != 1 || extrato.Dropped[0] != "Velocidade" {
		t.Fatalf("o extrato diz %+v, quero Velocidade caída", extrato)
	}
	efeitos, err := s.queries.ListActiveEffectsByCharacter(ctx, charID)
	if err != nil {
		t.Fatalf("listar efeitos: %v", err)
	}
	for _, e := range efeitos {
		if e.Catalogid == "velocidade" {
			t.Error("a sustentada não paga continua na ficha")
		}
	}
}
