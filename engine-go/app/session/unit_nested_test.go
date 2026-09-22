package session

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"t20engine/domain/live"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/db/testdb"
	"t20engine/infra/events"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

// nestingEffects é a ficha que, ao ser escrita pelo giro da vez, abre OUTRA
// unidade de trabalho com o contexto que recebeu. É o gesto que chama o gesto —
// a forma que a ALE-374 veio proteger.
type nestingEffects struct {
	units     Units
	sessionID int64
	// nested conta quantas vezes a segunda unidade foi aberta, para o caso não
	// passar verde sobre um caminho que nunca rodou.
	nested int
}

func (n *nestingEffects) SustainedOf(context.Context, int64) ([]live.SustainedEffect, error) {
	return nil, nil
}

func (n *nestingEffects) EndSustained(context.Context, int64, string) error { return nil }

// ExpireTurnEffects ESCREVE por dentro de uma unidade nova. Com a rede armada,
// o `Do` reconhece a unidade que o contexto carrega e reusa a transação; sem
// ela, este `Do` pede a segunda conexão do pool e espera a primeira — que
// espera este retorno. Quem desempata é o `busy_timeout`.
func (n *nestingEffects) ExpireTurnEffects(ctx context.Context, _ int64) error {
	n.nested++
	return n.units.Do(ctx, func(u Unit) error {
		_, err := u.Snapshots.Mutate(ctx, n.sessionID,
			func(*live.SessionRuntimeState) error { return nil })
		return err
	})
}

func (n *nestingEffects) ConditionsOf(context.Context, int64) ([]string, error) { return nil, nil }

func (n *nestingEffects) ConstitutionOf(context.Context, int64) (int, error) { return 0, nil }

func (n *nestingEffects) StabilizeBleeding(context.Context, int64) error { return nil }

// UM GESTO QUE CHAMA OUTRO GESTO TERMINA, EM VEZ DE ESPERAR O `busy_timeout`.
//
// A rede é o `turnGesture` embrulhar o contexto com `WithUnit` antes de descer:
// o `Do` aninhado reconhece a unidade aberta e reusa a transação dela. Sem
// isso, a segunda transação pega outra conexão do pool e pede a trava de
// escrita que a primeira segura — e a primeira só solta quando este trabalho
// retornar. É o impasse consigo mesmo da ALE-371.
//
// ESTE CASO FALHA LENTO DE PROPÓSITO, e o banco é de VERDADE por isso: com a
// rede desarmada ele não devolve um erro qualquer na hora, ele PARA pelos 5s do
// `busy_timeout` e volta com `database is locked (5) (SQLITE_BUSY)`. Um dublê
// de transação não tem como imitar essa espera, que é justamente o sintoma que
// alguém vai ver na mesa.
func TestANestedUnitReusesTheOpenTransaction(t *testing.T) {
	database, queries, sessionID := benchWithASession(t)
	effects := &nestingEffects{sessionID: sessionID}
	units := NewUnits(database, queries,
		func(q *sqlcgen.Queries) (live.SheetVitals, live.SheetTurnEffects) {
			return nil, effects
		})
	effects.units = units

	stored := live.EmptyRuntimeState()
	stored.Initiative = []live.InitiativeEntry{
		{ID: "a", Label: "Arcanista", Type: "character", CharacterID: live.PtrInt64(14)},
		{ID: "b", Label: "Goblin", Type: "npc"},
	}
	live.StartScene(stored, live.SceneAction)
	snapshots := NewSnapshots(queries)
	if _, err := snapshots.Mutate(t.Context(), sessionID,
		func(s *live.SessionRuntimeState) error { *s = *stored; return nil }); err != nil {
		t.Fatalf("arranjar a mesa gravada: %v", err)
	}

	store := NewStore(snapshots, units, func() string { return "id" }, nil, effects, &events.Bus{})

	started := time.Now()
	_, err := store.NextTurn(t.Context(), sessionID)
	elapsed := time.Since(started)

	if err != nil {
		t.Fatalf("o gesto aninhado não terminou depois de %s: %v\n"+
			"a espera é a marca do impasse: o `turnGesture` tem de descer o contexto "+
			"com `WithUnit`, para o `Do` de dentro REUSAR a transação aberta", elapsed, err)
	}
	// O CONTROLE: sem esta contagem, um caminho que nunca chamasse a porta
	// passaria verde sem ter aninhado nada.
	if effects.nested == 0 {
		t.Fatal("o controle falhou: a ficha nunca foi escrita, então nada aninhou")
	}
	if elapsed > time.Second {
		t.Errorf("o gesto levou %s, e o trabalho é de milissegundos — este é o "+
			"tempo de quem esperou a trava, não o de quem reusou a transação", elapsed)
	}
}

// benchWithASession monta um banco de verdade com uma sessão gravável: o
// retrato tem chave estrangeira até o dono, então a linha da mesa precisa da
// campanha, e a campanha precisa da conta.
func benchWithASession(t *testing.T) (*sql.DB, *sqlcgen.Queries, int64) {
	t.Helper()
	database, err := db.Open(testdb.Fresh(t))
	if err != nil {
		t.Fatalf("abrir o banco da bancada: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	queries := sqlcgen.New(database)
	now := dbvalue.NowISO()
	owner, err := queries.CreateUser(t.Context(), sqlcgen.CreateUserParams{
		Email: "mestre@t20.local", Passwordhash: "x", Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("criar o dono da mesa: %v", err)
	}
	campaign, err := queries.CreateCampaign(t.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: owner.ID, Name: "A Tormenta", Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("criar a campanha: %v", err)
	}
	session, err := queries.CreateSession(t.Context(), sqlcgen.CreateSessionParams{
		Campaignid: campaign.ID, Sessionnumber: 1, Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("criar a sessão: %v", err)
	}
	return database, queries, session.ID
}
