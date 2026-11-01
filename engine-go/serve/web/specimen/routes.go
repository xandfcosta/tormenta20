package specimen

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"t20engine/serve/web/ui"
)

// Routes monta a folha viva no roteador de quem a hospeda.
func Routes(r chi.Router, s Scene) {
	r.Get("/sistema", s.handleSpecimen)
}

func (s Scene) handleSpecimen(w http.ResponseWriter, r *http.Request) {
	s.deps.WritePage(w, r, http.StatusOK, ui.Page{
		Title:  "Sistema",
		Shape:  ui.ShellBare,
		System: ui.SystemTempestade,
	}, specimen(ui.ColorModeOf(r)))
}
