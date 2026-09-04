package api

import (
	"context"
	"database/sql"
	"t20engine/plataforma"
	"testing"

	"t20engine/db"
	"t20engine/db/sqlcgen"
)

// newTestServer spins a migrated temp SQLite + a catalog-less Server. The domain
// helpers under test (authz + combatant resolution) never touch the engine, so a nil
// catalog snapshot is fine — this is the seam the WS gateway will reuse.
// newTestServer boots the real server on a throwaway migrated DB. adminEmails
// is variadic so the dozens of callers that don't care about the Role keep
// reading as before (ALE-120).
func newTestServer(t *testing.T, adminEmails ...string) *Server {
	t.Helper()
	// Copiado do molde já migrado, e não migrado do zero: ver o molde em `db/testdb`
	// (ALE-260). São ~3.400 migrações a menos na suíte.
	path := bancoDeTeste(t)
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open test db: %v", err)
	}
	// O fecho do banco é registrado DEPOIS do servidor existir, mais abaixo: ele
	// precisa esperar o trabalho de segundo plano antes de fechar.
	// `synchronous=OFF` só no TESTE, e é o resto do conserto da ALE-260: o que
	// sobrava depois do molde eram os `fsync` das escritas dos próprios testes,
	// um por transação. Durabilidade é o que um banco de teste não tem o que
	// proteger — ele morre no fim do caso, e uma queda de energia no meio da
	// suíte não tem nada a salvar. Fica AQUI e não no `db.Open` porque em
	// produção essa linha seria perda de dados do mestre.
	if _, err := database.Exec("PRAGMA synchronous=OFF"); err != nil {
		t.Fatalf("PRAGMA synchronous=OFF: %v", err)
	}
	// DatabasePath carries the file actually opened, so the config does not lie
	// about it — /admin/status reports it, and reporting a path that is not the
	// one in use would send the owner looking at the wrong file (ALE-120).
	cfg := plataforma.Config{
		JWTSecret: "test-secret", CookieName: "t20_session",
		AdminEmails: adminEmails, DatabasePath: path,
	}
	srv := NewServer(cfg, database, nil)
	// ESPERAR ANTES DE FECHAR. A persistência do estado da sessão roda em
	// goroutine (`live_publish.go`), e fechar o banco debaixo dela produz
	// dois sintomas que não se parecem com a causa: um `Persist failed (sql:
	// database is closed)` no log, e — pior — um `TempDir RemoveAll cleanup:
	// directory not empty`, porque o SQLite recria `-wal`/`-shm` depois do
	// `RemoveAll`. O teste falha falando de LIMPEZA, e o caso que estourou não
	// tem nada a ver com o que ele mede.
	//
	// Só aparece sob CPU escassa: verde em 8 núcleos, vermelho nos 2 vCPUs do
	// CI, no `TestTheRefusedCommandReachesTheGm` — que derruba uma tabela de
	// propósito e por isso GARANTE a falha de persistência que abre a janela.
	t.Cleanup(func() {
		srv.WaitForBackground()
		_ = database.Close()
	})
	return srv
}

func seedUser(t *testing.T, s *Server, email string) int64 {
	t.Helper()
	u, err := s.queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		Email: email, Passwordhash: "x", Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed user %q: %v", email, err)
	}
	return u.ID
}

func seedCampaign(t *testing.T, s *Server, ownerID int64) int64 {
	t.Helper()
	c, err := s.queries.CreateCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: ownerID, Name: "Mesa", Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
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
		CreatedAt: plataforma.NowISO(), UpdatedAt: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed character %q: %v", name, err)
	}
	return id
}

// seedCharacterAtLevel: o nível importa para o descanso (a recuperação é o
// nível × fator), e o `seedCharacter` fixa nível 1.
func seedCharacterAtLevel(
	t *testing.T, s *Server, ownerID int64, name string, level, hpCur, hpMax, mpCur, mpMax int64,
) int64 {
	t.Helper()
	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: "Soldado", Level: level,
		HpMax: hpMax, HpCurrent: hpCur, MpMax: mpMax, MpCurrent: mpCur,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: plataforma.NowISO(), UpdatedAt: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed character %q: %v", name, err)
	}
	return id
}

// seedMember senta um personagem à mesa.
//
// Ele recebia um PAPEL e escrevia na coluna `role`, apagada na ALE-287 — e essa
// assinatura era a origem de dois verdes falsos: a produção sempre escreveu
// `'player'`, então todo caso que semeava `"gm"` media um estado que só a
// bancada sabia produzir. Quem mestra é o DONO da campanha, e o jeito de dizer
// isso a um teste é semear o personagem com o dono certo.
func seedMember(t *testing.T, s *Server, campaignID, characterID int64) {
	t.Helper()
	if _, err := s.queries.CreateMember(context.Background(), sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: characterID, Addedat: plataforma.NowISO(),
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
	seedMember(t, s, campaignID, pc)

	cases := []struct {
		name       string
		caller     AuthUser
		wantRole   string
		wantStatus int
	}{
		{"owner is gm", AuthUser{ID: gm}, "gm", 200},
		{"member is player", AuthUser{ID: player}, "player", 200},
		{"stranger forbidden", AuthUser{ID: stranger}, "", 403},
		// The admin enters any mesa as gm, and this is the rule the WS gateway
		// runs too — it is what lets them Join a live session (ALE-120).
		{"admin is gm anywhere", AuthUser{ID: stranger, IsAdmin: true}, "gm", 200},
	}
	for _, c := range cases {
		Role, status, err := s.campaignRules().resolveRole(ctx, c.caller, campaignID)
		if Role != c.wantRole || status != c.wantStatus {
			t.Errorf("%s: Role=%q status=%d err=%v, want Role=%q status=%d", c.name, Role, status, err, c.wantRole, c.wantStatus)
		}
	}
	if _, status, _ := s.campaignRules().resolveRole(ctx, AuthUser{ID: gm}, 999999); status != 404 {
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
	seedMember(t, s, campaignID, pc)
	loose := seedCharacter(t, s, player, "Solto", 5, 5, 0, 0) // not a member

	t.Run("owner resolves with vitals", func(t *testing.T) {
		got, status, err := s.tableRules().resolveCombatant(ctx, player, campaignID, pc)
		if err != nil || status != 200 {
			t.Fatalf("status=%d err=%v", status, err)
		}
		want := combatant{characterID: pc, name: "Herói", hpCurrent: 7, hpMax: 12, mpCurrent: 3, mpMax: 8}
		if got != want {
			t.Errorf("got %+v, want %+v", got, want)
		}
	})
	t.Run("gm resolves another player's pc", func(t *testing.T) {
		if _, status, err := s.tableRules().resolveCombatant(ctx, gm, campaignID, pc); status != 200 || err != nil {
			t.Errorf("gm should resolve: status=%d err=%v", status, err)
		}
	})
	t.Run("stranger forbidden", func(t *testing.T) {
		if _, status, _ := s.tableRules().resolveCombatant(ctx, stranger, campaignID, pc); status != 403 {
			t.Errorf("status=%d, want 403", status)
		}
	})
	t.Run("non-member character is bad request", func(t *testing.T) {
		if _, status, _ := s.tableRules().resolveCombatant(ctx, player, campaignID, loose); status != 400 {
			t.Errorf("status=%d, want 400", status)
		}
	})
	t.Run("missing character 404", func(t *testing.T) {
		if _, status, _ := s.tableRules().resolveCombatant(ctx, gm, campaignID, 999999); status != 404 {
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
		Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	t.Run("gm gets session + Role", func(t *testing.T) {
		got, Role, status, err := s.campaignRules().sessionForCaller(ctx, AuthUser{ID: gm}, campaignID, sess.ID)
		if err != nil || status != 200 || Role != "gm" || got.ID != sess.ID {
			t.Errorf("status=%d Role=%q id=%d err=%v", status, Role, got.ID, err)
		}
	})
	t.Run("stranger forbidden before session Load", func(t *testing.T) {
		if _, _, status, _ := s.campaignRules().sessionForCaller(ctx, AuthUser{ID: stranger}, campaignID, sess.ID); status != 403 {
			t.Errorf("status=%d, want 403", status)
		}
	})
	t.Run("missing session 404", func(t *testing.T) {
		if _, _, status, _ := s.campaignRules().sessionForCaller(ctx, AuthUser{ID: gm}, campaignID, 999999); status != 404 {
			t.Errorf("status=%d, want 404", status)
		}
	})
}

func seedEffect(t *testing.T, s *Server, charID int64, catalogID, scope string) {
	t.Helper()
	if _, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: charID, Catalogid: catalogID, Scope: scope, Modifiers: "[]", Createdat: plataforma.NowISO(),
	}); err != nil {
		t.Fatalf("seed effect %q/%q: %v", catalogID, scope, err)
	}
}

func effectScopes(t *testing.T, s *Server, charID int64) []string {
	t.Helper()
	rows, err := s.queries.ListActiveEffectsByCharacter(context.Background(), charID)
	if err != nil {
		t.Fatalf("list effects: %v", err)
	}
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = r.Scope
	}
	return out
}

func TestEndSceneEndDay(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gmID := seedUser(t, s, "gm@t.com")
	gm := AuthUser{ID: gmID}
	stranger := AuthUser{ID: seedUser(t, s, "x@t.com")}
	_ = seedCampaign(t, s, gmID)
	char := seedCharacter(t, s, gmID, "PC", 10, 10, 5, 5)

	t.Run("EndScene removes only scene effects", func(t *testing.T) {
		seedEffect(t, s, char, "buff-a", "scene")
		seedEffect(t, s, char, "buff-b", "day")
		if status, err := s.tableRules().EndScene(ctx, gm, char); status != 200 || err != nil {
			t.Fatalf("status=%d err=%v", status, err)
		}
		if got := effectScopes(t, s, char); len(got) != 1 || got[0] != "day" {
			t.Errorf("remaining scopes=%v, want [day]", got)
		}
	})
	t.Run("endDay removes scene and day", func(t *testing.T) {
		seedEffect(t, s, char, "buff-a", "scene") // re-Add the scene one
		if status, err := s.tableRules().endDay(ctx, gm, char); status != 200 || err != nil {
			t.Fatalf("status=%d err=%v", status, err)
		}
		if got := effectScopes(t, s, char); len(got) != 0 {
			t.Errorf("remaining scopes=%v, want []", got)
		}
	})
	t.Run("stranger forbidden, effects untouched", func(t *testing.T) {
		seedEffect(t, s, char, "buff-c", "scene")
		if status, _ := s.tableRules().EndScene(ctx, stranger, char); status != 403 {
			t.Errorf("status=%d, want 403", status)
		}
		if got := effectScopes(t, s, char); len(got) != 1 {
			t.Errorf("effect should survive a rejected rest, scopes=%v", got)
		}
	})
}

func TestRestVitals(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gmID := seedUser(t, s, "gm@t.com")
	gm := AuthUser{ID: gmID}
	_ = seedCampaign(t, s, gmID)

	// Level-1 characters: gain = floor(1 × mult) → ruim 0, normal 1, confortavel 2, luxuosa 3.
	t.Run("luxuosa gains 3, clamped to max", func(t *testing.T) {
		char := seedCharacter(t, s, gmID, "Ferido", 5, 20, 2, 8)
		got, status, err := s.tableRules().restVitals(ctx, gm, char, "luxuosa")
		if err != nil || status != 200 || got.hpCurrent != 8 || got.mpCurrent != 5 {
			t.Fatalf("got %+v status=%d err=%v, want hp=8 mp=5", got, status, err)
		}
	})
	t.Run("gain clamps at max", func(t *testing.T) {
		char := seedCharacter(t, s, gmID, "QuaseCheio", 19, 20, 8, 8)
		got, _, _ := s.tableRules().restVitals(ctx, gm, char, "luxuosa")
		if got.hpCurrent != 20 || got.mpCurrent != 8 {
			t.Errorf("got %+v, want hp=20 mp=8 (clamped)", got)
		}
	})
	t.Run("ruim gains nothing at level 1", func(t *testing.T) {
		char := seedCharacter(t, s, gmID, "Pobre", 5, 20, 2, 8)
		got, _, _ := s.tableRules().restVitals(ctx, gm, char, "ruim")
		if got.hpCurrent != 5 || got.mpCurrent != 2 {
			t.Errorf("got %+v, want unchanged 5/2", got)
		}
	})
	// O EXEMPLO TRABALHADO do livro, p106: "Helior, elfo caçador de 7º nível,
	// recupera 7 PV e 7 PM com uma noite de sono numa estalagem. Mas, como vive
	// com o pé na estrada, dormindo ao relento, se acostumou a recuperar apenas
	// 3 PV e 3 PM." Metade de 7 é 3,5, e o livro diz TRÊS — é o `math.Floor` da
	// implementação, que os casos de nível 1 nunca exercitavam (floor(0,5) = 0
	// não distingue arredondar para baixo de truncar de zerar).
	t.Run("Helior, 7º nível: estalagem devolve 7, relento devolve 3 (p106)", func(t *testing.T) {
		char := seedCharacterAtLevel(t, s, gmID, "Helior", 7, 1, 40, 1, 40)

		normal, _, _ := s.tableRules().restVitals(ctx, gm, char, "normal")
		if normal.hpCurrent != 8 || normal.mpCurrent != 8 {
			t.Errorf("estalagem: %+v, queria 1+7 em PV e PM", normal)
		}

		ferido := seedCharacterAtLevel(t, s, gmID, "Helior ao relento", 7, 1, 40, 1, 40)
		ruim, _, _ := s.tableRules().restVitals(ctx, gm, ferido, "ruim")
		if ruim.hpCurrent != 4 || ruim.mpCurrent != 4 {
			t.Errorf("relento: %+v, queria 1+3 em PV e PM (metade de 7 = 3, não 4)", ruim)
		}
	})

	t.Run("unknown condition falls back to normal (gain 1)", func(t *testing.T) {
		char := seedCharacter(t, s, gmID, "Default", 5, 20, 2, 8)
		got, _, _ := s.tableRules().restVitals(ctx, gm, char, "bogus")
		if got.hpCurrent != 6 || got.mpCurrent != 3 {
			t.Errorf("got %+v, want 6/3", got)
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
	seedMember(t, s, campaignID, pcA)
	seedMember(t, s, campaignID, pcB)
	seedMember(t, s, campaignID, npc)

	// TRÊS e não dois, e o número mudou sem a produção mudar (ALE-287).
	//
	// Aqui se esperava 2, com o NPC de fora, porque o `listPlayerCombatants`
	// filtrava `m.Role != "player"` e a bancada semeava o NPC como `"gm"`. **A
	// produção nunca escreveu `"gm"` nessa coluna** — o único escritor fixava
	// `'player'` —, então o filtro nunca excluiu ninguém e esta função sempre
	// devolveu todos os membros. O 2 era um verde sobre um estado que só a
	// bancada sabia montar.
	players, err := s.tableRules().listPlayerCombatants(ctx, campaignID)
	if err != nil || len(players) != 3 {
		t.Fatalf("players=%d err=%v, want 3", len(players), err)
	}
	if players[0].name != "A" || players[0].hpMax != 10 || players[1].name != "B" {
		t.Errorf("unexpected players: %+v", players)
	}

	ids, err := s.tableRules().listMemberCharacterIds(ctx, campaignID)
	if err != nil || len(ids) != 3 {
		t.Fatalf("ids=%v err=%v, want 3 (players + gm entry)", ids, err)
	}
}
