package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"t20engine/app/accounts"
	"testing"
	"time"
)

// OS GUARDAS DE COMPOSIÇÃO DA ADMINISTRAÇÃO.
//
// Os casos de REGRA moram em `web/admin`; estes ficam aqui porque montam um
// `api.Server` de verdade e dirigem o roteador de verdade — é onde se prova que
// a rota existe, que o `requireAdmin` recusa quem não é dono, e que o remendo
// volta com os painéis certos.

// O prazo é 24h e não os 7 dias do convite, e a diferença é de RISCO: o convite
// abre uma conta que ainda não existe; este abre uma que já existe e tem fichas
// dentro. Um link esquecido numa conversa vale mais para um estranho.
func TestTheResetLinkLastsTwentyFourHours(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")

	before := time.Now()
	reset, err := s.accountResets().Mint(context.Background(), owner, owner)
	if err != nil {
		t.Fatalf("cunhar: %v", err)
	}
	expires, err := time.Parse(time.RFC3339, reset.Expiresat)
	if err != nil {
		t.Fatalf("expiresat %q não é RFC3339: %v", reset.Expiresat, err)
	}
	life := expires.Sub(before)
	if life < 23*time.Hour || life > 25*time.Hour {
		t.Errorf("o link vale %v, queria ~24h — o prazo do CONVITE é 7 dias e são coisas diferentes", life)
	}
}

// Conta inexistente devolve um erro NOMEADO, e não um erro qualquer: é o que
// deixa quem chama escolher a resposta. A rota JSON traduz para 404, a cena
// para um aviso na tela — e nenhuma das duas repete a consulta.
func TestMintingForAMissingAccountSaysItIsMissing(t *testing.T) {
	s := newTestServer(t)
	owner := seedUser(t, s, "dono@t20.local")

	_, err := s.accountResets().Mint(context.Background(), 999999, owner)
	if !errors.Is(err, accounts.ErrUnknownAccount) {
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
// O guarda é em Go e não no navegador de propósito: é regra de handler, e a camada
// mais barata que a segura é esta.
func TestMintingFromAdminPatchesThePanelToo(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")

	rec := askScene(t, s, boss, http.MethodPost, "/admin/convites")
	body := rec.Body.String()

	if !strings.Contains(body, "invite-url") {
		t.Errorf("o remendo do link não veio:\n%s", body)
	}
	if !strings.Contains(body, `id="invites-panel"`) {
		t.Error("o painel de convites não foi remendado — a contagem ao lado do botão fica velha")
	}
	if !strings.Contains(body, "Convites abertos · 1") {
		t.Errorf("o painel voltou sem contar o convite recém-cunhado:\n%s", body)
	}
}

// A trava é do SERVIDOR: quem não administra não cunha, mesmo postando na mão.
// A tela nem oferece o botão, mas isso é UX — a fronteira é aqui.
//
// O caso ANÔNIMO não repete o de cima: sem sessão a cena MANDA para a porta
// (303, `requirePage`), e com sessão sem coroa ela RECUSA (403,
// `requireAdmin`) — dois middlewares diferentes, e trocar um pelo outro
// deixaria o servidor pedindo login a quem já está logado.
func TestANonAdminDoesNotReachTheInviteRoute(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	seedUser(t, s, "chefe@t20.local")
	anyone := seedUser(t, s, "outro@t20.local")

	rec := askScene(t, s, anyone, http.MethodPost, "/admin/convites")
	if rec.Code != http.StatusForbidden {
		t.Errorf("a rota respondeu %d para quem não é admin, esperado 403", rec.Code)
	}

	noSession := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(noSession, httptest.NewRequest(http.MethodPost, "/admin/convites", nil))
	if noSession.Code != http.StatusSeeOther {
		t.Errorf("sem credencial a cena respondeu %d, esperado 303 para a porta", noSession.Code)
	}
}

// askScene bate no roteador do app, que é OUTRO que o `Router()` da API.
func askScene(t *testing.T, s *Server, userID int64, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	u, err := s.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("usuário: %v", err)
	}
	token, err := s.accountGate().SignSession(u)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	s.WebRouter().ServeHTTP(rec, req)
	return rec
}

// A administração veste o Tempestade.
func TestTheAdminWearsTheNewSystem(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")
	body := askScene(t, s, boss, http.MethodGet, "/admin").Body.String()

	shell := regexp.MustCompile(`<section[^>]*data-slot="scene-shell"[^>]*>`).FindString(body)
	if !strings.Contains(shell, "scene-ds") {
		t.Errorf("a administração não saiu no escopo do Tempestade: %s", shell)
	}
	if !strings.Contains(body, `<h1 class="ds-header-title">Administração</h1>`) {
		t.Error("o cabeçalho não diz Administração")
	}
}

// revokeAddress é a rota de revogar o convite daquele token.
func revokeAddress(t *testing.T, s *Server, token string) string {
	t.Helper()
	invite, err := s.queries.GetAccountInvite(context.Background(), token)
	if err != nil {
		t.Fatalf("achar o convite: %v", err)
	}
	return fmt.Sprintf("/admin/convites/%d/revogar", invite.ID)
}

// Revogar vale no CADASTRO, e não só na tela: quem já recebeu o link não entra mais.
func TestARevokedInviteNoLongerOpensAnAccount(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")
	token := inviteFrom(t, s, boss)

	page := askScene(t, s, boss, http.MethodGet, "/admin").Body.String()
	label := "Revogar o Link de convite " + token[:6]
	if !strings.Contains(page, label) {
		t.Fatalf("o convite aberto não oferece %q", label)
	}

	rec := askScene(t, s, boss, http.MethodPost, revokeAddress(t, s, token))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `id="invites-panel"`) {
		t.Error("o painel de convites não foi remendado")
	}
	if strings.Contains(rec.Body.String(), label) {
		t.Error("o remendo ainda lista o convite revogado")
	}
	if err := registra(t, s, "revogado@t20.local", token); !errors.Is(err, accounts.ErrBadInvite) {
		t.Errorf("o link revogado tinha de ser recusado no cadastro, veio %v", err)
	}
}

// Só se revoga o que está aberto: o convite gasto continua no registro de quem entrou
// por ele, e a tela diz que não deu em vez de fingir que revogou.
func TestAUsedInviteCannotBeRevoked(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")
	token := inviteFrom(t, s, boss)
	address := revokeAddress(t, s, token)
	if err := registra(t, s, "usado@t20.local", token); err != nil {
		t.Fatalf("o cadastro com o convite novo falhou: %v", err)
	}

	rec := askScene(t, s, boss, http.MethodPost, address)
	if !strings.Contains(rec.Body.String(), "não está mais aberto") {
		t.Errorf("revogar o convite gasto não disse que ele não está aberto: %s", rec.Body.String())
	}
	if _, err := s.queries.GetAccountInvite(context.Background(), token); err != nil {
		t.Errorf("revogar apagou o registro do convite gasto: %v", err)
	}
}

// Quem não administra não revoga: a trava é do servidor, não do botão.
func TestANonAdminDoesNotRevoke(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	token := inviteFrom(t, s, seedUser(t, s, "chefe@t20.local"))
	outsider := seedUser(t, s, "jogador@t20.local")

	rec := askScene(t, s, outsider, http.MethodPost, revokeAddress(t, s, token))
	if rec.Code != http.StatusForbidden {
		t.Errorf("status %d, quero 403", rec.Code)
	}
	if err := registra(t, s, "ainda-vale@t20.local", token); err != nil {
		t.Errorf("quem não administra derrubou o convite: %v", err)
	}
}

// A lista traz todo backup, do mais novo para o mais velho, e diz quantos ficam guardados.
func TestTheAdminListsEveryBackup(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")
	s.cfg.BackupDir = t.TempDir()
	s.cfg.BackupKeep = 7
	for name, at := range map[string]time.Time{
		"t20--velho.db": time.Date(2026, 10, 30, 3, 0, 0, 0, time.Local),
		"t20--novo.db":  time.Date(2026, 11, 1, 3, 0, 0, 0, time.Local),
	} {
		path := filepath.Join(s.cfg.BackupDir, name)
		if err := os.WriteFile(path, []byte("sqlite"), 0o600); err != nil {
			t.Fatalf("escrever %s: %v", name, err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatalf("datar %s: %v", name, err)
		}
	}

	body := askScene(t, s, boss, http.MethodGet, "/admin").Body.String()
	if !strings.Contains(body, "2 de 7") {
		t.Error("a seção não diz quantos backups ficam guardados")
	}
	if !strings.Contains(body, "01/11/2026, 03:00") || !strings.Contains(body, "30/10/2026, 03:00") {
		t.Error("a lista não traz a data de cada backup")
	}
	novo, velho := strings.Index(body, "t20--novo.db"), strings.Index(body, "t20--velho.db")
	if novo < 0 || velho < 0 || novo > velho {
		t.Errorf("a ordem não é do mais novo para o mais velho (novo em %d, velho em %d)", novo, velho)
	}
}

// Sem backup, a seção diz isso, e o botão de fazer um continua lá.
func TestWithoutBackupsTheAdminSaysSo(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")
	s.cfg.BackupDir = t.TempDir()

	body := askScene(t, s, boss, http.MethodGet, "/admin").Body.String()
	if !strings.Contains(body, "Nenhum backup ainda.") || !strings.Contains(body, "Fazer backup") {
		t.Error("a seção vazia não diz que não há backup, ou perdeu o botão")
	}
}

// O link do convite LEVA à porta de cadastro. Os dois lugares que cunham e a lista são
// seguidos até o fim: o caminho sai do que a tela entrega, e não é escrito aqui.
func TestTheInviteLinkOpensTheSignUp(t *testing.T) {
	s := newTestServer(t, "chefe@t20.local")
	boss := seedUser(t, s, "chefe@t20.local")
	path := regexp.MustCompile(`data-path="([^"]+)"`)
	listed := regexp.MustCompile(`data-url="([^"]+)"`)

	fromAdmin := path.FindStringSubmatch(askScene(t, s, boss, http.MethodPost, "/admin/convites").Body.String())
	fromHub := path.FindStringSubmatch(askScene(t, s, boss, http.MethodPost, "/convites").Body.String())
	fromList := listed.FindStringSubmatch(askScene(t, s, boss, http.MethodGet, "/admin").Body.String())
	for name, found := range map[string][]string{"administração": fromAdmin, "início": fromHub, "lista": fromList} {
		if found == nil {
			t.Fatalf("%s: a tela não entregou caminho nenhum", name)
		}
		address := strings.ReplaceAll(found[1], "&amp;", "&")
		rec := httptest.NewRecorder()
		s.WebRouter().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, address, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: o link %s deu status %d", name, address, rec.Code)
			continue
		}
		if !strings.Contains(rec.Body.String(), `action="/criar-conta"`) {
			t.Errorf("%s: o link %s não abriu o cadastro", name, address)
		}
	}
}
