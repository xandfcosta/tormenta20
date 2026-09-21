package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"t20engine/app"
	"t20engine/app/initiative"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
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
	// O CATÁLOGO É O DE VERDADE, e não um `{"items":[]}` — nem nada.
	//
	// Catálogo vazio faz regra sumir do TESTE sem sumir da produção: um escudo
	// passa a ser VESTIDO porque o eixo de equipar não acha o item, e a
	// distribuição de atributo do humano aceita três vezes o mesmo porque a raça
	// não está primada. Fixture que desliga validação em silêncio é pior que
	// fixture lento.
	//
	// Ele subiu do `newSceneFixture` para cá na ALE-355, e a razão é que a
	// bancada passou a poder MENTIR sobre produção: o PV máximo virou derivado do
	// catálogo e o `cmd/api` se recusa a subir sem ele, então um servidor de
	// teste sem catálogo arranja um estado que não existe mais — e o sintoma não
	// é um erro, é uma ficha com os poços em zero.
	srv.primeCatalogs(primedCatalogs(t))
	return srv
}

// primedCatalogs lê e prima o catálogo de verdade, UMA vez por processo.
//
// Em cache porque são 134 servidores de teste neste pacote, e reler e analisar o
// despejo em cada um custava mais que tudo o que eles fazem juntos. O valor é
// só de leitura depois de primado.
var (
	catalogosUmaVez sync.Once
	catalogosPrimos *engine.Catalogs
	catalogosErro   error
)

func primedCatalogs(t *testing.T) *engine.Catalogs {
	t.Helper()
	catalogosUmaVez.Do(func() {
		raw, err := os.ReadFile(filepath.Join("..", "..", "parity", "_catalogs.json"))
		if err != nil {
			catalogosErro = fmt.Errorf("ler catálogos: %w (gere com `go run ./cmd/genoracle`)", err)
			return
		}
		catalogosPrimos, catalogosErro = engine.PrimeEngineCatalogs(raw)
	})
	if catalogosErro != nil {
		t.Fatalf("preparar catálogo: %v", catalogosErro)
	}
	return catalogosPrimos
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

// seedCharacter é o personagem válido mínimo para quem não se importa com o
// poço: um guerreiro de nível 1, inteiro.
//
// Ele recebia os quatro vitais e os escrevia — em trinta chamadas, sem CLASSE
// nenhuma, que é um estado impossível nas regras. O poço
// agora é o do livro, e quem precisa de outro chama o `seedCharacterAtLevel`
// direto dizendo classe, nível e o quanto foi gasto (ALE-355).
func seedCharacter(t *testing.T, s *Server, ownerID int64, name string) int64 {
	t.Helper()
	return seedCharacterAtLevel(t, s, ownerID, name, "Guerreiro", 1, 0, 0)
}

// seedCharacterAtLevel: o nível importa para o descanso (a recuperação é o
// nível × fator), e o `seedCharacter` fixa nível 1.
// seedCharacterAtLevel semeia um personagem com os poços que o LIVRO dá.
//
// # Ele não escolhe mais o máximo, e essa é a mudança
//
// A assinatura recebia os quatro vitais e os escrevia. Isso
// arranjava um estado que a regra não produz — um Arcanista de nível 8 tem 42
// PV pelo livro, e a bancada escrevia 30 —, e enquanto o máximo era coluna
// ninguém notava. Com ele derivado do catálogo (ALE-355), esses números viram
// ficção e todo caso que os afirmava passaria a medir outra coisa.
//
// Agora o ajudante PERGUNTA o poço ao motor, do mesmo jeito que a produção vai
// perguntar, e o chamador diz só quanto o personagem APANHOU. O efeito é que a
// bancada passa a dizer a verdade ANTES de a produção mudar: quando a derivação
// entrar, estes números já serão os dela.
//
// # E todo personagem tem CLASSE
//
// Personagem sem classe é impossível nas regras (decisão do dono), e trinta das
// quarenta chamadas o criavam. Sem classe o poço do livro é ZERO, então o
// estado impossível só era invisível porque ninguém derivava.
func seedCharacterAtLevel(
	t *testing.T, s *Server, ownerID int64, name, class string, level, hpDamage, mpSpent int64,
) int64 {
	t.Helper()
	id, err := s.queries.CreateCharacter(context.Background(), sqlcgen.CreateCharacterParams{
		OwnerId: ownerID, Name: name, Origin: "Soldado", Level: level,
		Size: "Médio", Displacement: 9,
		Proficiencies: "[]", RaceAttributeChoices: "{}", SecondaryRaceChoices: "[]",
		OriginChoices: "[]", ClassPowers: "[]", ClassChoices: "{}", PowerChoices: "{}",
		CreatedAt: dbvalue.NowISO(), UpdatedAt: dbvalue.NowISO(),
	})
	if err != nil {
		t.Fatalf("seed character %q: %v", name, err)
	}
	seedClasse(t, s, id, class, level)
	if hpDamage > 0 || mpSpent > 0 {
		arrangePools(t, s, id, func(p sheet.Pools) (sheet.Pools, error) {
			p.HpCurrent, p.MpCurrent = p.HpMax-hpDamage, p.MpMax-mpSpent
			return p, nil
		})
	}
	return id
}

// poolsOf é o poço DERIVADO de uma ficha — o par que a tela mostra.
//
// A bancada lia `row.Hpcurrent` e as irmãs, e elas saíram do schema na 00015: o
// máximo vem do catálogo e o atual é `máximo − dano`. Perguntar aqui é
// perguntar o mesmo que a cena pergunta (ALE-355).
func poolsOf(t *testing.T, s *Server, id int64) sheet.Pools {
	t.Helper()
	pools, err := sheet.PoolsForCharacters(context.Background(), s.queries, s.catalogs, []int64{id})
	if err != nil {
		t.Fatalf("derivar o poço da ficha %d: %v", id, err)
	}
	pool, found := pools[id]
	if !found {
		t.Fatalf("a ficha %d não existe", id)
	}
	return pool
}

// arrangePools arranja o estado vital de uma ficha PELO FUNIL, que é o único
// caminho que a produção tem.
//
// A bancada não escreve `hpCurrent` por fora nem inventa uma linha de dano: o
// que ela arranja tem de ser um estado que a produção CONSEGUE produzir, senão o
// caso mede um banco impossível. O `TestEveryVitalWriteGoesThroughTheFunnel`
// cobra isso desta bancada com a mesma régua que cobra do resto.
func arrangePools(t *testing.T, s *Server, id int64, rule sheet.PoolRule) sheet.Pools {
	t.Helper()
	row, err := s.queries.GetCharacter(context.Background(), id)
	if err != nil {
		t.Fatalf("ler a ficha %d para arranjar os poços: %v", id, err)
	}
	pools, err := sheet.ApplyToPools(context.Background(), s.queries, s.catalogs, row, rule)
	if err != nil {
		t.Fatalf("arranjar os poços da ficha %d: %v", id, err)
	}
	return pools
}

// bookPools é quanto o LIVRO dá de PV/PM para esta classe neste nível.
//
// Ela existe para a bancada parar de escolher o máximo: o número sai do mesmo
// motor que a ficha usa, então um caso que afirme "metade do PV" continua
// afirmando metade quando a tabela de classe mudar.
type seededPools struct{ PvMax, PmMax int64 }

func bookPools(t *testing.T, s *Server, class string, level int64) seededPools {
	t.Helper()
	pools := s.catalogs.ComputeVitals(engine.VitalContext{
		Level:      int(level),
		Classes:    []engine.ClassEntry{{ClassName: class, Level: int(level)}},
		AttrTotals: map[string]int{},
	})
	if pools.PvMax <= 0 {
		t.Fatalf("a classe %q no nível %d deu %d de PV — ela existe na tabela do livro?",
			class, level, pools.PvMax)
	}
	return seededPools{PvMax: int64(pools.PvMax), PmMax: int64(pools.PmMax)}
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

// O PAPEL NUMA CAMPANHA: o dono mestra, o membro joga, e o resto não entra.
//
// A asserção desceu para a REGRA (`session.Access`) e deixou de passar por uma
// tradução do `serve/api` que deixou de existir (ALE-348): ela era duas linhas
// sobre este mesmo caso de uso, e no fim existia só para este teste. Teste verde
// sobre código que ninguém usa cobra manutenção e não protege nada.
//
// E o que se afirma agora é a RECUSA TIPADA e não o número do HTTP. O número é
// do transporte e tem dono próprio (`statusForAccess`); afirmá-lo aqui faria
// este caso reprovar no dia em que uma tela escolhesse outro código para a
// mesma recusa.
func TestTheRoleInACampaignIsOwnerGmMemberPlayerAndNobodyElse(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	player := seedUser(t, s, "p@t.com")
	stranger := seedUser(t, s, "x@t.com")
	campaignID := seedCampaign(t, s, gm)
	pc := seedCharacter(t, s, player, "PC")
	seedMember(t, s, campaignID, pc)

	cases := []struct {
		name    string
		who     app.Caller
		role    string
		refused error
	}{
		{"o dono mestra", app.Caller{ID: gm}, app.RoleGM, nil},
		{"o membro joga", app.Caller{ID: player}, app.RolePlayer, nil},
		{"o estranho não entra", app.Caller{ID: stranger}, "", app.ErrForbidden},
		// O administrador entra em qualquer mesa como mestre: é o que o deixa
		// participar de uma sessão ao vivo.
		{"o admin mestra em qualquer mesa", app.Caller{ID: stranger, IsAdmin: true}, app.RoleGM, nil},
	}
	for _, c := range cases {
		role, err := s.sessionAccess().RoleInCampaign(ctx, c.who, campaignID)
		if role != c.role {
			t.Errorf("%s: papel=%q, esperado %q (err=%v)", c.name, role, c.role, err)
		}
		if c.refused == nil && err != nil {
			t.Errorf("%s: recusado com %v", c.name, err)
		}
		if c.refused != nil && !errors.Is(err, c.refused) {
			t.Errorf("%s: err=%v, esperado %v", c.name, err, c.refused)
		}
	}
	// A campanha que NÃO EXISTE é uma recusa diferente, e a diferença importa:
	// "não é sua" e "não existe" viram números diferentes no transporte.
	if _, err := s.sessionAccess().RoleInCampaign(ctx, app.Caller{ID: gm}, 999999); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("campanha inexistente devolveu %v, esperado ErrNotFound", err)
	}
}

func TestResolveCombatant(t *testing.T) {
	s := newTestServer(t)
	ctx := context.Background()
	gm := seedUser(t, s, "gm@t.com")
	player := seedUser(t, s, "p@t.com")
	stranger := seedUser(t, s, "x@t.com")
	campaignID := seedCampaign(t, s, gm)
	pc := seedCharacterAtLevel(t, s, player, "Herói", "Bardo", 2, 8, 5)
	seedMember(t, s, campaignID, pc)
	loose := seedCharacter(t, s, player, "Solto") // not a member

	roster := s.initiativeQueue().Roster()

	t.Run("o dono resolve, com os vitais", func(t *testing.T) {
		got, err := roster.Combatant(ctx, app.Caller{ID: player}, campaignID, pc)
		if err != nil {
			t.Fatalf("o dono foi barrado: %v", err)
		}
		// Bardo de nível 2: 15 PV e 8 PM pelo livro, menos os 8 e 5 semeados.
		want := initiative.Combatant{CharacterID: pc, Name: "Herói", HpCurrent: 7, HpMax: 15, MpCurrent: 3, MpMax: 8}
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

	lock := s.sessionLifecycle().Access()

	t.Run("o mestre recebe a sessão e o papel", func(t *testing.T) {
		got, role, err := lock.Session(ctx, app.Caller{ID: gm}, campaignID, sess.ID)
		if err != nil || role != app.RoleGM || got.ID != sess.ID {
			t.Errorf("papel=%q id=%d err=%v", role, got.ID, err)
		}
	})
	// A ORDEM importa: o estranho é barrado ANTES de a sessão ser lida. Sem
	// isso, a diferença entre 403 e 404 contaria a quem não pertence à campanha
	// quais sessões existem nela.
	t.Run("estranho é barrado antes de a sessão ser lida", func(t *testing.T) {
		_, _, err := lock.Session(ctx, app.Caller{ID: stranger}, campaignID, sess.ID)
		if !errors.Is(err, app.ErrForbidden) {
			t.Errorf("a recusa foi %v, e queria ErrForbidden", err)
		}
	})
	t.Run("sessão que não existe", func(t *testing.T) {
		_, _, err := lock.Session(ctx, app.Caller{ID: gm}, campaignID, 999999)
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
	char := seedCharacter(t, s, gmID, "PC")

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
