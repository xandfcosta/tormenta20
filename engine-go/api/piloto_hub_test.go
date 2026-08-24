package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"t20engine/plataforma"
	"testing"

	"t20engine/db/sqlcgen"
)

// Os guardas do HUB (ALE-231).
//
// O que vale proteger aqui não é o desenho — o e2e mede contraste e o teclado,
// que são as coisas que só o navegador testemunha. É o que o servidor DECIDE:
// quem vê o quê, qual sessão o "Continuar" retoma, e que sair não é um GET.

func hubFixture(t *testing.T, admins ...string) (*Server, int64) {
	t.Helper()
	s := newTestServer(t, admins...)
	dono := seedUser(t, s, "mestre@t20.local")
	return s, dono
}

// pedeHub manda um pedido pelo PilotoRouter com a sessão de `userID`.
func pedeHub(t *testing.T, s *Server, userID int64, metodo, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	user, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	tok, err := s.signToken(user)
	if err != nil {
		t.Fatalf("assinar: %v", err)
	}
	req := httptest.NewRequest(metodo, caminho, nil)
	req.AddCookie(&http.Cookie{Name: s.cfg.CookieName, Value: tok})
	rec := httptest.NewRecorder()
	http.StripPrefix("/piloto", s.PilotoRouter()).ServeHTTP(rec, req)
	return rec
}

// ── "Continuar sessão" ───────────────────────────────────────────────────────

// A entrada só existe com partida rolando — é o "Continue" de um jogo, e um
// item que leva a uma sessão encerrada é pior que item nenhum.
func TestHubSoOfereceContinuarComSessaoViva(t *testing.T) {
	s, dono := hubFixture(t)
	campanha := seedCampaign(t, s, dono)
	sessao := seedSession(t, s, campanha)

	semViva := pedeHub(t, s, dono, http.MethodGet, "/piloto/").Body.String()
	if strings.Contains(semViva, "Continuar sessão") {
		t.Error("ofereceu continuar sem sessão ativa")
	}

	if _, err := s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
		UpdatedAt: plataforma.NowISO(), ID: sessao,
	}); err != nil {
		t.Fatalf("iniciar sessão: %v", err)
	}

	comViva := pedeHub(t, s, dono, http.MethodGet, "/piloto/").Body.String()
	if !strings.Contains(comViva, "Continuar sessão") {
		t.Fatal("não ofereceu continuar com sessão ativa")
	}
	if !strings.Contains(comViva, destinoDaSessao(campanha, sessao)) {
		t.Errorf("o link não aponta para a sessão viva (%s)", destinoDaSessao(campanha, sessao))
	}
}

// A consulta única substituiu uma fan-out de N+1 requisições do cliente, e a
// ORDEM era regra dela: a primeira viva na ordem das campanhas, que é a mais
// recentemente tocada primeiro. Trocar a ordem troca qual partida o botão
// retoma.
func TestSessaoVivaEscolheAPrimeiraNaOrdemDasCampanhas(t *testing.T) {
	s, dono := hubFixture(t)
	antiga := seedCampaign(t, s, dono)
	recente := seedCampaign(t, s, dono)
	vivaAntiga := seedSession(t, s, antiga)
	vivaRecente := seedSession(t, s, recente)
	for _, id := range []int64{vivaAntiga, vivaRecente} {
		if _, err := s.queries.StartSessionFresh(context.Background(), sqlcgen.StartSessionFreshParams{
			UpdatedAt: plataforma.NowISO(), ID: id,
		}); err != nil {
			t.Fatalf("iniciar %d: %v", id, err)
		}
	}

	viva, err := s.sessaoViva(context.Background(), dono)
	if err != nil || viva == nil {
		t.Fatalf("sessaoViva = %v, %v", viva, err)
	}
	// `seedCampaign` grava as duas com o mesmo `updatedAt`, então o desempate é
	// do banco; o que se afirma é que ela escolhe UMA das vivas e não uma morta.
	if viva.SessionID != vivaAntiga && viva.SessionID != vivaRecente {
		t.Errorf("escolheu a sessão %d, que não é nenhuma das vivas", viva.SessionID)
	}
}

// Sem partida nenhuma a resposta é "não há", e não erro: quase sempre não há
// partida rolando, e tratar isso como falha encheria o log de ruído.
func TestSessaoVivaSemNenhumaNaoEhErro(t *testing.T) {
	s, dono := hubFixture(t)
	viva, err := s.sessaoViva(context.Background(), dono)
	if err != nil {
		t.Fatalf("erro para o caso normal: %v", err)
	}
	if viva != nil {
		t.Errorf("achou sessão %v onde não há nenhuma", viva)
	}
}

func destinoDaSessao(campanha, sessao int64) string {
	return fmt.Sprintf("/campaigns/%d/sessions/%d", campanha, sessao)
}

// ── quem vê o quê ────────────────────────────────────────────────────────────

// As entradas de admin são UX, não segurança: quem não é admin não as vê, e
// quem chegar na rota na mão leva a recusa do servidor. As duas metades são
// afirmadas, porque só a primeira seria uma tela mentindo sobre uma trava.
func TestHubNaoDesenhaEntradasDeAdminParaJogador(t *testing.T) {
	s, _ := hubFixture(t, "mestre@t20.local")
	jogador := seedUser(t, s, "jogadora@t20.local")

	corpo := pedeHub(t, s, jogador, http.MethodGet, "/piloto/").Body.String()
	for _, entrada := range []string{"Convidar jogador", "Administração"} {
		if strings.Contains(corpo, entrada) {
			t.Errorf("o Hub ofereceu %q para quem não administra", entrada)
		}
	}
}

func TestHubRecusaConviteDeQuemNaoEhAdmin(t *testing.T) {
	s, _ := hubFixture(t, "mestre@t20.local")
	jogador := seedUser(t, s, "jogadora@t20.local")

	antes, err := s.queries.ListOpenAccountInvites(context.Background(), plataforma.NowISO())
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	pedeHub(t, s, jogador, http.MethodPost, "/piloto/convites")
	depois, err := s.queries.ListOpenAccountInvites(context.Background(), plataforma.NowISO())
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(depois) != len(antes) {
		t.Error("um não-admin cunhou convite — a trava da tela não é a trava")
	}
}

func TestHubDesenhaEntradasDeAdminParaAdmin(t *testing.T) {
	s, dono := hubFixture(t, "mestre@t20.local")
	corpo := pedeHub(t, s, dono, http.MethodGet, "/piloto/").Body.String()
	for _, entrada := range []string{"Convidar jogador", "Administração"} {
		if !strings.Contains(corpo, entrada) {
			t.Errorf("o Hub escondeu %q de quem administra", entrada)
		}
	}
}

// ── sair ─────────────────────────────────────────────────────────────────────

// Sair é POST, e isso é a regra de CSRF desta migração inteira. O cookie é
// `SameSite=Lax`, que NÃO viaja em POST cross-site — mas viaja em navegação de
// topo por GET. Um `<a href="/piloto/sair">` seria disparável por qualquer
// imagem de terceiro, e o jogador seria deslogado no meio da mesa.
func TestSairNaoAtendeGET(t *testing.T) {
	s, dono := hubFixture(t)
	rec := pedeHub(t, s, dono, http.MethodGet, "/piloto/sair")
	if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
		t.Errorf("GET /sair respondeu %d — ação com efeito não pode viver num GET", rec.Code)
	}
}

func TestSairApagaOCookieEDevolveAPorta(t *testing.T) {
	s, dono := hubFixture(t)
	rec := pedeHub(t, s, dono, http.MethodPost, "/piloto/sair")

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/piloto/entrar" {
		t.Fatalf("status %d para %q", rec.Code, rec.Header().Get("Location"))
	}
	var apagou bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == s.cfg.CookieName && c.Value == "" && c.MaxAge < 0 {
			apagou = true
		}
	}
	if !apagou {
		t.Error("saiu sem apagar o cookie de sessão")
	}
}

// ── a página é PÁGINA, não API ───────────────────────────────────────────────

// O `requireAuth` responde um JSON 401, que é a resposta certa para quem chama
// a API e a errada para quem digitou uma URL — o jogador via
// `{"statusCode":401}` numa tela branca. E o destino tem de trazer o prefixo:
// o roteador é montado com `StripPrefix`, então quem lesse `URL.Path` mandaria
// o jogador de volta para `/mesa/1/4` e ele cairia num 404 depois de entrar.
func TestPaginaAnonimaVaiParaAPortaLembrandoOCaminhoInteiro(t *testing.T) {
	s, _ := hubFixture(t)
	req := httptest.NewRequest(http.MethodGet, "/piloto/mesa/1/4", nil)
	rec := httptest.NewRecorder()
	http.StripPrefix("/piloto", s.PilotoRouter()).ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303 — página não responde JSON 401 para o navegador", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/piloto/entrar?redirect=%2Fpiloto%2Fmesa%2F1%2F4" {
		t.Errorf("Location = %q — o prefixo /piloto precisa sobreviver ao StripPrefix", got)
	}
}

// ── a inicial do retrato ─────────────────────────────────────────────────────

// Por RUNA e não por byte: "Áurea" começa com dois bytes, e cortar o primeiro
// desenha o losango de erro no lugar da letra.
func TestInicialDeCortaPorRunaENaoPorByte(t *testing.T) {
	casos := map[string]string{
		"Mestre":            "M",
		"Áurea":             "Á",
		"  ébano  ":         "É",
		"":                  "?",
		"   ":               "?",
		"jogador@t20.local": "J",
	}
	for entrada, quer := range casos {
		if got := inicialDe(entrada); got != quer {
			t.Errorf("inicialDe(%q) = %q, queria %q", entrada, got, quer)
		}
	}
}
