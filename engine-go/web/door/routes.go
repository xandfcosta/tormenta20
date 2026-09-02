package door

import (
	"net/http"
	"strings"
	"t20engine/account"
	"t20engine/plataforma"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"

	"t20engine/web/ui"
)

// As rotas da PORTA (ALE-229). Anônimas — são elas que criam a sessão.
//
// Todas as escritas respondem 303 e não 200: depois de um POST bem-sucedido o
// navegador tem de trocar para GET, senão recarregar a página de destino
// reenvia o formulário (Post/Redirect/Get). Quando a escrita FALHA a resposta é
// a própria tela de novo, com o status honesto (400/401/403), porque aí não há
// nada de novo para onde navegar.

// Routes monta a porta no roteador de quem a hospeda.
//
// Os endereços moram AQUI e não em quem monta (ALE-278): a cena é a dona do que
// ela atende, e quem a hospeda escolhe só onde encaixá-la.
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
	// Quem já tem sessão não vê a porta. Era o `beforeLoad` da rota `/login`,
	// isto é, autorização morando no cliente; aqui é o handler, e some junto a
	// ida à rede que o guarda fazia para descobrir se havia sessão.
	if destino, autenticado := s.alreadySignedIn(r); autenticado {
		http.Redirect(w, r, destino, http.StatusSeeOther)
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
	senha := r.PostFormValue("senha")

	// A MESMA validação da API (`validateLogin`), com as chaves traduzidas para
	// os nomes dos campos deste formulário. Uma segunda regra aqui seria uma
	// porta mais frouxa que a outra, e a mais frouxa é a que passa a valer.
	if fields := account.ValidateLogin(account.LoginBody{Email: v.Email, Password: senha}); len(fields) > 0 {
		v.Errors = withFormFieldNames(fields)
		s.writeDoor(w, r, http.StatusBadRequest, signInPage(v))
		return
	}
	user, err := s.deps.Authenticate(r.Context(), v.Email, senha)
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
	if _, autenticado := s.alreadySignedIn(r); autenticado {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	convite := r.URL.Query().Get("convite")
	if convite == "" {
		// Sem convite a tela nem abre — era o outro `beforeLoad` (ALE-120). A
		// porta já era fechada (o servidor responde 403), mas a TELA ficava
		// aberta e parecia um cadastro comum. O destino é a de entrar, onde a
		// frase explica que a mesa é por convite.
		http.Redirect(w, r, "/entrar", http.StatusSeeOther)
		return
	}
	s.writeDoor(w, r, http.StatusOK, signUpPage(signUpView{Invite: convite}))
}

func (s Scene) handleSignUpSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	v := signUpView{
		Email:  plataforma.NormalizeEmail(r.PostFormValue("email")),
		Name:   strings.TrimSpace(r.PostFormValue("nome")),
		Invite: r.PostFormValue("convite"),
	}
	senha := r.PostFormValue("senha")
	corpo := account.RegisterBody{
		Email: v.Email, Password: senha, InviteToken: v.Invite,
		Name: nameOrNil(v.Name),
	}

	v.Errors = withFormFieldNames(account.ValidateRegister(corpo))
	// A conferência de senha é do FORMULÁRIO e não da API — o `confirmar` não
	// existe no corpo JSON. Ela roda no SERVIDOR e não só no `data-on:input`,
	// senão a página deixaria de proteger contra o typo com JavaScript
	// desligado, que é o que esta superfície ganhou ao não usar sinais.
	if r.PostFormValue("confirmar") != senha {
		v.Errors["confirmar"] = []string{noticePasswordMismatch}
	}
	if len(v.Errors) > 0 {
		s.writeDoor(w, r, http.StatusBadRequest, signUpPage(v))
		return
	}

	user, err := s.deps.CreateAccount(r.Context(), corpo)
	if err != nil {
		aviso, status := s.signUpRefusal(err)
		v.Notice = aviso
		s.writeDoor(w, r, status, signUpPage(v))
		return
	}
	if !s.deps.IssueSession(w, user) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// signUpRefusal traduz o erro do domínio na frase que o jogador lê e no
// status honesto. As frases da API continuam em inglês e continuam sendo as da
// API — quem lê JSON não é quem lê tela.
// signUpRefusal escolhe a FRASE; quem classifica o erro é o hospedeiro.
//
// Ela lia os sentinelas `errInviteRejected` e `errInviteSpent` direto do `api`,
// e é justamente esse tipo de alcance que a divisão existe para cortar. A
// repartição ficou assim: o hospedeiro sabe distinguir os erros dele e devolve
// um MOTIVO; a cena sabe o que o jogador lê. Nenhum dos dois faz o trabalho do
// outro, e a voz da porta não vai morar no `api`.
func (s Scene) signUpRefusal(err error) (string, int) {
	motivo, status := s.deps.SignUpRefusal(err)
	switch motivo {
	case RefusalEmailTaken:
		return noticeEmailTaken, status
	case RefusalBadInvite:
		return noticeBadInvite, status
	default:
		return ui.NoticeInternal, status
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
	senha := r.PostFormValue("senha")
	v := s.linkView(r, r.PostFormValue("token"))
	if !v.LinkIsValid {
		s.writeDoor(w, r, http.StatusForbidden, resetPage(v))
		return
	}

	v.Errors = withFormFieldNames(account.ValidatePassword(senha))
	if r.PostFormValue("confirmar") != senha {
		v.Errors["confirmar"] = []string{noticePasswordMismatch}
	}
	if len(v.Errors) > 0 {
		s.writeDoor(w, r, http.StatusBadRequest, resetPage(v))
		return
	}

	if !s.saveNewPassword(r, v.Token, senha) {
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
	v := resetView{Token: token, Errors: plataforma.FieldErrorMap{}}
	email, ok := s.deps.ResetLinkOwner(r.Context(), token)
	if !ok {
		return v
	}
	v.LinkIsValid = true
	v.AccountEmail = email
	return v
}

// saveNewPassword pede o caminho INTEIRO ao hospedeiro, de propósito.
//
// Ela gerava o hash aqui, com o `bcryptCost` do `api`. Isso obrigaria a porta a
// carregar uma constante de custo criptográfico para a cena fazer trabalho que
// não é dela — e o custo do bcrypt é decisão de segurança do servidor, não de
// quem desenha o formulário.
func (s Scene) saveNewPassword(r *http.Request, token, senha string) bool {
	return s.deps.ResetPassword(r.Context(), token, senha)
}

// ── auxiliares da porta ──────────────────────────────────────────────────────

// writeDoor desenha uma tela da porta com o status que a resposta merece.
//
// O status importa: um formulário recusado com 200 mente para tudo o que não é
// um navegador — teste, log, monitoração —, e a tela é a mesma nos dois casos.
func (s Scene) writeDoor(
	w http.ResponseWriter, r *http.Request, status int, corpo templ.Component,
) {
	s.deps.WritePage(w, r, status, ui.Page{
		// O `<title>` é o do JOGO e não o da tela: a porta é a tela-título, e o
		// nome dela já está desenhado em Cinzel no meio da página.
		Titulo: "Tormenta 20",
		Forma:  ui.ShellTitled,
		Kicker: "— Grimório de Arton —",
		// Sem `Sinais` e sem `Init`: esta superfície não tem estado de cliente
		// nenhum, e é isso que mantém a senha fora dele. O campo abaixo DIZ isso
		// para a casca, que de outro modo acrescentaria o `data-init` da
		// restauração de foco do trilho a toda página — a porta não tem trilho, e
		// o guarda desta regra existe justamente porque a omissão é silenciosa.
		SemEstadoDeCliente: true,
	}, corpo)
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
func requestedDestination(bruto string) string {
	if bruto == "" || !strings.HasPrefix(bruto, "/") || strings.HasPrefix(bruto, "//") {
		return "/"
	}
	return bruto
}

// withFormFieldNames traduz as chaves do `plataforma.FieldErrorMap` da API (`password`)
// para os nomes dos campos DESTE formulário (`senha`).
//
// A tradução é aqui e não no validador porque o `plataforma.FieldErrorMap` é contrato de
// fio da API JSON — renomear a chave lá quebraria o cliente que a lê.
func withFormFieldNames(fields plataforma.FieldErrorMap) plataforma.FieldErrorMap {
	out := plataforma.FieldErrorMap{}
	nomes := map[string]string{"password": "senha", "name": "nome", "email": "email"}
	for chave, msgs := range fields {
		if nome, ok := nomes[chave]; ok {
			out[nome] = msgs
			continue
		}
		out[chave] = msgs
	}
	return out
}

// nameOrNil: nome vazio é "sem nome", não a string vazia.
func nameOrNil(nome string) *string {
	if nome == "" {
		return nil
	}
	return &nome
}
