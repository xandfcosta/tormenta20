package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
)

// Quem enxerga quem, pelo router de verdade (ALE-186, bloco 1).
//
// `GET /users` devolve E-MAILS, e a única coisa entre ele e a lista inteira da
// instância é a regra de visibilidade do `handleListUsers` — que não tinha
// teste nenhum. É a família da ALE-120: "registro aberto na LAN" é o tipo de
// coisa que ninguém percebe até ser tarde. O `authz_http_test.go` já prova que
// a rota exige credencial; o que falta é provar que a credencial NÃO abre tudo.

func listedEmails(t *testing.T, rec *httptest.ResponseRecorder) []string {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200 (corpo %q)", rec.Code, rec.Body.String())
	}
	var users []userDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &users); err != nil {
		t.Fatalf("resposta não é uma lista de usuários (%s): %v", rec.Body.String(), err)
	}
	emails := make([]string, 0, len(users))
	for _, u := range users {
		emails = append(emails, u.Email)
	}
	sort.Strings(emails)
	return emails
}

func listUsersAs(t *testing.T, s *Server, userID int64) []string {
	t.Helper()
	return listedEmails(t, authed(t, s, userID, http.MethodGet, "/users", ""))
}

func sameEmails(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// mesaCom monta uma mesa do `gm` com os personagens dos `players` dentro, que é
// a única coisa que cria visibilidade entre duas contas.
func mesaCom(t *testing.T, s *Server, gm int64, players ...int64) int64 {
	t.Helper()
	campaign := seedCampaign(t, s, gm)
	for i, p := range players {
		char := seedCharacter(t, s, p, "PC"+id64(int64(i)), 10, 10, 0, 0)
		seedMember(t, s, campaign, char, "player")
	}
	return campaign
}

func TestListUsersHidesWhoeverSharesNoTable(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	seedUser(t, s, "estranho@t20.local") // outra conta da mesma instância
	mesaCom(t, s, gm, player)

	got := listUsersAs(t, s, gm)

	// O estranho NUNCA entra: é a asserção que dá sentido ao endpoint. Sem ela,
	// um refactor devolve o catálogo de e-mails da instância a qualquer
	// autenticado, e a suíte inteira continua verde.
	want := []string{"jogador@t20.local", "mestre@t20.local"}
	if !sameEmails(got, want) {
		t.Fatalf("o mestre vê %v, esperado %v", got, want)
	}
}

func TestListUsersShowsThePlayerTheirGm(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	seedUser(t, s, "estranho@t20.local") // sem ele o teste passa até devolvendo TODOS
	mesaCom(t, s, gm, player)

	got := listUsersAs(t, s, player)

	// A visibilidade vale nos DOIS sentidos, por caminhos de SQL diferentes
	// (`VisibleGmOwners` aqui, `VisiblePlayerOwners` no teste acima): provar só
	// um deixaria metade da regra sem rede.
	want := []string{"jogador@t20.local", "mestre@t20.local"}
	if !sameEmails(got, want) {
		t.Fatalf("o jogador vê %v, esperado %v", got, want)
	}
}

func TestListUsersDoesNotIntroducePlayersToEachOther(t *testing.T) {
	s := newTestServer(t)
	gm := seedUser(t, s, "mestre@t20.local")
	alice := seedUser(t, s, "alice@t20.local")
	bob := seedUser(t, s, "bob@t20.local")
	mesaCom(t, s, gm, alice, bob)

	got := listUsersAs(t, s, alice)

	// FIXAR o que o código FAZ, e não o que o comentário dele dizia: a visão do
	// jogador é a do MESTRE da mesa, não a da mesa inteira. Dois jogadores da
	// mesma crônica não trocam e-mail um com o outro — e quem vier ampliar isso
	// vai ter de apagar esta asserção de propósito, que é o ponto.
	want := []string{"alice@t20.local", "mestre@t20.local"}
	if !sameEmails(got, want) {
		t.Fatalf("Alice vê %v, esperado %v (o Bob joga com ela, mas isso não é visibilidade)", got, want)
	}
}

func TestListUsersAlwaysContainsTheCaller(t *testing.T) {
	s := newTestServer(t)
	sozinho := seedUser(t, s, "sozinho@t20.local")
	seedUser(t, s, "outro@t20.local")

	got := listUsersAs(t, s, sozinho)

	// Quem não tem mesa nenhuma ainda se vê: a lista nunca volta vazia, e o
	// vizinho continua invisível.
	if !sameEmails(got, []string{"sozinho@t20.local"}) {
		t.Fatalf("quem não tem mesa vê %v, esperado só a si mesmo", got)
	}
}
