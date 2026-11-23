package accounts

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"t20engine/infra/config"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/db/testdb"
)

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

// O CONVITE REVOGADO NO MEIO DO CADASTRO BARRA A CONTA. A checagem do convite vem antes do bcrypt, que leva tempo,
// e o mestre pode gerar outro link nesse intervalo — o que derruba o que o jogador tem na mão. Por isso a
// transação que cria a conta confere o convite de novo (ver `docs/infra.md`, "Refazer a checagem DENTRO da
// transação").
func TestAnInviteRevokedDuringTheSignUpBarsTheAccount(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(testdb.Fresh(t))
	if err != nil {
		t.Fatalf("abrir o banco: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	q := sqlcgen.New(database)
	gate := NewGate(database, q, config.Config{})

	now := dbvalue.NowISO()
	master, err := q.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email: "mestre@t20.local", Passwordhash: "x", Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("criar o mestre: %v", err)
	}
	link, err := gate.RotateInvite(ctx, master.ID)
	if err != nil {
		t.Fatalf("gerar o convite: %v", err)
	}
	// O link estava valendo quando o cadastro o conferiu…
	checked, ok := gate.usableInvite(ctx, link.Token)
	if !ok {
		t.Fatalf("o controle já estava errado: o convite recém-gerado não abre conta")
	}
	// …e o mestre gerou outro antes de a conta ser gravada.
	if _, err := gate.RotateInvite(ctx, master.ID); err != nil {
		t.Fatalf("gerar o segundo convite: %v", err)
	}

	_, err = gate.createUser(ctx, sqlcgen.CreateUserParams{
		Email: "jogador@t20.local", Passwordhash: "x", Createdat: now, Updatedat: now,
	}, &checked)
	if !errors.Is(err, ErrBadInvite) {
		t.Errorf("a conta pelo convite revogado deu %v; want ErrBadInvite", err)
	}
	if _, err := q.GetUserByEmail(ctx, "jogador@t20.local"); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("a conta pelo convite revogado foi gravada (erro da leitura: %v)", err)
	}
}
