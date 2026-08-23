package api

import (
	"net/http"
	"net/url"
	"strconv"

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
