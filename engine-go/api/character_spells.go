package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"
	"t20engine/db"
	"t20engine/db/sqlcgen"
)

// spellRowDTO is the CharacterSpell row Prisma returns from the spell mutations
// (full row incl. characterId; prepared as a bool from the INTEGER column).
type spellRowDTO struct {
	ID             int64  `json:"id"`
	CharacterID    int64  `json:"characterId"`
	CatalogSpellID string `json:"catalogSpellId"`
	Prepared       bool   `json:"prepared"`
	LearnedAt      string `json:"learnedAt"`
}

func spellRowFrom(s sqlcgen.CharacterSpell) spellRowDTO {
	return spellRowDTO{
		ID: s.ID, CharacterID: s.Characterid, CatalogSpellID: s.Catalogspellid,
		Prepared: s.Prepared != 0, LearnedAt: s.Learnedat,
	}
}

// handleLearnSpell adds a spell to the
// grimoire (unprepared). 409 if already known. NOTE: the spell-exists check
// (assertSpellExists) is deferred — the frontend only sends catalog ids.
func (s *Server) handleLearnSpell(w http.ResponseWriter, r *http.Request) {
	character, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	var body struct {
		CatalogSpellID string `json:"catalogSpellId"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.CatalogSpellID == "" {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"catalogSpellId": {"catalogSpellId must be longer than or equal to 1 characters"}})
		return
	}
	row, err := s.queries.CreateSpell(r.Context(), sqlcgen.CreateSpellParams{
		Characterid: character.ID, Catalogspellid: body.CatalogSpellID, Prepared: 0, Learnedat: plataforma.NowISO(),
	})
	if err != nil {
		if db.IsUniqueViolation(err) {
			plataforma.WriteFieldError(w, http.StatusConflict, fmt.Sprintf("Spell %q already known", body.CatalogSpellID), plataforma.FieldErrorMap{"catalogSpellId": {"Already learned"}})
			return
		}
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not learn spell")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, spellRowFrom(row))
}

// handleUnlearnSpell ports unlearnSpell: removes the spell, returning
// {catalogSpellId, removed}. removed=0 when it wasn't known (still 200).
func (s *Server) handleUnlearnSpell(w http.ResponseWriter, r *http.Request) {
	character, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	catalogSpellID := chi.URLParam(r, "catalogSpellId")
	removed, err := s.queries.DeleteSpell(r.Context(), sqlcgen.DeleteSpellParams{
		Characterid: character.ID, Catalogspellid: catalogSpellID,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not unlearn spell")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"catalogSpellId": catalogSpellID, "removed": removed})
}

// handleSetSpellPrepared ports setSpellPrepared: toggles the prepared flag; 404
// (not 400) when the spell isn't learned, so the UI can say "aprenda primeiro".
func (s *Server) handleSetSpellPrepared(w http.ResponseWriter, r *http.Request) {
	character, ok := s.characterFor(w, r)
	if !ok {
		return
	}
	catalogSpellID := chi.URLParam(r, "catalogSpellId")
	var body struct {
		Prepared *bool `json:"prepared"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if body.Prepared == nil {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"prepared": {"prepared must be a boolean value"}})
		return
	}
	row, err := s.queries.SetSpellPreparedByCatalog(r.Context(), sqlcgen.SetSpellPreparedByCatalogParams{
		Prepared: boolToInt(*body.Prepared), CharacterId: character.ID, CatalogSpellId: catalogSpellID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		plataforma.WriteError(w, http.StatusNotFound, fmt.Sprintf("Spell %q not in character's spellbook", catalogSpellID))
		return
	}
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not update spell")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, spellRowFrom(row))
}

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
