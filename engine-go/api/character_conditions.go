package api

import (
	"fmt"
	"net/http"
	"strings"
	"t20engine/plataforma"
	"t20engine/sheet"

	"t20engine/catalog"
	"t20engine/db/sqlcgen"
)

// handleUpdateConditions replace the
// active book conditions after validating every id against the catalog.
func (s *Server) handleUpdateConditions(w http.ResponseWriter, r *http.Request) {
	row, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		ActiveConditions []string `json:"activeConditions"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.ActiveConditions == nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"activeConditions": {"activeConditions must be an array"}})
		return
	}
	var unknown []string
	for _, c := range body.ActiveConditions {
		if !catalog.IsCondition(c) {
			unknown = append(unknown, c)
		}
	}
	if len(unknown) > 0 {
		plataforma.WriteError(w, http.StatusBadRequest, fmt.Sprintf(
			"Unknown condition ids: %s — expected ids from the CONDITIONS catalog", strings.Join(unknown, ", ")))
		return
	}
	activeConditions := sheet.MarshalStrings(&body.ActiveConditions)
	if err := s.queries.UpdateConditions(r.Context(), sqlcgen.UpdateConditionsParams{
		ActiveConditions: activeConditions, UpdatedAt: plataforma.NowISO(), ID: row.ID,
	}); err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update conditions")
		return
	}
	// Avisa a mesa AO VIVO (ALE-245). Sem isto o mestre aplica "Caído" num PC e a
	// tela do jogador não fica sabendo — e como o motor deriva Defesa e perícias
	// da condição (ALE-28), os dois passam a ver números diferentes do mesmo
	// personagem. É a família da ALE-122, e a rota é a única que grava.
	//
	// DEPOIS da escrita, nunca antes: avisar sobre algo que ainda pode falhar
	// faria a mesa buscar o estado velho e acreditar nele.
	s.characterChanged(row.ID)
	plataforma.WriteJSON(w, http.StatusOK, map[string]string{"activeConditions": activeConditions})
}
