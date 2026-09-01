package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"t20engine/web/ui"
)

// A rota da cena de PERSONAGENS (ALE-239). Mesma forma da de campanhas: uma
// rota serve a carga fria e o remendo da busca, e quem distingue é o cabeçalho
// `datastar-request`.

func (s *Server) rotasDePersonagens(r chi.Router) {
	r.Get("/personagens", s.handlePersonagens)
	// A forja mora aqui e não numa árvore própria porque ela é a porta de
	// entrada DESTA cena: o elenco é de onde se abre a folha em branco.
	r.Get("/personagens/nova", s.handleForja)
	r.Post("/personagens/nova", s.handleForjaPost)
	r.Post("/personagens/nova/esboco", s.handleForjaEsboco)
	// A segunda cena da forja. Ela vive sob o id porque o herói JÁ existe: o
	// nascimento é o `POST /personagens/nova`, e daqui em diante tudo é comando
	// sobre uma linha do banco.
	r.Get("/personagens/{id}/atributos", s.handleForjaAtributos)
	r.Post("/personagens/{id}/atributos/{atributo}/{passo}", s.handleForjaAtributoPasso)
}

func (s *Server) handlePersonagens(w http.ResponseWriter, r *http.Request) {
	view, err := s.carregaPersonagens(r.Context(), currentUser(r), buscaDoPedido(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := renderFragmento(r.Context(), cenaDePersonagens(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.escrevePagina(w, r, http.StatusOK, ui.Page{
		Titulo: "Personagens · Tormenta 20",
		// `cascaNua`: a cena desenha o próprio cabeçalho, que carrega a busca.
		Forma: ui.ShellBare,
	}, cenaDePersonagens(view))
}

// buscaDoPedido lê a busca da URL na carga fria e dos SINAIS quando o Datastar
// chama. Mesma razão do `filtroDoPedido` das campanhas: ler os dois no mesmo
// lugar é o que deixa `?busca=anao` ser um endereço que se recarrega.
func buscaDoPedido(r *http.Request) string {
	busca := r.URL.Query().Get("busca")
	sinais := struct {
		Busca string `json:"busca"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err == nil && sinais.Busca != "" {
		busca = sinais.Busca
	}
	return busca
}
