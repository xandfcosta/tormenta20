package api

import (
	"context"
	"errors"
	"strconv"
	"t20engine/app/accounts"
	"t20engine/domain/account"
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
	_, err := s.accountGate().Register(context.Background(),
		account.RegisterBody{Email: email, Password: "senha-da-mesa", InviteToken: token})
	return err
}

// inviteFrom cunha um convite pela REGRA e devolve o token.
func inviteFrom(t *testing.T, s *Server, adminID int64) string {
	t.Helper()
	invite, err := s.accountGate().RotateInvite(context.Background(), adminID)
	if err != nil {
		t.Fatalf("cunhar convite: %v", err)
	}
	return invite.Token
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

// A porta em si: sem link, ninguém na rede cria conta.
func TestRegisterRequiresAnInvite(t *testing.T) {
	s := newTestServer(t, adminEmail)

	err := registra(t, s, "invasor@t20.local", "")

	if !errors.Is(err, accounts.ErrBadInvite) {
		t.Fatalf("esperada recusa por convite, veio %v", err)
	}
	if _, err := s.queries.GetUserByEmail(context.Background(), "invasor@t20.local"); err == nil {
		t.Error("a conta não podia ter sido criada")
	}
}

// Um link serve para várias contas (decisão do dono): quem administra manda o
// mesmo endereço para a mesa inteira.
func TestOneInviteLinkOpensSeveralAccounts(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))

	for _, email := range []string{"bruna@t20.local", "caio@t20.local"} {
		if err := registra(t, s, email, token); err != nil {
			t.Fatalf("%s não entrou pelo link: %v", email, err)
		}
	}
	if got := countRows(t, s, "SELECT COUNT(*) FROM account_invite_uses"); got != 2 {
		t.Errorf("o registro de quem entrou tem %d linhas, want 2", got)
	}
}

// A mesa inteira abre o link ao mesmo tempo, e ninguém perde a corrida: o que
// esta transação protege é a conta nunca ficar sem o registro de por onde entrou.
func TestEveryoneOpeningTheLinkTogetherGetsAnAccount(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))

	const racers = 4
	errs := make(chan error, racers)
	start := make(chan struct{})
	for i := range racers {
		go func() {
			email := "corrida" + strconv.Itoa(i) + "@t20.local"
			<-start
			_, err := s.accountGate().Register(context.Background(),
				account.RegisterBody{Email: email, Password: "senha-da-mesa", InviteToken: token})
			errs <- err
		}()
	}
	close(start)

	for range racers {
		if err := <-errs; err != nil {
			t.Errorf("um cadastro simultâneo foi recusado: %v", err)
		}
	}
	// O admin semeado mais os quatro.
	if got := countUsers(t, s); got != 5 {
		t.Errorf("o banco ficou com %d contas, esperado 5", got)
	}
	if got := countRows(t, s, "SELECT COUNT(*) FROM account_invite_uses"); got != racers {
		t.Errorf("o registro de quem entrou tem %d linhas, want %d", got, racers)
	}
}

// O e-mail repetido volta atrás inteiro: nem conta, nem registro de entrada.
func TestAFailedRegistrationLeavesNoTraceOnTheLink(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))
	seedUser(t, s, "bruna@t20.local")

	if clash := registra(t, s, "bruna@t20.local", token); !errors.Is(clash, accounts.ErrEmailTaken) {
		t.Fatalf("esperada colisão de e-mail repetido, veio %v", clash)
	}
	if got := countRows(t, s, "SELECT COUNT(*) FROM account_invite_uses"); got != 0 {
		t.Errorf("o cadastro recusado deixou %d linhas no registro de entrada", got)
	}
}

// NÃO há rota que pergunte se um convite ainda vale, e a ausência é deliberada:
// a porta prefixa o campo com o `?convite=` e deixa o `Register` recusar
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

	if err := registra(t, s, "tarde@t20.local", invite.Token); !errors.Is(err, accounts.ErrBadInvite) {
		t.Errorf("esperada recusa do convite vencido, veio %v", err)
	}
}

// A partida do zero: numa máquina nova o dono não tem convite nenhum na mão, e
// "quem se cadastra primeiro ganha" entregaria a coroa a quem abrisse a página.
func TestTheAdminEmailRegistersWithoutAnInvite(t *testing.T) {
	s := newTestServer(t, adminEmail)

	if err := registra(t, s, adminEmail, ""); err != nil {
		t.Fatalf("o dono tem de conseguir criar a própria conta, veio %v", err)
	}
}

// A conferência de admin ignora a caixa, e isso seria um buraco se os e-mails
// não fossem normalizados: `DONO@` se cadastraria SEM convite, como uma segunda
// conta, e seria admin também. A normalização faz dela a mesma linha, então ela
// colide em vez de passar.
//
// A normalização mora no `Register`, e não no manipulador: garantia que mora no
// TRANSPORTE é garantia que o próximo chamador esquece, e é aqui que a decisão
// de "quem é admin" já era tomada.
func TestACaseVariantCannotBecomeASecondAdmin(t *testing.T) {
	s := newTestServer(t, adminEmail)
	if err := registra(t, s, adminEmail, ""); err != nil {
		t.Fatalf("semear o dono: %v", err)
	}

	clash := registra(t, s, "DONO@T20.local", "")

	if !errors.Is(clash, accounts.ErrEmailTaken) {
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
	return countRows(t, s, "SELECT COUNT(*) FROM account_invite_links WHERE revokedAt IS NULL")
}

func countRows(t *testing.T, s *Server, query string) int {
	t.Helper()
	var count int
	if err := s.db.QueryRowContext(context.Background(), query).Scan(&count); err != nil {
		t.Fatalf("contar (%s): %v", query, err)
	}
	return count
}
