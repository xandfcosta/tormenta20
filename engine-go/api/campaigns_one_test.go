package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"t20engine/db/sqlcgen"
	"t20engine/plataforma"
	"t20engine/web/campaigns"
	"testing"
)

// Os guardas da CRÔNICA (ALE-255).

func pedeNaCronica(t *testing.T, s *Server, userID int64, metodo, caminho, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.accountRules().signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	if corpo != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// AS SESSÕES RECENTES SÃO AS RECENTES. O `ListSessions` ordena por número
// CRESCENTE, e a primeira versão desta cena pegava as três primeiras — que são
// as mais ANTIGAS. O defeito não aparece numa mesa com três sessões, só numa
// que já jogou bastante, e a tela não tem como avisar que está mentindo.
func TestSessionsComeFromTheNewestToTheOldest(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "A Queda de Tauron", "")
	for i := 1; i <= 5; i++ {
		seedSessao(t, s, campanha, int64(i))
	}

	v, err := campaigns.New(s.campaignsHost()).LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "")
	if err != nil {
		t.Fatalf("carregar: %v", err)
	}
	if len(v.Sessoes) != 5 {
		t.Fatalf("esperava 5 sessões, veio %d", len(v.Sessoes))
	}
	if v.Sessoes[0].Numero != 5 || v.Sessoes[4].Numero != 1 {
		numeros := make([]int64, len(v.Sessoes))
		for i, sess := range v.Sessoes {
			numeros[i] = sess.Numero
		}
		t.Errorf("ordem = %v, queria da mais nova para a mais velha", numeros)
	}
}

// O MESTRE VEM PRIMEIRO no elenco. É a regra do `sortRoster` da SPA, e ela é o
// que faz o grupo se ler como grupo em vez de fila.
//
// # Ela nunca aconteceu, e este caso passava mesmo assim (ALE-287)
//
// A ordenação comparava a coluna `campaign_members.role`, que valia `'player'`
// em toda linha que a produção escreveu — o único escritor fixava a string. O
// comparador devolvia zero para todo par e a lista saía na ordem de entrada; a
// coroa ao lado do nome nunca foi desenhada.
//
// O que fazia este caso passar era a BANCADA: o `seedMember` recebia um papel e
// escrevia `"gm"`, um estado que só ela sabia produzir. Verde sobre dado que a
// produção não tem é a mesma família do convite desta issue — e as duas moravam
// no mesmo arquivo de fixture.
//
// Hoje quem mestra é o DONO da mesa, e é por isso que os dois personagens
// precisam de donos DIFERENTES: com os dois pertencendo ao mesmo usuário, os
// dois são do mestre, e não haveria fila para ordenar.
func TestTheGmComesFirstInTheCast(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	quemJoga := seedUser(t, s, "jogador@t20.local")
	campanha := seedCampanha(t, s, dono, "Mesa", "")
	jogador := seedCharacterAtLevel(t, s, quemJoga, "Yrla", 4, 10, 14, 2, 6)
	mestre := seedCharacterAtLevel(t, s, dono, "Thalen", 5, 16, 12, 3, 8)
	seedMember(t, s, campanha, jogador)
	seedMember(t, s, campanha, mestre)

	v, err := campaigns.New(s.campaignsHost()).LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "")
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
	dono := seedUser(t, s, "dono@t20.local")
	visitante := seedUser(t, s, "visitante@t20.local")
	campanha := seedCampanha(t, s, dono, "Mesa", "")
	heroi := seedCharacterAtLevel(t, s, visitante, "Yrla", 4, 10, 14, 2, 6)
	seedMember(t, s, campanha, heroi)

	v, err := campaigns.New(s.campaignsHost()).LoadOne(context.Background(), visitante, s.ehAdmin(t, visitante), campanha, "config")
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
	dono := seedUser(t, s, "dono@t20.local")
	visitante := seedUser(t, s, "visitante@t20.local")
	campanha := seedCampanha(t, s, dono, "Mesa", "")
	heroi := seedCharacterAtLevel(t, s, visitante, "Yrla", 4, 10, 14, 2, 6)
	seedMember(t, s, campanha, heroi)
	base := "/campanhas/" + strconv.FormatInt(campanha, 10)

	for _, caminho := range []string{base + "/editar", base + "/excluir", base + "/regras/carga"} {
		rec := pedeNaCronica(t, s, visitante, http.MethodPost, caminho, "name=Roubada")
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s respondeu %d para o jogador, queria 403", caminho, rec.Code)
		}
	}
	// E a campanha continua intacta depois das três tentativas.
	c, err := s.queries.GetCampaign(context.Background(), campanha)
	if err != nil || c.Name != "Mesa" {
		t.Errorf("a campanha mudou: %+v (%v)", c, err)
	}
}

// A recusa do cadastro DEVOLVE O QUE FOI DIGITADO, e não o que está no banco:
// a pessoa está olhando para o próprio texto, e devolver o antigo apagaria a
// edição dela na cara.
func TestTheSignUpRefusalGivesBackTheTypedText(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "Nome antigo", "")
	const novaDescricao = "A caravana parte de Valkaria ao amanhecer."

	form := url.Values{"name": {"   "}, "description": {novaDescricao}}
	rec := pedeNaCronica(t, s, dono, http.MethodPost,
		"/campanhas/"+strconv.FormatInt(campanha, 10)+"/editar", form.Encode())

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
	c, _ := s.queries.GetCampaign(context.Background(), campanha)
	if c.Name != "Nome antigo" {
		t.Errorf("o nome mudou para %q apesar da recusa", c.Name)
	}
}

// O interruptor ALTERNA, e o remendo volta com o estado novo. O conjunto
// guardado é o das regras DESLIGADAS — o padrão do livro é a regra valer.
func TestTheSwitchTogglesWhatIsInForceAndNotTheOpposite(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")
	campanha := seedCampanha(t, s, dono, "Mesa", "")
	rota := "/campanhas/" + strconv.FormatInt(campanha, 10) + "/regras/carga"

	// Nasce EM VIGOR: nenhuma linha no banco significa "a regra vale".
	v, _ := campaigns.New(s.campaignsHost()).LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "config")
	if !v.RegraEmVigor("carga") {
		t.Fatal("a regra nasceu desligada — o padrão do livro é ela valer")
	}

	if rec := pedeNaCronica(t, s, dono, http.MethodPost, rota, ""); rec.Code != http.StatusOK {
		t.Fatalf("alternar respondeu %d", rec.Code)
	}
	v, _ = campaigns.New(s.campaignsHost()).LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "config")
	if v.RegraEmVigor("carga") {
		t.Error("a regra continua em vigor depois de alternada")
	}

	if rec := pedeNaCronica(t, s, dono, http.MethodPost, rota, ""); rec.Code != http.StatusOK {
		t.Fatalf("alternar de volta respondeu %d", rec.Code)
	}
	v, _ = campaigns.New(s.campaignsHost()).LoadOne(context.Background(), dono, s.ehAdmin(t, dono), campanha, "config")
	if !v.RegraEmVigor("carga") {
		t.Error("a regra não voltou a valer")
	}
}

func seedSessao(t *testing.T, s *Server, campanhaID, numero int64) int64 {
	t.Helper()
	agora := plataforma.NowISO()
	sess, err := s.queries.CreateSession(context.Background(), sqlcgen.CreateSessionParams{
		Campaignid: campanhaID, Sessionnumber: numero,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		t.Fatalf("seed sessão %d: %v", numero, err)
	}
	return sess.ID
}
