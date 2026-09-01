package api

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

func (s *Server) GrimoireRoutes(r chi.Router) {
	r.Get("/grimorio", s.handleGrimorio)
}

func (s *Server) handleGrimorio(w http.ResponseWriter, r *http.Request) {
	s.WritePage(w, r, http.StatusOK, ui.Page{
		Titulo:        "Grimório",
		Forma:         ui.ShellDense,
		TituloVisivel: "Grimório",
		Voltar:        "/",
		VoltarRotulo:  "Hub",
		Scripts: []string{
			EstaticoDoPiloto("grimorio.js"),
			EstaticoDoPiloto("pecas-solid.js"),
		},
	}, grimorio())
}
