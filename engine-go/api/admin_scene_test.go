package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// OS GUARDAS DE COMPOSIÇÃO DA ADMINISTRAÇÃO (ALE-278).
//
// A cena virou `web/admin` e os casos de REGRA foram junto. Estes ficam porque
// montam um `api.Server` de verdade e dirigem o roteador de verdade — é onde se
// prova que a rota existe, que o `requireAdmin` recusa quem não é dono, e que o
// remendo volta com os painéis certos.

// O prazo é 24h e não os 7 dias do convite, e a diferença é de RISCO: o convite
// abre uma conta que ainda não existe; este abre uma que já existe e tem fichas
// dentro. Um link esquecido numa conversa vale mais para um estranho.
func TestTheResetLinkLastsTwentyFourHours(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")

	antes := time.Now()
	reset, err := s.mintPasswordReset(context.Background(), dono, dono)
	if err != nil {
		t.Fatalf("cunhar: %v", err)
	}
	expira, err := time.Parse(time.RFC3339, reset.Expiresat)
	if err != nil {
		t.Fatalf("expiresat %q não é RFC3339: %v", reset.Expiresat, err)
	}
	vida := expira.Sub(antes)
	if vida < 23*time.Hour || vida > 25*time.Hour {
		t.Errorf("o link vale %v, queria ~24h — o prazo do CONVITE é 7 dias e são coisas diferentes", vida)
	}
}

// Conta inexistente devolve um erro NOMEADO, e não um erro qualquer: é o que
// deixa quem chama escolher a resposta. A rota JSON traduz para 404, a cena do
// piloto para um aviso na tela — e nenhuma das duas repete a consulta.

// Conta inexistente devolve um erro NOMEADO, e não um erro qualquer: é o que
// deixa quem chama escolher a resposta. A rota JSON traduz para 404, a cena do
// piloto para um aviso na tela — e nenhuma das duas repete a consulta.
func TestMintingForAMissingAccountSaysItIsMissing(t *testing.T) {
	s := newTestServer(t)
	dono := seedUser(t, s, "dono@t20.local")

	_, err := s.mintPasswordReset(context.Background(), 999999, dono)
	if !errors.Is(err, errUserNotFound) {
		t.Errorf("erro = %v, queria errUsuarioInexistente", err)
	}
}

// O botão da linha ABRE o diálogo; quem posta é o de dentro. Mesma propriedade
// do Apagar e pela mesma razão: gerar um link é um efeito no banco, e o
// primeiro clique não deve produzi-lo.
//
// E ele LIMPA o `#reset-link` ao abrir. Isso não é zelo: sem limpar, gerar o
// link da Ana, fechar, e abrir a caixa da Bia mostraria o link da ANA sob o
// nome da BIA — link de redefinição entregue à pessoa errada. O e2e irmão prova
// isso no navegador; aqui se afirma que a limpeza está no marcador.

// Cunhar convite pela ADMINISTRAÇÃO remenda DUAS coisas: o link e o painel.
//
// O segundo é a diferença entre esta rota e a do Hub, e ele não é enfeite:
// aqui a lista de convites está a três centímetros do botão, e sem remendá-la a
// tela diz "Convites abertos (0)" logo depois de a pessoa abrir um. No Hub não
// existe essa lista, e por isso lá basta o link.
//
// O guarda é em Go e não no navegador de propósito: cunhar grava uma linha que
// a TELA não sabe revogar, então um e2e desta garantia deixaria lixo permanente
// no banco de desenvolvimento a cada corrida — que é a família de problema da
// ALE-238. Aqui o banco é descartável.
func TestMintingFromAdminPatchesThePanelToo(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	chefe := seedUser(t, s, "chefe@t20.local")

	rec := pedeNoPiloto(t, s, chefe, http.MethodPost, "/admin/convites")
	corpo := rec.Body.String()

	if !strings.Contains(corpo, "convite-url") {
		t.Errorf("o remendo do link não veio:\n%s", corpo)
	}
	if !strings.Contains(corpo, `id="painel-convites"`) {
		t.Error("o painel de convites não foi remendado — a contagem ao lado do botão fica velha")
	}
	if !strings.Contains(corpo, "Convites abertos (1)") {
		t.Errorf("o painel voltou sem contar o convite recém-cunhado:\n%s", corpo)
	}
}

// A trava é do SERVIDOR: quem não administra não cunha, mesmo postando na mão.
// A tela nem oferece o botão, mas isso é UX — a fronteira é aqui.
// O caso ANÔNIMO entrou na ALE-277, vindo do `TestOnlyAnAdminIssuesInvites`
// que media a rota JSON `/admin/invites`. Ele não repete o de cima: sem sessão
// a cena MANDA para a porta (303, `requirePage`), e com sessão sem coroa ela
// RECUSA (403, `requireAdmin`) — dois middlewares diferentes, e trocar um pelo
// outro deixaria o servidor pedindo login a quem já está logado.
func TestANonAdminDoesNotReachTheInviteRoute(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	seedUser(t, s, "chefe@t20.local")
	qualquerUm := seedUser(t, s, "outro@t20.local")

	rec := pedeNoPiloto(t, s, qualquerUm, http.MethodPost, "/admin/convites")
	if rec.Code != http.StatusForbidden {
		t.Errorf("a rota respondeu %d para quem não é admin, esperado 403", rec.Code)
	}

	semSessao := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(semSessao, httptest.NewRequest(http.MethodPost, "/admin/convites", nil))
	if semSessao.Code != http.StatusSeeOther {
		t.Errorf("sem credencial a cena respondeu %d, esperado 303 para a porta", semSessao.Code)
	}
}

// pedeNoPiloto bate no roteador do piloto, que é OUTRO que o `Router()` da API.
func pedeNoPiloto(t *testing.T, s *Server, userID int64, metodo, caminho string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.signToken(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(metodo, caminho, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}
