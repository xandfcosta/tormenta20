package characters

import (
	"net/http"

	"t20engine/web/ui"

	"github.com/go-chi/chi/v5"
	"github.com/starfederation/datastar-go/datastar"
)

// A rota da cena de PERSONAGENS (ALE-239). Mesma forma da de campanhas: uma
// rota serve a carga fria e o remendo da busca, e quem distingue é o cabeçalho
// `datastar-request`.

// Routes registra a rota da cena. A FORJA saiu daqui na ALE-278 e subiu para o
// roteador do `api`: montá-la aqui era organização e não dependência — o `chi`
// não liga para quem registra o quê, e uma cena montando outra faria esta
// importar a forja para sempre.
func Routes(r chi.Router, s Scene) {
	r.Get("/personagens", s.handleCharacters)
}

func (s Scene) handleCharacters(w http.ResponseWriter, r *http.Request) {
	view, err := s.Load(r.Context(), s.deps.CurrentUserID(r), termFromRequest(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("datastar-request") != "" {
		sse := datastar.NewSSE(w, r)
		fragmento, err := ui.RenderFragment(r.Context(), SceneBody(view))
		if err != nil {
			return
		}
		_ = sse.PatchElements(fragmento)
		return
	}

	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo: "Personagens · Tormenta 20",
		// `ui.ShellBare`: a cena desenha o próprio cabeçalho, que carrega a busca.
		Forma: ui.ShellBare,
	}, SceneBody(view))
}

// termFromRequest lê a busca da URL na carga fria e dos SINAIS quando o Datastar
// chama. Mesma razão do `filtroDoPedido` das campanhas: ler os dois no mesmo
// lugar é o que deixa `?busca=anao` ser um endereço que se recarrega.
func termFromRequest(r *http.Request) string {
	busca := r.URL.Query().Get("busca")
	sinais := struct {
		Busca string `json:"busca"`
	}{}
	if err := datastar.ReadSignals(r, &sinais); err == nil && sinais.Busca != "" {
		busca = sinais.Busca
	}
	return busca
}
