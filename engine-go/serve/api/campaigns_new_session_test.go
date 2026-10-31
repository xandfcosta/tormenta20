package api

import (
	"context"
	"net/http"
	"strconv"
	"testing"
)

/*
O NOVA SESSÃO da aba Sessões (ALE-526). Antes dele, só a API JSON criava
sessão, e a crônica não tinha como chegar a uma mesa que ainda não existia.
*/

func newSessionUrl(campaign int64) string {
	return "/campanhas/" + strconv.FormatInt(campaign, 10) + "/nova-sessao"
}

// A sessão nasce com o PRÓXIMO número, planejada, e o mestre cai na mesa dela —
// é lá que mora o "Iniciar sessão".
func TestNewSessionTakesTheNextNumberAndLandsOnItsTable(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	seedSession(t, s, campaign) // a sessão 1

	rec := pedeNaCronica(t, s, owner, http.MethodPost, newSessionUrl(campaign), "title=+A+ca%C3%A7ada+ao+Bicho-pap%C3%A3o+")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("o Nova sessão respondeu %d, e o esperado era 303 para a mesa nova", rec.Code)
	}
	sessions, err := s.queries.ListSessions(context.Background(), campaign)
	if err != nil {
		t.Fatalf("listar as sessões: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("a campanha tem %d sessões depois do Nova sessão; o esperado era 2", len(sessions))
	}
	created := sessions[1]
	if created.Sessionnumber != 2 {
		t.Errorf("a sessão nova é a de número %d; depois da 1 vem a 2", created.Sessionnumber)
	}
	// O nome digitado no diálogo vai junto, sem os espaços das pontas.
	if created.Title.String != "A caçada ao Bicho-papão" {
		t.Errorf("a sessão nova se chama %q; o mestre digitou \"A caçada ao Bicho-papão\"", created.Title.String)
	}
	if created.Status != "planned" {
		t.Errorf("a sessão nova nasceu %q; ela nasce planejada e o mestre a inicia na mesa", created.Status)
	}
	want := "/campanhas/" + strconv.FormatInt(campaign, 10) + "/sessoes/" + strconv.FormatInt(created.ID, 10)
	if to := rec.Header().Get("Location"); to != want {
		t.Errorf("o Nova sessão levou a %q; o esperado era a mesa da sessão nova, %q", to, want)
	}
}

// Sem nome, a sessão nasce sem título — a tela a chama de "Sessão N" — e não
// com uma string vazia, que é outra grafia de "sem título".
func TestANewSessionWithoutANameHasNoTitle(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")

	rec := pedeNaCronica(t, s, owner, http.MethodPost, newSessionUrl(campaign), "title=+++")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("o Nova sessão sem nome respondeu %d; o nome é opcional", rec.Code)
	}
	sessions, err := s.queries.ListSessions(context.Background(), campaign)
	if err != nil || len(sessions) != 1 {
		t.Fatalf("esperava uma sessão, veio %d (%v)", len(sessions), err)
	}
	if sessions[0].Title.Valid {
		t.Errorf("a sessão sem nome foi gravada com título %q; ela nasce NULL", sessions[0].Title.String)
	}
}

// Quem não é dono não cria sessão — a trava é a do caso de uso, e a aba nem
// oferece o botão ao jogador, mas isso é UX.
func TestThePlayerCannotCreateASession(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	hero := seedCharacterAtLevel(t, s, player, "Guerreiro", "Guerreiro", 1, 0, 5)
	seedMember(t, s, campaign, hero)

	rec := pedeNaCronica(t, s, player, http.MethodPost, newSessionUrl(campaign), "x=1")

	if rec.Code < 400 {
		t.Errorf("o jogador criou uma sessão (%d); só o mestre cria", rec.Code)
	}
	sessions, err := s.queries.ListSessions(context.Background(), campaign)
	if err != nil {
		t.Fatalf("listar as sessões: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("a campanha ganhou %d sessões pelo pedido do jogador", len(sessions))
	}
}
