package api

import (
	"net/http"
	"net/url"

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
