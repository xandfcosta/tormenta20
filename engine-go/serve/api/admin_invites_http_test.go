package api

import (
	"context"
	"errors"
	"strconv"
	"t20engine/domain/account"
	"t20engine/infra/db"
	"t20engine/infra/db/dbvalue"
	"testing"
	"time"

	"t20engine/infra/db/sqlcgen"
)

// O cadastro deixou de ser aberto quando a mesa foi para a rede local.
//
// O que estes casos prendem nunca foi o transporte: é o convite valer UMA vez, a
// corrida de dois cadastros gastá-lo uma só, o vencido ser recusado, e o e-mail
// do dono dispensar convite. Cada um chama a REGRA — a mesma que a PORTA em
// Datastar chama por `CreateAccount`. A AUTORIZAÇÃO continua medida pela ROTA,
// porque ela é da rota.

const adminEmail = "dono@t20.local"

// registra chama a REGRA do cadastro, sem passar por transporte nenhum.
func registra(t *testing.T, s *Server, email, token string) error {
	t.Helper()
	_, err := s.accountRules().createAccount(context.Background(),
		account.RegisterBody{Email: email, Password: "senha-da-mesa", InviteToken: token})
	return err
}

// inviteFrom cunha um convite pela REGRA e devolve o token.
func inviteFrom(t *testing.T, s *Server, adminID int64) string {
	t.Helper()
	convite, err := mintAccountInvite(context.Background(), s.queries, adminID)
	if err != nil {
		t.Fatalf("cunhar convite: %v", err)
	}
	return convite.Token
}

// A autorização de quem CUNHA não é medida aqui de propósito: ela mora no
// endereço que a administração atende de verdade
// (`TestANonAdminDoesNotReachTheInviteRoute` e
// `TestMintingFromAdminPatchesThePanelToo`, em `admin_scene_test.go`). O que
// sobra para este arquivo é o que nunca foi da rota.

// Um convite é um link que anda por fora do app, então adivinhá-lo é entrar na
// mesa. O piso não é estética de URL: é o espaço de busca.
func TestAnInviteTokenIsTooLongToGuess(t *testing.T) {
	s := newTestServer(t, adminEmail)

	if token := inviteFrom(t, s, seedUser(t, s, adminEmail)); len(token) < 16 {
		t.Errorf("token curto demais para ser um link: %q", token)
	}
}

// The door itself: without a link nobody on the network creates an account.
func TestRegisterRequiresAnInvite(t *testing.T) {
	s := newTestServer(t, adminEmail)

	err := registra(t, s, "invasor@t20.local", "")

	if !errors.Is(err, errInviteRejected) {
		t.Fatalf("esperada recusa por convite, veio %v", err)
	}
	if _, err := s.queries.GetUserByEmail(context.Background(), "invasor@t20.local"); err == nil {
		t.Error("a conta não podia ter sido criada")
	}
}

func TestRegisterWithAnInviteWorksExactlyOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))

	first := registra(t, s, "bruna@t20.local", token)
	second := registra(t, s, "caio@t20.local", token)

	if first != nil {
		t.Fatalf("o convite válido tinha de valer, veio %v", first)
	}
	if !errors.Is(second, errInviteRejected) {
		t.Fatalf("esperada recusa reusando o convite, veio %v", second)
	}
	// Rolled back with the invite: a spent link leaves no half-created account.
	if _, err := s.queries.GetUserByEmail(context.Background(), "caio@t20.local"); err == nil {
		t.Error("o segundo registro não podia ter deixado conta para trás")
	}
}

// The sequential reuse above is caught by the READ check alone, so it does not
// prove the `usedAt IS NULL` on the UPDATE. What that clause protects is the
// race: everyone who opens the link at the same instant passes the read, and
// only one UPDATE finds the invite unspent — the losers roll back with it.
func TestConcurrentRegistrationsSpendTheInviteOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))

	const racers = 4
	erros := make(chan error, racers)
	start := make(chan struct{})
	for i := range racers {
		go func() {
			email := "corrida" + strconv.Itoa(i) + "@t20.local"
			<-start
			_, err := s.accountRules().createAccount(context.Background(),
				account.RegisterBody{Email: email, Password: "senha-da-mesa", InviteToken: token})
			erros <- err
		}()
	}
	close(start)

	created := 0
	for range racers {
		if <-erros == nil {
			created++
		}
	}
	if created != 1 {
		t.Errorf("%d cadastros aceitos com UM convite, esperado exatamente 1", created)
	}
	// O admin semeado mais o vencedor da corrida, e mais ninguém.
	if got := countUsers(t, s); got != 2 {
		t.Errorf("o banco ficou com %d contas, esperado 2 (o admin e o vencedor)", got)
	}
}

// A duplicate e-mail must not burn the link — the player still has to Get in.
func TestAFailedRegistrationKeepsTheInviteSpendable(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))
	seedUser(t, s, "bruna@t20.local")

	clash := registra(t, s, "bruna@t20.local", token)
	retry := registra(t, s, "bruna2@t20.local", token)

	if !db.IsUniqueViolation(clash) {
		t.Fatalf("esperada colisão de e-mail repetido, veio %v", clash)
	}
	if retry != nil {
		t.Fatalf("o convite tinha de continuar valendo, veio %v", retry)
	}
}

// NÃO há rota que pergunte se um convite ainda vale, e a ausência é deliberada:
// a porta prefixa o campo com o `?convite=` e deixa o `CreateAccount` recusar
// (`web/door/routes.go`). O prazo continua preso aqui embaixo, do lado que
// decide.
func TestExpiredInviteIsRejected(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	past := time.Now().Add(-time.Minute)
	invite, err := s.queries.CreateAccountInvite(context.Background(), sqlcgen.CreateAccountInviteParams{
		Token: "convite-vencido", Createdby: admin, Createdat: dbvalue.IsoAt(past), Expiresat: dbvalue.IsoAt(past),
	})
	if err != nil {
		t.Fatalf("semear convite: %v", err)
	}

	if err := registra(t, s, "tarde@t20.local", invite.Token); !errors.Is(err, errInviteRejected) {
		t.Errorf("esperada recusa do convite vencido, veio %v", err)
	}
}

// The bootstrap: on a fresh machine the owner has no invite to hold, and
// "first to register wins" would hand the crown to whoever opens the page.
func TestTheAdminEmailRegistersWithoutAnInvite(t *testing.T) {
	s := newTestServer(t, adminEmail)

	if err := registra(t, s, adminEmail, ""); err != nil {
		t.Fatalf("o dono tem de conseguir criar a própria conta, veio %v", err)
	}
}

// The case-insensitive admin check would be a hole if e-mails were not
// normalized: `DONO@` could register WITHOUT an invite as a second account and
// be admin too. Normalization makes it the same row, so it collides instead.
//
// A normalização mora no `createAccount`, e não no manipulador: garantia que
// mora no TRANSPORTE é garantia que o próximo chamador esquece, e é aqui que a
// decisão de "quem é admin" já era tomada.
func TestACaseVariantCannotBecomeASecondAdmin(t *testing.T) {
	s := newTestServer(t, adminEmail)
	if err := registra(t, s, adminEmail, ""); err != nil {
		t.Fatalf("semear o dono: %v", err)
	}

	clash := registra(t, s, "DONO@T20.local", "")

	if !db.IsUniqueViolation(clash) {
		t.Fatalf("esperada colisão para a variante de caixa, veio %v", clash)
	}
	if got := countUsers(t, s); got != 1 {
		t.Errorf("esperado 1 conta, o banco tem %d", got)
	}
}

// NÃO há caso afirmando que o `isAdmin` do `ADMIN_EMAILS` chega em quem
// pergunta, e a ausência é deliberada: a mesma derivação está medida por um
// caminho que a tela usa de verdade — o dono passa pelo `requireAdmin` e o
// jogador leva 403, os dois lendo o flag do mesmo `authUser`.

func countUsers(t *testing.T, s *Server) int {
	t.Helper()
	return countRows(t, s, "SELECT COUNT(*) FROM users")
}

func countOpenInvites(t *testing.T, s *Server) int {
	t.Helper()
	return countRows(t, s, "SELECT COUNT(*) FROM account_invites WHERE usedAt IS NULL")
}

func countRows(t *testing.T, s *Server, query string) int {
	t.Helper()
	var count int
	if err := s.db.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("contar (%s): %v", query, err)
	}
	return count
}
