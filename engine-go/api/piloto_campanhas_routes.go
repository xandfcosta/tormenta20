package api

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"t20engine/db/sqlcgen"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A rota da cena de CAMPANHAS (ALE-234).
//
// UMA rota serve os dois casos, e é isso que a mantém pequena: a carga fria
// devolve a página inteira, e a busca — que chega pelo mesmo `GET` com os
// sinais do Datastar — devolve só o remendo da cena. Quem distingue os dois é o
// cabeçalho `datastar-request`, que o cliente põe.

func (s *Server) rotasDeCampanhas(r chi.Router) {
	r.Get("/campanhas", s.handleCampanhas)
	r.Get("/campanhas/nova", s.handleCampanhaNova)
	r.Post("/campanhas/nova", s.handleCampanhaNovaPost)
	r.Get("/campanhas/entrar", s.handleCampanhaEntrar)
	r.Post("/campanhas/entrar", s.handleCampanhaEntrarPost)
}

func (s *Server) handleCampanhas(w http.ResponseWriter, r *http.Request) {
	busca, papel := filtroDoPedido(r)
	view, err := s.carregaCampanhas(r.Context(), currentUser(r), busca, papel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Pedido do Datastar: remendo. A cena inteira, e não só a lista, porque a
	// barra também muda — o chip de papel aceso e o texto da busca.
	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDeCampanhas(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.escrevePagina(w, r, http.StatusOK, paginaPiloto{
		Titulo: "Campanhas · Tormenta 20",
		// `cascaNua`: esta cena desenha o próprio cabeçalho, porque ele carrega a
		// busca e os filtros. A casca densa poria um segundo `<h1>` acima.
		Forma: cascaNua,
	}, cenaDeCampanhas(view))
}

// filtroDoPedido lê a busca e o papel, venham de onde vierem.
//
// Da URL na carga fria (`?busca=...`), e dos SINAIS quando o Datastar chama —
// ele os manda no `?datastar=` como JSON. Ler os dois no mesmo lugar é o que
// deixa a tela filtrada ser um endereço que se guarda: recarregar `?busca=anao`
// devolve exatamente o que estava.
func filtroDoPedido(r *http.Request) (busca, papel string) {
	q := r.URL.Query()
	busca, papel = q.Get("busca"), q.Get("papel")
	sinais := struct {
		Busca string `json:"busca"`
		Papel string `json:"papel"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err == nil {
		if sinais.Busca != "" || sinais.Papel != "" {
			busca, papel = sinais.Busca, sinais.Papel
		}
	}
	return busca, papel
}

// urlDeCampanhas monta o endereço que a cena representa, para o histórico do
// navegador acompanhar a busca.
func urlDeCampanhas(busca, papel string) string {
	q := url.Values{}
	if busca != "" {
		q.Set("busca", busca)
	}
	if papel != "" && papel != "todas" {
		q.Set("papel", papel)
	}
	if len(q) == 0 {
		return "/piloto/campanhas"
	}
	return "/piloto/campanhas?" + q.Encode()
}

// ── a folha em branco: abrir campanha (ALE-246) ──────────────────────────────

// handleCampanhaNova desenha o formulário vazio.
func (s *Server) handleCampanhaNova(w http.ResponseWriter, r *http.Request) {
	s.escreveFolhaNova(w, r, http.StatusOK, campanhaNovaView{})
}

// handleCampanhaNovaPost cria e vai para a crônica recém-aberta.
//
// A recusa REDESENHA a folha com o que foi digitado e o erro no campo — não
// redireciona. Redirecionar perderia o texto, e a descrição é o campo caro de
// reescrever. Status 422 e não 200 porque a resposta É uma recusa, e o
// navegador não trata os dois igual no histórico.
func (s *Server) handleCampanhaNovaPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.escreveFolhaNova(w, r, http.StatusBadRequest, campanhaNovaView{Aviso: avisoInterno})
		return
	}
	v := campanhaNovaView{
		Nome:      r.PostFormValue("name"),
		Descricao: r.PostFormValue("description"),
		Erros:     FieldErrorMap{},
	}
	// A MESMA regra da rota JSON, e não uma cópia dela — ver `campanha_regras.go`.
	nome, err := nomeDeCampanha(v.Nome)
	if err != nil {
		v.Erros["name"] = []string{"O nome é obrigatório e cabe em 120 caracteres."}
	}
	descricao, errDesc := descricaoDeCampanha(&v.Descricao)
	if errDesc != nil {
		v.Erros["description"] = []string{"A descrição cabe em 2000 caracteres."}
	}
	if len(v.Erros) > 0 {
		s.escreveFolhaNova(w, r, http.StatusUnprocessableEntity, v)
		return
	}

	agora := nowISO()
	c, err := s.queries.CreateCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: nome, Description: descricao,
		Createdat: agora, Updatedat: agora,
	})
	if err != nil {
		v.Aviso = avisoInterno
		s.escreveFolhaNova(w, r, http.StatusInternalServerError, v)
		return
	}
	// 303 e não 302: depois de um POST, o `See Other` é o que garante que o
	// navegador siga com GET. Sem ele, recarregar a crônica reenviaria o
	// formulário e abriria uma segunda campanha igual.
	http.Redirect(w, r, "/campaigns/"+strconv.FormatInt(c.ID, 10), http.StatusSeeOther)
}

func (s *Server) escreveFolhaNova(w http.ResponseWriter, r *http.Request, status int, v campanhaNovaView) {
	s.escrevePagina(w, r, status, paginaPiloto{
		Titulo: "Abrir nova campanha",
		// `cascaDensa`: a tela da SPA usa o cabeçalho compacto com o "‹ Voltar",
		// e sem ele a folha nasce sem saída visível — o Esc existe, mas atalho
		// não é a única porta.
		Forma:  cascaDensa,
		Voltar: "/piloto/campanhas",
	}, campanhaNova(v))
}

// ── a carta de convite: entrar na mesa (ALE-249) ─────────────────────────────

// handleCampanhaEntrar desenha a carta, com o convite JÁ RESOLVIDO.
func (s *Server) handleCampanhaEntrar(w http.ResponseWriter, r *http.Request) {
	v, err := s.carregaCartaDeConvite(r.Context(), currentUser(r), r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.escreveCarta(w, r, http.StatusOK, v)
}

// handleCampanhaEntrarPost senta o herói à mesa.
//
// As sete travas são do `entrarNaMesa` e não daqui — a mesma função que a rota
// JSON usa. O que este manipulador faz é TRADUZIR cada recusa para uma frase em
// português no campo certo, que é trabalho de tela e não de regra.
func (s *Server) handleCampanhaEntrarPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.escreveCarta(w, r, http.StatusBadRequest, campanhaEntrarView{Aviso: avisoInterno})
		return
	}
	eu := currentUser(r)
	token := r.PostFormValue("token")
	v, err := s.carregaCartaDeConvite(r.Context(), eu, token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	v.NumeroDigitado = r.PostFormValue("campaignId")

	// O id da campanha vem do CONVITE quando há um, e do campo quando não há.
	// Nunca dos dois: com convite na mão, um número digitado seria uma segunda
	// fonte para a mesma coisa, e a tela nem mostra o campo.
	campanhaID := v.CampanhaID
	if !v.TemConvite {
		n, erroNum := strconv.ParseInt(strings.TrimSpace(v.NumeroDigitado), 10, 64)
		if erroNum != nil || n <= 0 {
			v.Erros["campaignId"] = []string{"Informe o número da campanha."}
			s.escreveCarta(w, r, http.StatusUnprocessableEntity, v)
			return
		}
		campanhaID = n
	}

	heroiID, erroHeroi := strconv.ParseInt(r.PostFormValue("characterId"), 10, 64)
	if erroHeroi != nil {
		v.Erros["characterId"] = []string{"Escolha o herói que entra na mesa."}
		s.escreveCarta(w, r, http.StatusUnprocessableEntity, v)
		return
	}
	v.EscolhidoID = heroiID

	_, err = s.entrarNaMesa(r.Context(), pedidoDeEntrada{
		CampanhaID: campanhaID, PersonagemID: heroiID,
		Convite: token, Papel: "player", QuemPede: eu.ID,
	})
	if err != nil {
		v.Erros, v.Aviso = recusaDeEntrada(err)
		s.escreveCarta(w, r, http.StatusUnprocessableEntity, v)
		return
	}
	// 303, como a folha em branco: depois de um POST, recarregar a crônica não
	// pode reenviar o formulário.
	http.Redirect(w, r, "/campaigns/"+strconv.FormatInt(campanhaID, 10), http.StatusSeeOther)
}

// recusaDeEntrada traduz cada erro da regra para a frase que a pessoa lê.
//
// Uma frase por recusa, e não um "não foi possível entrar" para tudo: cada uma
// destas tem uma AÇÃO diferente do outro lado — pedir link novo, conferir o
// número, escolher outro herói, ou nada, porque já está lá dentro.
func recusaDeEntrada(err error) (FieldErrorMap, string) {
	switch {
	case errors.Is(err, errCampanhaInexistente):
		return FieldErrorMap{"campaignId": {"Não existe campanha com esse número."}}, ""
	case errors.Is(err, errConviteExigido):
		return FieldErrorMap{}, "Esta mesa é fechada. Peça um link de convite ao mestre."
	case errors.Is(err, errPersonagemInexistente), errors.Is(err, errPersonagemDeOutro):
		return FieldErrorMap{"characterId": {"Escolha um herói seu."}}, ""
	case errors.Is(err, errJaTemPersonagem):
		return FieldErrorMap{"characterId": {"Você já tem um herói nesta mesa."}}, ""
	case errors.Is(err, errAlreadyInCampaign):
		return FieldErrorMap{"characterId": {"Esse herói já está nesta mesa."}}, ""
	default:
		return FieldErrorMap{}, avisoInterno
	}
}

func (s *Server) escreveCarta(w http.ResponseWriter, r *http.Request, status int, v campanhaEntrarView) {
	if v.Erros == nil {
		v.Erros = FieldErrorMap{}
	}
	s.escrevePagina(w, r, status, paginaPiloto{
		Titulo: "Entrar na mesa",
		Forma:  cascaDensa,
		Voltar: "/piloto/campanhas",
	}, campanhaEntrar(v))
}
