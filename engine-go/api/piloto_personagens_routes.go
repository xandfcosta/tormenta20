package api

import (
	"net/http"
	"t20engine/web/forge"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
	"t20engine/web/ui"
)

// A rota da cena de PERSONAGENS (ALE-239). Mesma forma da de campanhas: uma
// rota serve a carga fria e o remendo da busca, e quem distingue é o cabeçalho
// `datastar-request`.

func (s *Server) CharacterRoutes(r chi.Router) {
	r.Get("/personagens", s.handlePersonagens)
	// A FORJA mora noutro PACOTE desde a ALE-278, e é montada aqui porque este é
	// o endereço dela: o elenco é de onde se abre a folha em branco.
	//
	// `forge.New(s)` passa o `*Server` como a porta que a cena declarou. Quem
	// decide o que ela pode usar é ela — a interface está em `web/forge`, e o
	// `Server` a cumpre. No dia em que ele deixar de cumprir, o erro aparece
	// AQUI, na linha que monta, e não espalhado pela cena.
	forge.Routes(r, forge.New(s))
}

func (s *Server) handlePersonagens(w http.ResponseWriter, r *http.Request) {
	view, err := s.carregaPersonagens(r.Context(), currentUser(r), buscaDoPedido(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), cenaDePersonagens(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.WritePage(w, r, http.StatusOK, ui.Page{
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
