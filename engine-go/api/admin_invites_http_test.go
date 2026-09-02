package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"t20engine/account"
	"t20engine/plataforma"
	"testing"
	"time"

	"t20engine/db/sqlcgen"
)

// Registration stopped being Open when the table moved to the LAN (ALE-120):
// these go through the REAL router, so the route table, requireAuth,
// requireAdmin and the handler are all in the path — the same reason
// authz_http_test.go exists.

const adminEmail = "dono@t20.local"

// registerBodyJSON is the wire shape a browser would post.
func registerBodyJSON(email, token string) string {
	body, _ := json.Marshal(account.RegisterBody{Email: email, Password: "senha-da-mesa", InviteToken: token})
	return string(body)
}

// inviteFrom issues an invite through the admin route and returns its token.
func inviteFrom(t *testing.T, s *Server, adminID int64) string {
	t.Helper()
	rec := authed(t, s, adminID, http.MethodPost, "/admin/invites", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("criar convite: esperado 201, veio %d (%s)", rec.Code, rec.Body.String())
	}
	token, _ := jsonField(t, rec, "token").(string)
	if token == "" {
		t.Fatalf("convite sem token: %s", rec.Body.String())
	}
	return token
}

func TestOnlyAnAdminIssuesInvites(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")

	t.Run("anônimo", func(t *testing.T) {
		if rec := anon(t, s, http.MethodPost, "/admin/invites"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("esperado 401 sem credencial, veio %d", rec.Code)
		}
	})

	t.Run("jogador autenticado", func(t *testing.T) {
		rec := authed(t, s, player, http.MethodPost, "/admin/invites", "")
		if rec.Code != http.StatusForbidden {
			t.Fatalf("esperado 403 para não-admin, veio %d (%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("admin", func(t *testing.T) {
		if token := inviteFrom(t, s, admin); len(token) < 16 {
			t.Errorf("token curto demais para ser um link: %q", token)
		}
	})
}

// The door itself: without a link nobody on the network creates an account.
func TestRegisterRequiresAnInvite(t *testing.T) {
	s := newTestServer(t, adminEmail)

	rec := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("invasor@t20.local", ""), "")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("esperado 403 sem convite, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if _, err := s.queries.GetUserByEmail(context.Background(), "invasor@t20.local"); err == nil {
		t.Error("a conta não podia ter sido criada")
	}
}

func TestRegisterWithAnInviteWorksExactlyOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	token := inviteFrom(t, s, seedUser(t, s, adminEmail))

	first := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("bruna@t20.local", token), "")
	second := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("caio@t20.local", token), "")

	if first.Code != http.StatusCreated {
		t.Fatalf("esperado 201 com convite válido, veio %d (%s)", first.Code, first.Body.String())
	}
	if second.Code != http.StatusForbidden {
		t.Fatalf("esperado 403 reusando o convite, veio %d (%s)", second.Code, second.Body.String())
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
	codes := make(chan int, racers)
	start := make(chan struct{})
	for i := range racers {
		go func() {
			email := "corrida" + strconv.Itoa(i) + "@t20.local"
			<-start
			codes <- sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON(email, token), "").Code
		}()
	}
	close(start)

	created := 0
	for range racers {
		if <-codes == http.StatusCreated {
			created++
		}
	}
	if created != 1 {
		t.Errorf("%d respostas 201 com UM convite, esperado exatamente 1", created)
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

	clash := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("bruna@t20.local", token), "")
	retry := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("bruna2@t20.local", token), "")

	if clash.Code != http.StatusConflict {
		t.Fatalf("esperado 409 para e-mail repetido, veio %d (%s)", clash.Code, clash.Body.String())
	}
	if retry.Code != http.StatusCreated {
		t.Fatalf("o convite tinha de continuar valendo, veio %d (%s)", retry.Code, retry.Body.String())
	}
}

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

	resolve := anon(t, s, http.MethodGet, "/account-invites/"+invite.Token)
	register := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("tarde@t20.local", invite.Token), "")

	if resolve.Code != http.StatusNotFound {
		t.Errorf("esperado 404 resolvendo convite vencido, veio %d", resolve.Code)
	}
	if register.Code != http.StatusForbidden {
		t.Errorf("esperado 403 registrando com convite vencido, veio %d", register.Code)
	}
}

// The bootstrap: on a fresh machine the owner has no invite to hold, and
// "first to register wins" would hand the crown to whoever opens the page.
func TestTheAdminEmailRegistersWithoutAnInvite(t *testing.T) {
	s := newTestServer(t, adminEmail)

	rec := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON(adminEmail, ""), "")

	if rec.Code != http.StatusCreated {
		t.Fatalf("o dono tem de conseguir criar a própria conta, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if isAdmin, _ := jsonField(t, rec, "isAdmin").(bool); !isAdmin {
		t.Errorf("a conta do ADMIN_EMAILS tinha de vir isAdmin, veio %s", rec.Body.String())
	}
}

// The case-insensitive admin check would be a hole if e-mails were not
// normalized: `DONO@` could register WITHOUT an invite as a second account and
// be admin too. Normalization makes it the same row, so it collides instead.
func TestACaseVariantCannotBecomeASecondAdmin(t *testing.T) {
	s := newTestServer(t, adminEmail)
	sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON(adminEmail, ""), "")

	rec := sendRaw(t, s, http.MethodPost, "/auth/register", registerBodyJSON("DONO@T20.local", ""), "")

	if rec.Code != http.StatusConflict {
		t.Fatalf("esperado 409 para a variante de caixa, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if got := countUsers(t, s); got != 1 {
		t.Errorf("esperado 1 conta, o banco tem %d", got)
	}
}

// The UI reads isAdmin off /auth/me to decide whether the admin door exists.
func TestMeCarriesTheAdminFlag(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")

	for _, tc := range []struct {
		name string
		id   int64
		want bool
	}{
		{name: "admin", id: admin, want: true},
		{name: "jogador", id: player, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := authed(t, s, tc.id, http.MethodGet, "/auth/me", "")
			if got, _ := jsonField(t, rec, "isAdmin").(bool); got != tc.want {
				t.Errorf("isAdmin = %v, esperado %v (%s)", got, tc.want, rec.Body.String())
			}
		})
	}
}

func countUsers(t *testing.T, s *Server) int {
	t.Helper()
	var count int
	if err := s.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		t.Fatalf("contar usuários: %v", err)
	}
	return count
}
