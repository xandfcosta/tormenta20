package api

import (
	"errors"
	"net/http"
	"strings"
	"t20engine/plataforma"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"t20engine/db"
)

// As rotas da PORTA (ALE-229). Anônimas — são elas que criam a sessão.
//
// Todas as escritas respondem 303 e não 200: depois de um POST bem-sucedido o
// navegador tem de trocar para GET, senão recarregar a página de destino
// reenvia o formulário (Post/Redirect/Get). Quando a escrita FALHA a resposta é
// a própria tela de novo, com o status honesto (400/401/403), porque aí não há
// nada de novo para onde navegar.

func (s *Server) rotasDaPorta(r chi.Router) {
	r.Get("/entrar", s.handlePortaEntrar)
	r.Post("/entrar", s.handlePortaEntrarSubmit)
	r.Get("/criar-conta", s.handlePortaCriarConta)
	r.Post("/criar-conta", s.handlePortaCriarContaSubmit)
	r.Get("/redefinir-senha", s.handlePortaRedefinir)
	r.Post("/redefinir-senha", s.handlePortaRedefinirSubmit)
}

// ── entrar ───────────────────────────────────────────────────────────────────

func (s *Server) handlePortaEntrar(w http.ResponseWriter, r *http.Request) {
	// Quem já tem sessão não vê a porta. Era o `beforeLoad` da rota `/login`,
	// isto é, autorização morando no cliente; aqui é o handler, e some junto a
	// ida à rede que o guarda fazia para descobrir se havia sessão.
	if destino, autenticado := s.jaEntrou(r); autenticado {
		http.Redirect(w, r, destino, http.StatusSeeOther)
		return
	}
	s.escrevePorta(w, r, http.StatusOK, paginaEntrar(entrarView{
		Destino: destinoPedido(r.URL.Query().Get("redirect")),
	}))
}

func (s *Server) handlePortaEntrarSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	v := entrarView{
		Email:   strings.TrimSpace(r.PostFormValue("email")),
		Destino: destinoPedido(r.PostFormValue("destino")),
	}
	senha := r.PostFormValue("senha")

	// A MESMA validação da API (`validateLogin`), com as chaves traduzidas para
	// os nomes dos campos deste formulário. Uma segunda regra aqui seria uma
	// porta mais frouxa que a outra, e a mais frouxa é a que passa a valer.
	if fields := validateLogin(loginBody{Email: v.Email, Password: senha}); len(fields) > 0 {
		v.Erros = comNomesDoFormulario(fields)
		s.escrevePorta(w, r, http.StatusBadRequest, paginaEntrar(v))
		return
	}
	user, err := s.authenticate(r.Context(), v.Email, senha)
	if err != nil {
		v.Aviso = avisoCredenciais
		s.escrevePorta(w, r, http.StatusUnauthorized, paginaEntrar(v))
		return
	}
	if !s.issueSession(w, user) {
		return
	}
	http.Redirect(w, r, v.Destino, http.StatusSeeOther)
}

// ── criar conta ──────────────────────────────────────────────────────────────

func (s *Server) handlePortaCriarConta(w http.ResponseWriter, r *http.Request) {
	if _, autenticado := s.jaEntrou(r); autenticado {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	convite := r.URL.Query().Get("convite")
	if convite == "" {
		// Sem convite a tela nem abre — era o outro `beforeLoad` (ALE-120). A
		// porta já era fechada (o servidor responde 403), mas a TELA ficava
		// aberta e parecia um cadastro comum. O destino é a de entrar, onde a
		// frase explica que a mesa é por convite.
		http.Redirect(w, r, "/piloto/entrar", http.StatusSeeOther)
		return
	}
	s.escrevePorta(w, r, http.StatusOK, paginaCriarConta(criarContaView{Convite: convite}))
}

func (s *Server) handlePortaCriarContaSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	v := criarContaView{
		Email:   plataforma.NormalizeEmail(r.PostFormValue("email")),
		Nome:    strings.TrimSpace(r.PostFormValue("nome")),
		Convite: r.PostFormValue("convite"),
	}
	senha := r.PostFormValue("senha")
	corpo := registerBody{
		Email: v.Email, Password: senha, InviteToken: v.Convite,
		Name: nomeOuNada(v.Nome),
	}

	v.Erros = comNomesDoFormulario(validateRegister(corpo))
	// A conferência de senha é do FORMULÁRIO e não da API — o `confirmar` não
	// existe no corpo JSON. Ela roda no SERVIDOR e não só no `data-on:input`,
	// senão a página deixaria de proteger contra o typo com JavaScript
	// desligado, que é o que esta superfície ganhou ao não usar sinais.
	if r.PostFormValue("confirmar") != senha {
		v.Erros["confirmar"] = []string{avisoConfere}
	}
	if len(v.Erros) > 0 {
		s.escrevePorta(w, r, http.StatusBadRequest, paginaCriarConta(v))
		return
	}

	user, err := s.createAccount(r.Context(), corpo)
	if err != nil {
		aviso, status := recusaDoRegistro(err)
		v.Aviso = aviso
		s.escrevePorta(w, r, status, paginaCriarConta(v))
		return
	}
	if !s.issueSession(w, user) {
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// recusaDoRegistro traduz o erro do domínio na frase que o jogador lê e no
// status honesto. As frases da API continuam em inglês e continuam sendo as da
// API — quem lê JSON não é quem lê tela.
func recusaDoRegistro(err error) (string, int) {
	switch {
	case db.IsUniqueViolation(err):
		return avisoEmUso, http.StatusConflict
	case errors.Is(err, errInviteRejected), errors.Is(err, errInviteSpent):
		return avisoConvite, http.StatusForbidden
	default:
		return avisoInterno, http.StatusInternalServerError
	}
}

// ── redefinir senha ──────────────────────────────────────────────────────────

func (s *Server) handlePortaRedefinir(w http.ResponseWriter, r *http.Request) {
	s.escrevePorta(w, r, http.StatusOK,
		paginaRedefinir(s.olhaOLink(r, r.URL.Query().Get("token"))))
}

func (s *Server) handlePortaRedefinirSubmit(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "formulário inválido", http.StatusBadRequest)
		return
	}
	senha := r.PostFormValue("senha")
	v := s.olhaOLink(r, r.PostFormValue("token"))
	if !v.LinkVale {
		s.escrevePorta(w, r, http.StatusForbidden, paginaRedefinir(v))
		return
	}

	v.Erros = comNomesDoFormulario(validatePassword(senha))
	if r.PostFormValue("confirmar") != senha {
		v.Erros["confirmar"] = []string{avisoConfere}
	}
	if len(v.Erros) > 0 {
		s.escrevePorta(w, r, http.StatusBadRequest, paginaRedefinir(v))
		return
	}

	if !s.gravaNovaSenha(r, v.Token, senha) {
		// Perder a corrida pelo link é a MESMA resposta de link inválido: quem
		// chegou depois não pode saber que houve um primeiro.
		v.LinkVale = false
		s.escrevePorta(w, r, http.StatusForbidden, paginaRedefinir(v))
		return
	}
	// Sem sessão: quem redefiniu a senha entra com ela. Emitir cookie aqui
	// transformaria um link de recuperação num login, e o link chega por um
	// canal que ninguém controla.
	http.Redirect(w, r, "/piloto/entrar", http.StatusSeeOther)
}

// olhaOLink pergunta pelo link ANTES de o formulário existir. Um link vencido
// dizer isso de cara é melhor que falhar no envio com a senha já digitada duas
// vezes.
func (s *Server) olhaOLink(r *http.Request, token string) redefinirView {
	v := redefinirView{Token: token, Erros: plataforma.FieldErrorMap{}}
	reset, ok := s.usableReset(r.Context(), token)
	if !ok {
		return v
	}
	user, err := s.queries.GetUserByID(r.Context(), reset.Userid)
	if err != nil {
		return v
	}
	v.LinkVale = true
	v.EmailDaConta = user.Email
	return v
}

func (s *Server) gravaNovaSenha(r *http.Request, token, senha string) bool {
	reset, ok := s.usableReset(r.Context(), token)
	if !ok {
		return false
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcryptCost)
	if err != nil {
		return false
	}
	return s.applyReset(r.Context(), reset, string(hash)) == nil
}

// ── auxiliares da porta ──────────────────────────────────────────────────────

// escrevePorta desenha uma tela da porta com o status que a resposta merece.
//
// O status importa: um formulário recusado com 200 mente para tudo o que não é
// um navegador — teste, log, monitoração —, e a tela é a mesma nos dois casos.
func (s *Server) escrevePorta(
	w http.ResponseWriter, r *http.Request, status int, corpo templ.Component,
) {
	s.escrevePagina(w, r, status, paginaPiloto{
		// O `<title>` é o do JOGO e não o da tela: a porta é a tela-título, e o
		// nome dela já está desenhado em Cinzel no meio da página.
		Titulo: "Tormenta 20",
		Forma:  cascaTitulo,
		Kicker: "— Grimório de Arton —",
		// Sem `Sinais` e sem `Init`: esta superfície não tem estado de cliente
		// nenhum, e é isso que mantém a senha fora dele. O campo abaixo DIZ isso
		// para a casca, que de outro modo acrescentaria o `data-init` da
		// restauração de foco do trilho a toda página — a porta não tem trilho, e
		// o guarda desta regra existe justamente porque a omissão é silenciosa.
		SemEstadoDeCliente: true,
	}, corpo)
}

// jaEntrou responde se o pedido já traz uma sessão válida, e para onde mandar
// quem tem uma.
func (s *Server) jaEntrou(r *http.Request) (string, bool) {
	if _, err := s.sessionUser(r); err != nil {
		return "", false
	}
	return destinoPedido(r.URL.Query().Get("redirect")), true
}

// destinoPedido só aceita caminho INTERNO. Um `?redirect=` que aceitasse
// `https://outro.site` transformaria a porta em redirecionamento aberto: o link
// sai do nosso domínio, o jogador confia, e a página que recebe pode imitar
// esta. Barra dupla é o caso que engana — `//outro.site` é protocol-relative e
// o navegador o trata como absoluto.
func destinoPedido(bruto string) string {
	if bruto == "" || !strings.HasPrefix(bruto, "/") || strings.HasPrefix(bruto, "//") {
		return "/"
	}
	return bruto
}

// comNomesDoFormulario traduz as chaves do `plataforma.FieldErrorMap` da API (`password`)
// para os nomes dos campos DESTE formulário (`senha`).
//
// A tradução é aqui e não no validador porque o `plataforma.FieldErrorMap` é contrato de
// fio da API JSON — renomear a chave lá quebraria o cliente que a lê.
func comNomesDoFormulario(fields plataforma.FieldErrorMap) plataforma.FieldErrorMap {
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

// nomeOuNada: nome vazio é "sem nome", não a string vazia.
func nomeOuNada(nome string) *string {
	if nome == "" {
		return nil
	}
	return &nome
}
