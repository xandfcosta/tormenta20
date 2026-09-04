package api

import (
	"context"
	"errors"
	"strconv"
	"t20engine/account"
	"t20engine/db"
	"t20engine/plataforma"
	"testing"
	"time"

	"t20engine/db/sqlcgen"
)

// Registration stopped being Open when the table moved to the LAN (ALE-120).
//
// Estes casos batiam em `POST /auth/register` e `POST /admin/invites`, que
// saíram na ALE-277 junto com as outras rotas JSON sem consumidor. O que eles
// prendem nunca foi o transporte: é o convite valer UMA vez, a corrida de dois
// cadastros gastá-lo uma só, o vencido ser recusado, e o e-mail do dono
// dispensar convite. Cada um agora chama a REGRA — a mesma que a PORTA em
// Datastar chama por `CreateAccount`. A AUTORIZAÇÃO continua medida pela rota,
// porque ela é da rota: `POST /admin/convites` é o endereço que a administração
// atende hoje.

const adminEmail = "dono@t20.local"

// registra chama a REGRA do cadastro, sem passar por transporte nenhum.
func registra(t *testing.T, s *Server, email, token string) error {
	t.Helper()
	_, err := s.createAccount(context.Background(),
		account.RegisterBody{Email: email, Password: "senha-da-mesa", InviteToken: token})
	return err
}

// inviteFrom cunha um convite pela REGRA e devolve o token.
func inviteFrom(t *testing.T, s *Server, adminID int64) string {
	t.Helper()
	convite, err := s.mintAccountInvite(context.Background(), adminID)
	if err != nil {
		t.Fatalf("cunhar convite: %v", err)
	}
	return convite.Token
}

// LÁPIDE — `POST /admin/invites` saiu na ALE-277, e com ela o
// `TestOnlyAnAdminIssuesInvites`.
//
// A autorização que ele media já estava medida no endereço que a administração
// atende de verdade: `TestANonAdminDoesNotReachTheInviteRoute` prende o 403 do
// não-admin e o 303 de quem não tem sessão, e
// `TestMintingFromAdminPatchesThePanelToo` prende o dono cunhando — os dois em
// `admin_scene_test.go`, pelo `WebRouter`. O que sobrou aqui é a única parte
// que não era da rota: o tamanho do token.

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
			_, err := s.createAccount(context.Background(),
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

// LÁPIDE — `GET /account-invites/{token}` saiu na ALE-277.
//
// A rota existia para a tela de cadastro da SPA distinguir "peça um convite" de
// "esse link já foi usado" ANTES de a pessoa mandar o formulário. A porta em
// Datastar não pergunta: ela prefixa o campo com o `?convite=` e deixa o
// `CreateAccount` recusar (`web/door/routes.go`). O prazo do convite continua
// preso aqui embaixo, do lado que decide.
func TestExpiredInviteIsRejected(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	past := time.Now().Add(-time.Minute)
	invite, err := s.queries.CreateAccountInvite(context.Background(), sqlcgen.CreateAccountInviteParams{
		Token: "convite-vencido", Createdby: admin, Createdat: plataforma.IsoAt(past), Expiresat: plataforma.IsoAt(past),
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
// Quem normalizava era o MANIPULADOR, e cada chamador repetia a linha. Com a
// rota JSON fora (ALE-277) sobrou um chamador só — e uma garantia que morava no
// transporte é uma garantia que o próximo chamador esquece. Ela desceu para o
// `createAccount`, que é onde a decisão de "quem é admin" já era tomada.
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

// LÁPIDE — `TestMeCarriesTheAdminFlag` saiu com o `GET /auth/me` (ALE-277).
//
// Ele afirmava que o `isAdmin` do `ADMIN_EMAILS` chega em quem pergunta. A
// mesma derivação continua medida, e por um caminho que a tela usa de verdade:
// no `TestOnlyAnAdminIssuesInvites` o dono passa pelo `requireAdmin` e o
// jogador leva 403, e os dois lêem o flag do mesmo `authUser`.

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
