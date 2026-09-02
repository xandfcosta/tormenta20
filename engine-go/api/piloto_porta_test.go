package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"t20engine/plataforma"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"t20engine/db/sqlcgen"
)

// Os guardas da PORTA (ALE-229).
//
// O que vale proteger aqui não é o desenho: é que a tela nova obedece às mesmas
// recusas que a API obedece, e que a senha não vira estado de cliente. Guarda de
// aparência não entra — o e2e mede contraste, e classe de CSS ninguém prometeu.

type portaFixture struct {
	s     *Server
	email string
	senha string
}

func novaPorta(t *testing.T, admins ...string) portaFixture {
	t.Helper()
	s := newTestServer(t, admins...)
	f := portaFixture{s: s, email: "jogadora@t20.local", senha: "senha-de-verdade"}
	hash, err := bcrypt.GenerateFromPassword([]byte(f.senha), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if _, err := s.queries.CreateUser(context.Background(), sqlcgen.CreateUserParams{
		Email: f.email, Passwordhash: string(hash), Createdat: plataforma.NowISO(), Updatedat: plataforma.NowISO(),
	}); err != nil {
		t.Fatalf("semear conta: %v", err)
	}
	return f
}

// bate manda um pedido pelo WebRouter. `form` nil = GET.
func (f portaFixture) bate(t *testing.T, caminho string, form url.Values, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	metodo, corpo := http.MethodGet, ""
	if form != nil {
		metodo, corpo = http.MethodPost, form.Encode()
	}
	req := httptest.NewRequest(metodo, caminho, strings.NewReader(corpo))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != "" {
		req.AddCookie(&http.Cookie{Name: f.s.cfg.CookieName, Value: cookie})
	}
	rec := httptest.NewRecorder()
	f.s.WebRouter().ServeHTTP(rec, req)
	return rec
}

func (f portaFixture) sessao(t *testing.T) string {
	t.Helper()
	user, err := f.s.queries.GetUserByEmail(context.Background(), f.email)
	if err != nil {
		t.Fatalf("conta sumiu: %v", err)
	}
	tok, err := f.s.signToken(user)
	if err != nil {
		t.Fatalf("assinar: %v", err)
	}
	return tok
}

func temCookieDeSessao(f portaFixture, rec *httptest.ResponseRecorder) bool {
	for _, c := range rec.Result().Cookies() {
		if c.Name == f.s.cfg.CookieName && c.Value != "" {
			return true
		}
	}
	return false
}

// ── entrar ───────────────────────────────────────────────────────────────────

// A recusa é a metade que importa: uma porta que devolve 200 e uma tela sem
// aviso deixa o jogador achando que entrou.
func TestTheDoorRefusesAWrongPasswordWithoutOpeningASession(t *testing.T) {
	f := novaPorta(t)
	rec := f.bate(t, "/entrar", url.Values{
		"email": {f.email}, "senha": {"não é essa"},
	}, "")

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, queria 401 — uma recusa que responde 200 mente para tudo o que não é navegador", rec.Code)
	}
	if temCookieDeSessao(f, rec) {
		t.Fatal("a senha errada abriu SESSÃO")
	}
	if !strings.Contains(rec.Body.String(), avisoCredenciais) {
		t.Error("a tela voltou sem dizer o que houve")
	}
}

// E-mail que não existe e senha errada respondem a MESMA coisa: distinguir os
// dois entrega a quem sonda a lista de quem tem conta na mesa.
func TestTheDoorDoesNotTellAMissingAccountFromAWrongPassword(t *testing.T) {
	f := novaPorta(t)
	inexistente := f.bate(t, "/entrar", url.Values{
		"email": {"ninguem@t20.local"}, "senha": {"seja o que for"},
	}, "")
	errada := f.bate(t, "/entrar", url.Values{
		"email": {f.email}, "senha": {"não é essa"},
	}, "")

	if inexistente.Code != errada.Code {
		t.Errorf("status diferentes (%d vs %d) — dá para enumerar contas", inexistente.Code, errada.Code)
	}
	if !strings.Contains(inexistente.Body.String(), avisoCredenciais) {
		t.Error("a conta inexistente recebeu outra frase")
	}
}

func TestTheDoorSignsInAndSendsToTheDestination(t *testing.T) {
	f := novaPorta(t)
	rec := f.bate(t, "/entrar", url.Values{
		"email": {f.email}, "senha": {f.senha}, "destino": {"/campaigns/7"},
	}, "")

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303 (Post/Redirect/Get: sem ele, recarregar reenvia o formulário)", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/campaigns/7" {
		t.Errorf("Location = %q, queria /campaigns/7", got)
	}
	if !temCookieDeSessao(f, rec) {
		t.Error("entrou e não abriu sessão")
	}
}

// O `?redirect=` da SPA vira campo oculto aqui, e um destino EXTERNO
// transformaria a porta em redirecionamento aberto: o link sai do nosso
// domínio, o jogador confia nele, e a página que recebe pode imitar esta.
func TestTheRequestedDestinationOnlyAcceptsAnInternalPath(t *testing.T) {
	casos := map[string]string{
		"/campaigns/7":        "/campaigns/7",
		"":                    "/",
		"https://outro.site":  "/",
		"//outro.site":        "/", // protocol-relative: o navegador trata como absoluto
		"javascript:alert(1)": "/",
	}
	for entrada, quer := range casos {
		if got := destinoPedido(entrada); got != quer {
			t.Errorf("destinoPedido(%q) = %q, queria %q", entrada, got, quer)
		}
	}
}

func TestTheDoorSendsWhoAlreadyHasASessionAwayFromTheSignInScreen(t *testing.T) {
	f := novaPorta(t)
	rec := f.bate(t, "/entrar", nil, f.sessao(t))
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, queria 303 — quem já entrou não vê a porta", rec.Code)
	}
}

// ── criar conta ──────────────────────────────────────────────────────────────

// A porta já era fechada (o servidor responde 403), mas a TELA ficava aberta e
// parecia um cadastro comum (ALE-120).
func TestTheDoorDoesNotOpenSignUpWithoutAnInvite(t *testing.T) {
	f := novaPorta(t)
	rec := f.bate(t, "/criar-conta", nil, "")
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/entrar" {
		t.Errorf("status %d para %q — sem convite a tela não abre", rec.Code, rec.Header().Get("Location"))
	}
}

func TestTheDoorRefusesAnInvalidInviteInPortuguese(t *testing.T) {
	f := novaPorta(t)
	rec := f.bate(t, "/criar-conta", url.Values{
		"convite": {"não-existe"}, "email": {"nova@t20.local"},
		"senha": {"uma senha boa"}, "confirmar": {"uma senha boa"},
	}, "")

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, queria 403", rec.Code)
	}
	corpo := rec.Body.String()
	if !strings.Contains(corpo, avisoConvite) {
		t.Error("a recusa do convite não chegou à tela")
	}
	if strings.Contains(corpo, inviteRejected) {
		t.Errorf("a frase EM INGLÊS da API vazou para a tela: %q", inviteRejected)
	}
}

// O `confirmar` não existe no corpo JSON da API — ele é do formulário. Conferir
// no SERVIDOR e não só no `data-on:input` é o que mantém a proteção contra o
// typo com JavaScript desligado.
func TestTheDoorRefusesPasswordsThatDoNotMatchOnTheServer(t *testing.T) {
	f := novaPorta(t, "dono@t20.local")
	rec := f.bate(t, "/criar-conta", url.Values{
		"convite": {"tanto-faz"}, "email": {"dono@t20.local"},
		"senha": {"uma senha boa"}, "confirmar": {"outra senha"},
	}, "")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, queria 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), avisoConfere) {
		t.Error("o typo passou sem aviso")
	}
	if _, err := f.s.queries.GetUserByEmail(context.Background(), "dono@t20.local"); err == nil {
		t.Fatal("criou a conta mesmo com as senhas diferentes")
	}
}

// A mensagem do validador é a que o jogador lê, e ela era em INGLÊS: a SPA
// escondia isso validando com Zod antes de chamar.
func TestTheDoorSaysValidationRefusalsInPortuguese(t *testing.T) {
	f := novaPorta(t)
	rec := f.bate(t, "/entrar", url.Values{
		"email": {"isto-não-é-e-mail"}, "senha": {"x"},
	}, "")

	corpo := rec.Body.String()
	if !strings.Contains(corpo, msgEmailInvalido) {
		t.Errorf("não achei %q na tela", msgEmailInvalido)
	}
	if strings.Contains(corpo, "must be an email") {
		t.Error("a frase em inglês do class-validator vazou para a tela")
	}
}

// ── redefinir senha ──────────────────────────────────────────────────────────

func (f portaFixture) semeiaLink(t *testing.T, validade time.Duration) string {
	t.Helper()
	user, err := f.s.queries.GetUserByEmail(context.Background(), f.email)
	if err != nil {
		t.Fatalf("conta: %v", err)
	}
	agora := time.Now()
	reset, err := f.s.queries.CreatePasswordReset(context.Background(), sqlcgen.CreatePasswordResetParams{
		Token: generateInviteToken(), Userid: user.ID, Createdby: user.ID,
		Createdat: plataforma.IsoAt(agora), Expiresat: plataforma.IsoAt(agora.Add(validade)),
	})
	if err != nil {
		t.Fatalf("semear link: %v", err)
	}
	return reset.Token
}

// Perguntar pelo link ANTES de desenhar o formulário é o ponto da tela: falhar
// no envio com a senha já digitada duas vezes é pior.
func TestTheDoorShowsNoFormWithAnExpiredLink(t *testing.T) {
	f := novaPorta(t)
	token := f.semeiaLink(t, -time.Hour)

	rec := f.bate(t, "/redefinir-senha?token="+token, nil, "")
	corpo := rec.Body.String()

	if !strings.Contains(corpo, avisoLink) {
		t.Error("o link vencido não disse que não vale mais")
	}
	if strings.Contains(corpo, `name="senha"`) {
		t.Error("desenhou o formulário para um link que não vale")
	}
}

func TestTheDoorChangesThePasswordAndSendsBackToSignIn(t *testing.T) {
	f := novaPorta(t)
	token := f.semeiaLink(t, time.Hour)
	nova := "outra senha boa"

	rec := f.bate(t, "/redefinir-senha", url.Values{
		"token": {token}, "senha": {nova}, "confirmar": {nova},
	}, "")

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/entrar" {
		t.Fatalf("status %d para %q", rec.Code, rec.Header().Get("Location"))
	}
	// Sem cookie: um link de recuperação chega por um canal que ninguém
	// controla, e emitir sessão aqui o transformaria num login.
	if temCookieDeSessao(f, rec) {
		t.Error("o link de redefinição abriu sessão")
	}
	if _, err := f.s.authenticate(context.Background(), f.email, nova); err != nil {
		t.Errorf("a senha nova não vale: %v", err)
	}
	// Uso único: o mesmo link de novo não vale mais.
	if _, ok := f.s.usableReset(context.Background(), token); ok {
		t.Error("o link continua utilizável depois de gasto")
	}
}

// ── a senha não vira estado de cliente ───────────────────────────────────────

// O guarda que justifica a decisão de projeto desta superfície: um sinal é
// estado do cliente e o Datastar o serializa em toda requisição seguinte, então
// uma senha em `data-bind` viajaria de novo a cada pedido da página.
func TestTheDoorPutsNothingInADatastarSignal(t *testing.T) {
	f := novaPorta(t)
	for _, caminho := range []string{"/entrar", "/criar-conta?convite=x", "/redefinir-senha"} {
		corpo := f.bate(t, caminho, nil, "").Body.String()
		for _, proibido := range []string{"data-bind", "data-signals", "data-init"} {
			if strings.Contains(corpo, proibido) {
				t.Errorf("%s trouxe %q — a porta não pode ter estado de cliente", caminho, proibido)
			}
		}
	}
}
