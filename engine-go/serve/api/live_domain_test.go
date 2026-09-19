package api

import (
	"context"
	"database/sql"
	"errors"
	"t20engine/app"
	"t20engine/app/initiative"
	"t20engine/infra/config"
	"t20engine/infra/db/dbvalue"
	"testing"

	"t20engine/infra/db"
	"t20engine/infra/db/sqlcgen"
)

// newTestServer sobe o servidor de verdade sobre um SQLite descartável já
// migrado, com catálogo nulo: os helpers de domínio daqui (autorização e
// resolução de combatente) não tocam o motor. O `adminEmails` é variádico para
// as dezenas de chamadores que não se importam com o papel seguirem iguais.
func newTestServer(t *testing.T, adminEmails ...string) *Server {
	t.Helper()
	// Copiado do molde já migrado, e não migrado do zero (ver `db/testdb`): são
	// ~3.400 migrações a menos na suíte.
	path := bancoDeTeste(t)
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("Open test db: %v", err)
	}
	// O fecho do banco é registrado DEPOIS do servidor existir, mais abaixo: ele
	// precisa esperar o trabalho de segundo plano antes de fechar.
	// `synchronous=OFF` só no TESTE: o que sobra depois do molde são os `fsync`
	// das escritas dos próprios casos, um por transação. Durabilidade é o que um
	// banco de teste não tem o que proteger. Fica AQUI e não no `db.Open` porque
	// em produção essa linha seria perda de dados do mestre.
	if _, err := database.Exec("PRAGMA synchronous=OFF"); err != nil {
		t.Fatalf("PRAGMA synchronous=OFF: %v", err)
	}
	// O `DatabasePath` carrega o arquivo que foi REALMENTE aberto: o
	// `/admin/status` o reporta, e um caminho que não é o em uso mandaria o dono
	// olhar o arquivo errado.
	cfg := config.Config{
		JWTSecret: "test-secret", CookieName: "t20_session",
		AdminEmails: adminEmails, DatabasePath: path,
	}
	srv := NewServer(cfg, database, nil)
	// ESPERAR ANTES DE FECHAR. A persistência do estado da sessão roda em
	// goroutine (`table_live_publish.go`), e fechar o banco debaixo dela produz
	// dois sintomas que não se parecem com a causa: um `Persist failed (sql:
	// database is closed)` no log, e — pior — um `TempDir RemoveAll cleanup:
	// directory not empty`, porque o SQLite recria `-wal`/`-shm` depois do
	// `RemoveAll`. O teste falha falando de LIMPEZA, e o caso que estourou não
	// tem nada a ver com o que ele mede.
	//
	// Só aparece sob CPU escassa — verde em 8 núcleos, vermelho em 2 —, e no caso
	// que derruba uma tabela de propósito, porque ele GARANTE a falha de
	// persistência que abre a janela.
	t.Cleanup(func() {
		srv.WaitForBackground()
		_ = database.Close()
	})
	return srv
}

func seedUser(t *testing.T, s *Server, email string) int64 {
	t.Helper()
	u, err := s.queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		Email: email, Passwordhash: "x", Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed user %q: %v", email, err)
	}
	return u.ID
}

func seedCampaign(t *testing.T, s *Server, ownerID int64) int64 {
	t.Helper()
	c, err := s.queries.CreateCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: ownerID, Name: "Mesa", Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	return c.ID
}

// seedCharacter insere o personagem válido mínimo (colunas JSON no padrão) com o
// dono e os vitais dados, e devolve o id dele.
func seedCharacter(t *testing.T, s *Server, ownerID int64, name string, hpCur, hpMax, mpCur, mpMax int64) int64 {
	t.Helper()
	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: "Soldado", Level: 1,
		HpMax: hpMax, HpCurrent: hpCur, MpMax: mpMax, MpCurrent: mpCur,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
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
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed character %q: %v", name, err)
	}
	return id
}

// seedMember senta um personagem à mesa.
//
// Ele NÃO recebe papel, e a ausência é deliberada: a coluna `role` foi apagada
// porque a produção só escrevia `'player'`, e todo caso que semeava `"gm"` media
// um estado que só a bancada sabia produzir. Quem mestra é o DONO da campanha, e
// o jeito de dizer isso a um teste é semear o personagem com o dono certo.
func seedMember(t *testing.T, s *Server, campaignID, characterID int64) {
	t.Helper()
	if _, err := s.queries.CreateMember(context.Background(), sqlcgen.CreateMemberParams{
		Campaignid: campaignID, Characterid: characterID, Addedat: dbvalue.NowISO(),
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
		// O administrador entra em qualquer mesa como mestre: é o que o deixa
		// participar de uma sessão ao vivo.
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

	roster := s.initiativeQueue().Roster()

	t.Run("o dono resolve, com os vitais", func(t *testing.T) {
		got, err := roster.Combatant(ctx, app.Caller{ID: player}, campaignID, pc)
		if err != nil {
			t.Fatalf("o dono foi barrado: %v", err)
		}
		want := initiative.Combatant{CharacterID: pc, Name: "Herói", HpCurrent: 7, HpMax: 12, MpCurrent: 3, MpMax: 8}
		if got != want {
			t.Errorf("veio %+v, queria %+v", got, want)
		}
	})
	t.Run("o mestre resolve o personagem de outro jogador", func(t *testing.T) {
		if _, err := roster.Combatant(ctx, app.Caller{ID: gm}, campaignID, pc); err != nil {
			t.Errorf("o mestre foi barrado: %v", err)
		}
	})
	// As TRÊS recusas são distintas de propósito, e o transporte as traduz em
	// números diferentes: quem não pertence à mesa não pode descobrir, pela
	// diferença entre elas, quais personagens existem nela.
	t.Run("estranho é recusado por não ser dele", func(t *testing.T) {
		_, err := roster.Combatant(ctx, app.Caller{ID: stranger}, campaignID, pc)
		if !errors.Is(err, app.ErrForbidden) {
			t.Errorf("a recusa foi %v, e queria ErrForbidden", err)
		}
	})
	t.Run("personagem que não é membro é recusado pela REGRA", func(t *testing.T) {
		_, err := roster.Combatant(ctx, app.Caller{ID: player}, campaignID, loose)
		if !errors.Is(err, app.ErrRefused) {
			t.Errorf("a recusa foi %v, e queria ErrRefused", err)
		}
	})
	t.Run("personagem que não existe", func(t *testing.T) {
		_, err := roster.Combatant(ctx, app.Caller{ID: gm}, campaignID, 999999)
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("a recusa foi %v, e queria ErrNotFound", err)
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
		Createdat: dbvalue.NowISO(), Updatedat: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}

	trava := s.sessionLifecycle().Access()

	t.Run("o mestre recebe a sessão e o papel", func(t *testing.T) {
		got, papel, err := trava.Session(ctx, app.Caller{ID: gm}, campaignID, sess.ID)
		if err != nil || papel != app.RoleGM || got.ID != sess.ID {
			t.Errorf("papel=%q id=%d err=%v", papel, got.ID, err)
		}
	})
	// A ORDEM importa: o estranho é barrado ANTES de a sessão ser lida. Sem
	// isso, a diferença entre 403 e 404 contaria a quem não pertence à campanha
	// quais sessões existem nela.
	t.Run("estranho é barrado antes de a sessão ser lida", func(t *testing.T) {
		_, _, err := trava.Session(ctx, app.Caller{ID: stranger}, campaignID, sess.ID)
		if !errors.Is(err, app.ErrForbidden) {
			t.Errorf("a recusa foi %v, e queria ErrForbidden", err)
		}
	})
	t.Run("sessão que não existe", func(t *testing.T) {
		_, _, err := trava.Session(ctx, app.Caller{ID: gm}, campaignID, 999999)
		if !errors.Is(err, app.ErrNotFound) {
			t.Errorf("a recusa foi %v, e queria ErrNotFound", err)
		}
	})
}

func seedEffect(t *testing.T, s *Server, charID int64, catalogID, scope string) {
	t.Helper()
	if _, err := s.queries.CreateActiveEffect(context.Background(), sqlcgen.CreateActiveEffectParams{
		Characterid: charID, Catalogid: catalogID, Scope: scope, Modifiers: "[]", Createdat: dbvalue.NowISO(),
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
