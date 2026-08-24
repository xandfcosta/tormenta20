package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// Redefinição de senha (ALE-120). O que precisa ser verdade: o link troca a
// senha DE VERDADE (a antiga para de valer), serve uma vez só, e nada disso
// depende de estar logado — quem esqueceu a senha não consegue autenticar.

// resetLinkFor mints a Reset link for UserID through the admin route.
func resetLinkFor(t *testing.T, s *Server, adminID, UserID int64) string {
	t.Helper()
	path := "/admin/users/" + strconv.FormatInt(UserID, 10) + "/password-Reset"
	rec := authed(t, s, adminID, http.MethodPost, path, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("gerar link: esperado 201, veio %d (%s)", rec.Code, rec.Body.String())
	}
	token, _ := jsonField(t, rec, "token").(string)
	if token == "" {
		t.Fatalf("link sem token: %s", rec.Body.String())
	}
	return token
}

func passwordOf(t *testing.T, s *Server, UserID int64) string {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), UserID)
	if err != nil {
		t.Fatalf("carregar usuário: %v", err)
	}
	return user.Passwordhash
}

func TestTheResetLinkActuallyChangesThePassword(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)
	before := passwordOf(t, s, player)

	rec := sendRaw(t, s, http.MethodPost, "/auth/Reset-password",
		`{"token":"`+token+`","password":"nova-senha-do-jogador"}`, "")

	if rec.Code != http.StatusNoContent {
		t.Fatalf("esperado 204, veio %d (%s)", rec.Code, rec.Body.String())
	}
	after := passwordOf(t, s, player)
	if after == before {
		t.Fatal("o hash não mudou — o link não trocou senha nenhuma")
	}
	if bcrypt.CompareHashAndPassword([]byte(after), []byte("nova-senha-do-jogador")) != nil {
		t.Error("a senha nova não abre a conta")
	}
}

// Um link gasto que continuasse valendo seria uma senha que qualquer um com o
// histórico da conversa pode trocar.
func TestTheResetLinkWorksOnlyOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)
	body := `{"token":"` + token + `","password":"outra-senha-longa"}`

	first := sendRaw(t, s, http.MethodPost, "/auth/Reset-password", body, "")
	second := sendRaw(t, s, http.MethodPost, "/auth/Reset-password", body, "")

	if first.Code != http.StatusNoContent {
		t.Fatalf("primeiro uso: esperado 204, veio %d", first.Code)
	}
	if second.Code != http.StatusForbidden {
		t.Errorf("segundo uso: esperado 403, veio %d (%s)", second.Code, second.Body.String())
	}
}

// O uso sequencial acima é barrado na LEITURA, então ele não prova o
// `usedAt IS NULL` do UPDATE. O que a cláusula protege é a corrida: todos
// passam pela leitura, e só um UPDATE encontra o link por gastar.
func TestConcurrentResetsSpendTheLinkOnce(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)

	const racers = 4
	codes := make(chan int, racers)
	start := make(chan struct{})
	for i := range racers {
		go func() {
			body := `{"token":"` + token + `","password":"senha-do-corredor-` + strconv.Itoa(i) + `"}`
			<-start
			codes <- sendRaw(t, s, http.MethodPost, "/auth/Reset-password", body, "").Code
		}()
	}
	close(start)

	won := 0
	for range racers {
		if <-codes == http.StatusNoContent {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d redefinições aceitas com UM link, esperado exatamente 1", won)
	}
}

// A tela pergunta antes de mostrar o formulário: um link vencido diz isso em
// vez de falhar no envio com a senha já digitada duas vezes.
func TestResolvingAResetLinkNamesTheAccount(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)

	ok := anon(t, s, http.MethodGet, "/password-resets/"+token)
	bogus := anon(t, s, http.MethodGet, "/password-resets/nao-existe")

	if email, _ := jsonField(t, ok, "email").(string); email != "jogador@t20.local" {
		t.Errorf("email = %q, esperado a conta do link (%s)", email, ok.Body.String())
	}
	if bogus.Code != http.StatusNotFound {
		t.Errorf("link inventado: esperado 404, veio %d", bogus.Code)
	}
}

// A senha nova passa pela MESMA regra do registro — a tela que ficasse mais
// frouxa seria justamente a que troca a senha de uma conta que já existe.
func TestAResetRefusesAWeakPassword(t *testing.T) {
	s := newTestServer(t, adminEmail)
	admin := seedUser(t, s, adminEmail)
	player := seedUser(t, s, "jogador@t20.local")
	token := resetLinkFor(t, s, admin, player)
	before := passwordOf(t, s, player)

	rec := sendRaw(t, s, http.MethodPost, "/auth/Reset-password",
		`{"token":"`+token+`","password":"curta"}`, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400, veio %d (%s)", rec.Code, rec.Body.String())
	}
	if passwordOf(t, s, player) != before {
		t.Error("a senha mudou apesar da recusa")
	}
}
