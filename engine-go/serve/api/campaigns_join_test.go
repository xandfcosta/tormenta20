package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/serve/web/campaigns"
	"t20engine/serve/web/ui"
	"testing"
)

// Os guardas da CARTA DE CONVITE.
//
// O caminho feliz mora AQUI e não no navegador de propósito: entrar numa mesa
// grava um membro e uma CÓPIA do personagem, e a tela não sabe desfazer isso. Um
// e2e do caminho feliz deixaria lixo permanente no banco de desenvolvimento a
// cada corrida; aqui o banco é descartável.
//
// O que fica no navegador é só o que só ele testemunha: a recusa NATIVA do
// grupo de rádios sem escolha.

func postaCarta(t *testing.T, s *Server, userID int64, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.accountGate().SignSession(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/campanhas/entrar", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

func seedCampanha(t *testing.T, s *Server, owner int64, name, invite string) int64 {
	t.Helper()
	now := dbvalue.NowISO()
	c, err := s.queries.CreateCampaign(context.Background(), sqlcgen.CreateCampaignParams{
		Ownerid: owner, Name: name, Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("seed campanha: %v", err)
	}
	if invite != "" {
		if _, err := s.db.ExecContext(context.Background(),
			"UPDATE campaigns SET inviteToken = ? WHERE id = ?", invite, c.ID); err != nil {
			t.Fatalf("seed convite: %v", err)
		}
	}
	return c.ID
}

// O DONO entra na própria mesa sem convite, e sai daqui para a crônica com 303.
func TestTheOwnerEntersTheirOwnTableWithoutAnInvite(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	hero := seedCharacterAtLevel(t, s, owner, "Thalen", "Guerreiro", 5, -4, 5)

	rec := postaCarta(t, s, owner, url.Values{
		"campaignId":  {strconv.FormatInt(campaign, 10)},
		"characterId": {strconv.FormatInt(hero, 10)},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303\n%s", rec.Code, rec.Body.String())
	}
	// O destino é a CRÔNICA: quem acabou de sentar à mesa cai na página dela.
	if destination := rec.Header().Get("Location"); destination != "/campanhas/"+strconv.FormatInt(campaign, 10) {
		t.Errorf("destino = %q", destination)
	}
}

// Mesa de OUTRA pessoa sem convite é recusada, e a frase diz o que fazer —
// pedir o link — em vez de \"não foi possível entrar\".
func TestWithoutAnInviteSomeoneElsesTableIsRefusedWithTheNextStep(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	visitor := seedUser(t, s, "visitante@t20.local")
	campaign := seedCampanha(t, s, owner, "Mesa fechada", "o-token-certo")
	hero := seedCharacterAtLevel(t, s, visitor, "Yrla", "Arcanista", 4, 4, 4)

	rec := postaCarta(t, s, visitor, url.Values{
		"campaignId":  {strconv.FormatInt(campaign, 10)},
		"characterId": {strconv.FormatInt(hero, 10)},
	})

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, queria 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Peça um link de convite ao mestre") {
		t.Errorf("a recusa não diz o próximo passo:\n%s", rec.Body.String())
	}
}

// COM o convite certo, a mesma pessoa entra. É o par do teste acima: sem ele,
// aquele passaria numa tela que recusasse todo mundo.
func TestWithTheRightInviteTheVisitorEnters(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	visitor := seedUser(t, s, "visitante@t20.local")
	_ = seedCampanha(t, s, owner, "Mesa aberta", "o-token-certo")
	hero := seedCharacterAtLevel(t, s, visitor, "Yrla", "Arcanista", 4, 4, 4)

	rec := postaCarta(t, s, visitor, url.Values{
		"token":       {"o-token-certo"},
		"characterId": {strconv.FormatInt(hero, 10)},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303\n%s", rec.Code, rec.Body.String())
	}
}

// O id vem do CONVITE quando há um, e o campo de número nem aparece. Um número
// digitado junto não pode virar uma segunda fonte para a mesma coisa — senão o
// convite de uma mesa serviria de senha para entrar em OUTRA.
func TestWithAnInviteTheTypedNumberIsIgnored(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	visitor := seedUser(t, s, "visitante@t20.local")
	invited := seedCampanha(t, s, owner, "A que convidou", "o-token-certo")
	other := seedCampanha(t, s, owner, "A que NÃO convidou", "outro-token")
	hero := seedCharacterAtLevel(t, s, visitor, "Yrla", "Arcanista", 4, 4, 4)

	rec := postaCarta(t, s, visitor, url.Values{
		"token":       {"o-token-certo"},
		"campaignId":  {strconv.FormatInt(other, 10)},
		"characterId": {strconv.FormatInt(hero, 10)},
	})

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303\n%s", rec.Code, rec.Body.String())
	}
	if destination := rec.Header().Get("Location"); destination != "/campanhas/"+strconv.FormatInt(invited, 10) {
		t.Errorf("entrou em %q — o número digitado venceu o convite", destination)
	}
}

// A carta resolve o convite NO SERVIDOR: o nome da mesa já vem na primeira
// resposta, e não há estado de \"carregando\" para existir.
func TestTheCardAlreadyCarriesTheTableName(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	visitor := seedUser(t, s, "visitante@t20.local")
	seedCampanha(t, s, owner, "A Queda de Tauron", "o-token-certo")

	v, err := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadJoin(context.Background(), visitor, "o-token-certo")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if !v.ConviteVale || v.NomeDaCampanha != "A Queda de Tauron" {
		t.Errorf("carta = %+v, queria o nome da mesa resolvido", v)
	}
}

// Convite morto é uma RESPOSTA e não um erro de página: a carta diz isso em voz
// alta, para a pessoa pedir outro link em vez de olhar um botão que não envia.
func TestADeadInviteBecomesASentenceAndNotABrokenPage(t *testing.T) {
	s := newTestServer(t)
	visitor := seedUser(t, s, "visitante@t20.local")

	v, err := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadJoin(context.Background(), visitor, "nao-existe")
	if err != nil {
		t.Fatalf("convite morto derrubou a carta: %v", err)
	}
	if v.ConviteVale {
		t.Fatal("convite inexistente foi dado como válido")
	}
	html, err := ui.RenderFragment(t.Context(), campaigns.JoinBody(v))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Convite inválido ou expirado") {
		t.Error("a carta não avisa que o convite morreu")
	}
	if strings.Contains(html, ">Entrar na mesa<") {
		t.Error("a carta ofereceu o botão de entrar com convite morto — porta pintada na parede")
	}
}

func (s *Server) authUserPorID(t *testing.T, id int64) AuthUser {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return s.accountRules().authUser(u)
}

// ehAdmin responde a pergunta que a cena faz por parâmetro, para a bancada não
// ter de montar um `AuthUser` só para ler um booleano.
//
// Ela olha a MESMA configuração que o `currentUser` olha, e não um valor
// inventado: um seed que por acaso caia na lista de administradores tem de
// mudar o que a lista mostra aqui também, senão o guarda mede outra coisa.
func (s *Server) ehAdmin(t *testing.T, id int64) bool {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	return s.accountRules().authUser(u).IsAdmin
}
