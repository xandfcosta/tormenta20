package grimoire

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"t20engine/web/ui"
)

// A rota da FOLHA DE ESPECIFICAÇÃO (ALE-251).
//
// Ela é a única cena do piloto que pede um script PRÓPRIO — o `grimorio.js`,
// que mede. Ele não entra no `layout` de todas as páginas de propósito: canvas
// e cálculo de contraste não têm o que fazer no caminho de quem só quer jogar.

// Routes monta o grimório no roteador de quem o hospeda.
//
// O endereço mora AQUI (ALE-278): a cena é a dona do que ela atende.
func Routes(r chi.Router, s Scene) {
	r.Get("/grimorio", s.handleGrimoire)
}

func (s Scene) handleGrimoire(w http.ResponseWriter, r *http.Request) {
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:        "Grimório",
		Forma:         ui.ShellDense,
		TituloVisivel: "Grimório",
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		Scripts: []string{
			s.deps.Asset("grimorio.js"),
			s.deps.Asset("pecas-solid.js"),
		},
	}, grimoire())
}
