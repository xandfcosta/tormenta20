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
	"testing"
)

// Os guardas da CRÔNICA.

func pedeNaCronica(t *testing.T, s *Server, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.accountGate().SignSession(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// AS SESSÕES RECENTES SÃO AS RECENTES. O `ListSessions` ordena por número
// CRESCENTE, então pegar as três PRIMEIRAS dá as mais ANTIGAS — um defeito que
// não aparece numa mesa com três sessões, só numa que já jogou bastante, e a
// tela não tem como avisar que está mentindo.
func TestSessionsComeFromTheNewestToTheOldest(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "A Queda de Tauron", "")
	for i := 1; i <= 5; i++ {
		seedSessao(t, s, campaign, int64(i))
	}

	v, err := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Sessoes) != 5 {
		t.Fatalf("esperava 5 sessões, veio %d", len(v.Sessoes))
	}
	if v.Sessoes[0].Numero != 5 || v.Sessoes[4].Numero != 1 {
		numbers := make([]int64, len(v.Sessoes))
		for i, sess := range v.Sessoes {
			numbers[i] = sess.Numero
		}
		t.Errorf("ordem = %v, queria da mais nova para a mais velha", numbers)
	}
}

// O MESTRE VEM PRIMEIRO no elenco: é o que faz o grupo se ler como grupo em vez
// de fila.
//
// Quem mestra é o DONO da mesa, e é por isso que os dois personagens precisam de
// donos DIFERENTES: com os dois pertencendo ao mesmo usuário, os dois são do
// mestre, e não haveria fila para ordenar.
//
// A BANCADA é o risco aqui: uma semente que escreva um papel que a PRODUÇÃO
// nunca escreve deixa o caso verde sobre uma ordenação que nunca aconteceu.
func TestTheGmComesFirstInTheCast(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	playerTurn := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampanha(t, s, owner, "Mesa", "")
	player := seedCharacterAtLevel(t, s, playerTurn, "Yrla", "Arcanista", 4, 4, 4)
	gm := seedCharacterAtLevel(t, s, owner, "Thalen", "Guerreiro", 5, -4, 5)
	seedMember(t, s, campaign, player)
	seedMember(t, s, campaign, gm)

	v, err := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Herois) != 2 || !v.Herois[0].EhMestre || v.Herois[0].Nome != "Thalen" {
		t.Errorf("elenco = %+v, queria o mestre primeiro", v.Herois)
	}
	// E o sinete conta JOGADORES, não membros: são duas contagens legítimas, e
	// trocá-las faz a tela dizer "2 heróis" numa mesa de um jogador só.
	if v.TotalHerois != 1 {
		t.Errorf("TotalHerois = %d, queria 1 (o mestre não é herói do grupo)", v.TotalHerois)
	}
}

// `?tab=config` na URL de um JOGADOR cai para a visão geral: a aba não existe
// no trilho dele, e desenhar a seção sem o trilho seria tela pela metade. A
// trava de verdade é das rotas de escrita, que respondem 403.
func TestAPlayerAskingForConfigFallsBackToTheOverview(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	visitor := seedUser(t, s, "visitante@t20.local")
	campaign := seedCampanha(t, s, owner, "Mesa", "")
	hero := seedCharacterAtLevel(t, s, visitor, "Yrla", "Arcanista", 4, 4, 4)
	seedMember(t, s, campaign, hero)

	v, err := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadOne(context.Background(), visitor, s.ehAdmin(t, visitor), campaign, "config")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if v.AbaAtiva() != "visao" {
		t.Errorf("aba = %q, queria cair para visao", v.AbaAtiva())
	}
	if v.EhMestre {
		t.Error("o jogador foi marcado como mestre")
	}
}

// AS TRÊS AÇÕES SÃO DE MESTRE, e a trava é do servidor. A tela não mostra a aba
// para o jogador, mas isso é UX — quem postar na mão leva 403.
func TestTheCampaignActionsBelongToTheGm(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	visitor := seedUser(t, s, "visitante@t20.local")
	campaign := seedCampanha(t, s, owner, "Mesa", "")
	hero := seedCharacterAtLevel(t, s, visitor, "Yrla", "Arcanista", 4, 4, 4)
	seedMember(t, s, campaign, hero)
	base := "/campanhas/" + strconv.FormatInt(campaign, 10)

	for _, path := range []string{base + "/editar", base + "/excluir", base + "/regras/carga"} {
		rec := pedeNaCronica(t, s, visitor, http.MethodPost, path, "name=Roubada")
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s respondeu %d para o jogador, queria 403", path, rec.Code)
		}
	}
	// E a campanha continua intacta depois das três tentativas.
	c, err := s.queries.GetCampaign(context.Background(), campaign)
	if err != nil || c.Name != "Mesa" {
		t.Errorf("a campanha mudou: %+v (%v)", c, err)
	}
}

// A recusa do cadastro DEVOLVE O QUE FOI DIGITADO, e não o que está no banco:
// a pessoa está olhando para o próprio texto, e devolver o antigo apagaria a
// edição dela na cara.
func TestTheSignUpRefusalGivesBackTheTypedText(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "Nome antigo", "")
	const novaDescricao = "A caravana parte de Valkaria ao amanhecer."

	form := url.Values{"name": {"   "}, "description": {novaDescricao}}
	rec := pedeNaCronica(t, s, owner, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campaign, 10)+"/editar", form.Encode())

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, queria 422", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), novaDescricao) {
		t.Error("a descrição digitada sumiu na recusa")
	}
	if !strings.Contains(rec.Body.String(), "O nome é obrigatório") {
		t.Error("a recusa não diz o que houve")
	}
	// E nada foi gravado.
	c, _ := s.queries.GetCampaign(context.Background(), campaign)
	if c.Name != "Nome antigo" {
		t.Errorf("o nome mudou para %q apesar da recusa", c.Name)
	}
}

// O interruptor ALTERNA, e o remendo volta com o estado novo. O conjunto
// guardado é o das regras DESLIGADAS — o padrão do livro é a regra valer.
func TestTheSwitchTogglesWhatIsInForceAndNotTheOpposite(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")
	campaign := seedCampanha(t, s, owner, "Mesa", "")
	route := "/campanhas/" + strconv.FormatInt(campaign, 10) + "/regras/carga"

	// Nasce EM VIGOR: nenhuma linha no banco significa "a regra vale".
	v, _ := campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "config")
	if !v.RegraEmVigor("carga") {
		t.Fatal("a regra nasceu desligada — o padrão do livro é ela valer")
	}

	if rec := pedeNaCronica(t, s, owner, http.MethodPost, route, ""); rec.Code != http.StatusOK {
		t.Fatalf("alternar respondeu %d", rec.Code)
	}
	v, _ = campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "config")
	if v.RegraEmVigor("carga") {
		t.Error("a regra continua em vigor depois de alternada")
	}

	if rec := pedeNaCronica(t, s, owner, http.MethodPost, route, ""); rec.Code != http.StatusOK {
		t.Fatalf("alternar de volta respondeu %d", rec.Code)
	}
	v, _ = campaigns.New(s.campaignsHost(), s.sessionAccess(), s.campaignDirectory(), s.campaignLifecycle(), s.campaignSeating(), s.boards).LoadOne(context.Background(), owner, s.ehAdmin(t, owner), campaign, "config")
	if !v.RegraEmVigor("carga") {
		t.Error("a regra não voltou a valer")
	}
}

func seedSessao(t *testing.T, s *Server, campaignID, number int64) int64 {
	t.Helper()
	now := dbvalue.NowISO()
	sess, err := s.queries.CreateSession(context.Background(), sqlcgen.CreateSessionParams{
		Campaignid: campaignID, Sessionnumber: number,
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		t.Fatalf("seed sessão %d: %v", number, err)
	}
	return sess.ID
}

// O ADMIN MESTRA EM QUALQUER MESA, e isso vale para EDITAR e não só para VER.
//
// A casa decide isso em três lugares: a lista entrega `gm` ao admin, o
// `Access.RoleIn` também, e o `Access.OwnedCampaign` deixa o admin passar pela
// mesma porta do dono — é com as ferramentas de mestre que ele vem consertar a
// mesa de um jogador no meio da sessão.
//
// A CENA discordava dos três. Ela tinha trava própria — o id do dono contra o
// de quem pede, sem a condição do admin —, então ela MOSTRAVA a aba de
// configuração ao administrador (porque o papel dele é `gm`) e devolvia 403
// quando ele salvava. Duas cópias de uma regra de autorização divergem em
// silêncio, e o sintoma é uma superfície deixando entrar quem a outra barra.
//
// O CONTROLE é o jogador, e ele vem junto: sem essa metade, o caso ficaria verde
// sobre uma trava que deixou de recusar qualquer um.
func TestTheAdminEditsSomeoneElsesCampaignAndAPlayerStillCannot(t *testing.T) {
	const emailDoAdmin = "chefe@t20.local"
	s := newTestServer(t, emailDoAdmin)
	owner := seedUser(t, s, "dono@t20.local")
	admin := seedUser(t, s, emailDoAdmin)
	player := seedUser(t, s, "jogador@t20.local")
	campaign := seedCampanha(t, s, owner, "Mesa", "")
	hero := seedCharacterAtLevel(t, s, player, "Yrla", "Arcanista", 4, 4, 4)
	seedMember(t, s, campaign, hero)
	edit := "/campanhas/" + strconv.FormatInt(campaign, 10) + "/editar"

	if rec := pedeNaCronica(t, s, admin, http.MethodPost, edit, "name=Consertada"); rec.Code == http.StatusForbidden {
		t.Error("o admin levou 403 ao salvar a mesa que a própria cena o deixa abrir como mestre")
	}
	if c, _ := s.queries.GetCampaign(context.Background(), campaign); c.Name != "Consertada" {
		t.Errorf("o nome ficou em %q — a edição do admin não chegou ao banco", c.Name)
	}

	if rec := pedeNaCronica(t, s, player, http.MethodPost, edit, "name=Roubada"); rec.Code != http.StatusForbidden {
		t.Errorf("o jogador salvou a mesa de outra pessoa: %d", rec.Code)
	}
}
