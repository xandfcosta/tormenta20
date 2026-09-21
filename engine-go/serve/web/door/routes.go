package door

import (
	"errors"
	"net/http"
	"strings"

	"t20engine/app/accounts"
	"t20engine/domain/account"
	"t20engine/infra/wire"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"t20engine/serve/web/ui"
)

// As rotas da PORTA. Anônimas — são elas que criam a sessão.
//
// Todas as escritas respondem 303 e não 200: depois de um POST bem-sucedido o
// navegador tem de trocar para GET, senão recarregar a página de destino
// reenvia o formulário (Post/Redirect/Get). Quando a escrita FALHA a resposta é
// a própria tela de novo, com o status honesto (400/401/403), porque aí não há
// nada de novo para onde navegar.

// Routes monta a porta no roteador de quem a hospeda.
//
// Os endereços moram AQUI e não em quem monta: a cena é a dona do que ela
// atende, e quem a hospeda escolhe só onde encaixá-la.
func Routes(r chi.Router, s Scene) {
	r.Get("/entrar", s.handleSignIn)
	r.Post("/entrar", s.handleSignInSubmit)
	r.Get("/criar-conta", s.handleSignUp)
	r.Post("/criar-conta", s.handleSignUpSubmit)
	r.Get("/redefinir-senha", s.handleReset)
	r.Post("/redefinir-senha", s.handleResetSubmit)
}

// ── entrar ───────────────────────────────────────────────────────────────────

func (s Scene) handleSignIn(w http.ResponseWriter, r *http.Request) {
	// Quem já tem sessão não vê a porta, e quem decide isso é o HANDLER: uma
	// guarda no cliente custaria uma ida à rede só para descobrir se há sessão.
	if destination, authenticated := s.alreadySignedIn(r); authenticated {
		http.Redirect(w, r, destination, http.StatusSeeOther)
		return
	}
	s.writeDoor(w, r, http.StatusOK, signInPage(signInView{
		Destination: requestedDestination(r.URL.Query().Get("redirect")),
	}))
}

func (s Scene) handleSignInSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	v := signInView{
		Email:       strings.TrimSpace(r.PostFormValue("email")),
		Destination: requestedDestination(r.PostFormValue("destino")),
	}
	password := r.PostFormValue("senha")

	// A MESMA validação da API (`validateLogin`), com as chaves traduzidas para
	// os nomes dos campos deste formulário. Uma segunda regra aqui seria uma
	// porta mais frouxa que a outra, e a mais frouxa é a que passa a valer.
	if fields := account.ValidateLogin(account.LoginBody{Email: v.Email, Password: password}); len(fields) > 0 {
		v.Errors = withFormFieldNames(fields)
		s.writeDoor(w, r, http.StatusBadRequest, signInPage(v))
		return
	}
	user, err := s.gate.Authenticate(r.Context(), v.Email, password)
	if err != nil {
		v.Notice = noticeBadCredentials
		s.writeDoor(w, r, http.StatusUnauthorized, signInPage(v))
		return
	}
	if !s.deps.IssueSession(w, user) {
		return
	}
	http.Redirect(w, r, v.Destination, http.StatusSeeOther)
}

// ── criar conta ──────────────────────────────────────────────────────────────

func (s Scene) handleSignUp(w http.ResponseWriter, r *http.Request) {
	if _, authenticated := s.alreadySignedIn(r); authenticated {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	invite := r.URL.Query().Get("convite")
	if invite == "" {
		// Sem convite a tela nem abre. O servidor já recusa com 403, mas uma tela
		// de cadastro aberta parece um cadastro comum — o destino é a de entrar,
		// onde a frase explica que a mesa é por convite.
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	s.writeDoor(w, r, http.StatusOK, signUpPage(signUpView{Invite: invite}))
}

func (s Scene) handleSignUpSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	v := signUpView{
		Email:  wire.NormalizeEmail(r.PostFormValue("email")),
		Name:   strings.TrimSpace(r.PostFormValue("nome")),
		Invite: r.PostFormValue("convite"),
	}
	password := r.PostFormValue("senha")
	body := account.RegisterBody{
		Email: v.Email, Password: password, InviteToken: v.Invite,
		Name: nameOrNil(v.Name),
	}

	v.Errors = withFormFieldNames(account.ValidateRegister(body))
	// A conferência de senha é do FORMULÁRIO e não da API — o `confirmar` não
	// existe no corpo JSON. Ela roda no SERVIDOR e não só no `data-on:input`,
	// senão a página deixaria de proteger contra o typo com JavaScript
	// desligado, que é o que esta superfície ganhou ao não usar sinais.
	if r.PostFormValue("confirmar") != password {
		v.Errors["confirmar"] = []string{noticePasswordMismatch}
	}
	if len(v.Errors) > 0 {
		s.writeDoor(w, r, http.StatusBadRequest, signUpPage(v))
		return
	}

	user, err := s.gate.Register(r.Context(), body)
	if err != nil {
		notice, status := s.signUpRefusal(err)
		v.Notice = notice
		s.writeDoor(w, r, status, signUpPage(v))
		return
	}
	if !s.deps.IssueSession(w, user) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// signUpRefusal escolhe a FRASE que o jogador lê e o STATUS da resposta.
//
// As duas escolhas são da CENA, e agora sem intermediário: as recusas são
// valores EXPORTADOS do caso de uso (`accounts.ErrEmailTaken`,
// `accounts.ErrBadInvite`), e `errors.Is` as lê daqui.
//
// Havia um vocabulário só para atravessar a fronteira — um tipo de MOTIVO
// declarado nesta cena, que o hospedeiro devolvia porque os sentinelas eram
// dele e a cena não podia alcançá-los. Com a regra no `app/`, o tipo do meio não
// tem mais o que traduzir e não existe mais (ALE-349).
func (s Scene) signUpRefusal(err error) (string, int) {
	switch {
	case errors.Is(err, accounts.ErrEmailTaken):
		return noticeEmailTaken, http.StatusConflict
	case errors.Is(err, accounts.ErrBadInvite):
		return noticeBadInvite, http.StatusForbidden
	default:
		return ui.NoticeInternal, http.StatusInternalServerError
	}
}

// ── redefinir senha ──────────────────────────────────────────────────────────

func (s Scene) handleReset(w http.ResponseWriter, r *http.Request) {
	s.writeDoor(w, r, http.StatusOK,
		resetPage(s.linkView(r, r.URL.Query().Get("token"))))
}

func (s Scene) handleResetSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	password := r.PostFormValue("senha")
	v := s.linkView(r, r.PostFormValue("token"))
	if !v.LinkIsValid {
		s.writeDoor(w, r, http.StatusForbidden, resetPage(v))
		return
	}

	v.Errors = withFormFieldNames(account.ValidatePassword(password))
	if r.PostFormValue("confirmar") != password {
		v.Errors["confirmar"] = []string{noticePasswordMismatch}
	}
	if len(v.Errors) > 0 {
		s.writeDoor(w, r, http.StatusBadRequest, resetPage(v))
		return
	}

	if !s.saveNewPassword(r, v.Token, password) {
		// Perder a corrida pelo link é a MESMA resposta de link inválido: quem
		// chegou depois não pode saber que houve um primeiro.
		v.LinkIsValid = false
		s.writeDoor(w, r, http.StatusForbidden, resetPage(v))
		return
	}
	// Sem sessão: quem redefiniu a senha entra com ela. Emitir cookie aqui
	// transformaria um link de recuperação num login, e o link chega por um
	// canal que ninguém controla.
	http.Redirect(w, r, "/entrar", http.StatusSeeOther)
}

// linkView pergunta pelo link ANTES de o formulário existir. Um link vencido
// dizer isso de cara é melhor que falhar no envio com a senha já digitada duas
// vezes.
func (s Scene) linkView(r *http.Request, token string) resetView {
	v := resetView{Token: token, Errors: wire.FieldErrorMap{}}
	email, err := s.resets.OwnerOfLink(r.Context(), token)
	if err != nil {
		return v
	}
	v.LinkIsValid = true
	v.AccountEmail = email
	return v
}

// saveNewPassword pede o caminho INTEIRO ao caso de uso, de propósito: gerar o
// hash aqui obrigaria esta cena a carregar a constante de custo do bcrypt, que é
// decisão de segurança do servidor e não de quem desenha o formulário.
func (s Scene) saveNewPassword(r *http.Request, token, password string) bool {
	return s.resets.Apply(r.Context(), token, password) == nil
}

// ── auxiliares da porta ──────────────────────────────────────────────────────

// writeDoor desenha uma tela da porta com o status que a resposta merece.
//
// O status importa: um formulário recusado com 200 mente para tudo o que não é
// um navegador — teste, log, monitoração —, e a tela é a mesma nos dois casos.
func (s Scene) writeDoor(
	w http.ResponseWriter, r *http.Request, status int, body templ.Component,
) {
	s.deps.WritePage(w, r, status, ui.Page{
		// O `<title>` é o do JOGO e não o da tela: a porta é a tela-título, e o
		// nome dela já está desenhado em Cinzel no meio da página.
		Title:  "Tormenta 20",
		Forma:  ui.ShellTitled,
		Kicker: "— Grimório de Arton —",
		// Sem `Sinais` e sem `Init`: esta superfície não tem estado de cliente
		// nenhum, e é isso que mantém a senha fora dele. O campo abaixo DIZ isso
		// para a casca, que de outro modo acrescentaria o `data-init` da
		// restauração de foco do trilho a toda página — a porta não tem trilho, e
		// o guarda desta regra existe justamente porque a omissão é silenciosa.
		SemEstadoDeCliente: true,
	}, body)
}

// alreadySignedIn responde se o pedido já traz uma sessão válida, e para onde mandar
// quem tem uma.
func (s Scene) alreadySignedIn(r *http.Request) (string, bool) {
	if !s.deps.HasSession(r) {
		return "", false
	}
	return requestedDestination(r.URL.Query().Get("redirect")), true
}

// requestedDestination só aceita caminho INTERNO. Um `?redirect=` que aceitasse
// `https://outro.site` transformaria a porta em redirecionamento aberto: o link
// sai do nosso domínio, o jogador confia, e a página que recebe pode imitar
// esta. Barra dupla é o caso que engana — `//outro.site` é protocol-relative e
// o navegador o trata como absoluto.
func requestedDestination(raw string) string {
	if raw == "" || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return "/"
	}
	return raw
}

// withFormFieldNames traduz as chaves do `wire.FieldErrorMap` da API (`password`)
// para os nomes dos campos DESTE formulário (`password`).
//
// A tradução é aqui e não no validador porque o `wire.FieldErrorMap` é contrato de
// fio da API JSON — renomear a chave lá quebraria o cliente que a lê.
func withFormFieldNames(fields wire.FieldErrorMap) wire.FieldErrorMap {
	out := wire.FieldErrorMap{}
	names := map[string]string{"password": "senha", "name": "nome", "email": "email"}
	for key, msgs := range fields {
		if name, ok := names[key]; ok {
			out[name] = msgs
			continue
		}
		out[key] = msgs
	}
	return out
}

// nameOrNil: nome vazio é "sem nome", não a string vazia.
func nameOrNil(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}
