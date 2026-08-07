package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"t20engine/db/sqlcgen"
)

// handleListCharacters returns the caller's own characters (newest-updated first),
// each as the full aggregate — mirrors CharactersService.list.
func (s *Server) handleListCharacters(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	rows, err := s.queries.ListCharactersByOwner(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list characters")
		return
	}
	out := make([]CharacterDTO, 0, len(rows))
	for _, row := range rows {
		dto, err := s.loadCharacter(r.Context(), row)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Could not load character")
			return
		}
		out = append(out, dto)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGetCharacter returns one character aggregate. Access = owner OR campaign
// GM (CharactersService.findOne). 404 when missing, 403 when unauthorized.
func (s *Server) handleGetCharacter(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	writeJSON(w, http.StatusOK, dto)
}

// authorizedCharacter loads a character and enforces the read/mutation guard
// (owner or campaign GM). Returns the row, or an HTTP status + error to emit.
func (s *Server) authorizedCharacter(ctx context.Context, user AuthUser, id int64) (sqlcgen.Character, int, error) {
	row, err := s.queries.GetCharacter(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return row, http.StatusNotFound, fmt.Errorf("Character %d not found", id)
	}
	if err != nil {
		return row, http.StatusInternalServerError, errors.New("Could not load character")
	}
	if row.Ownerid == user.ID {
		return row, http.StatusOK, nil
	}
	isGm, err := s.queries.IsCampaignGmForCharacter(ctx, sqlcgen.IsCampaignGmForCharacterParams{
		Characterid: id,
		Ownerid:     user.ID,
	})
	if err != nil {
		return row, http.StatusInternalServerError, errors.New("Could not check access")
	}
	if !isGm {
		return row, http.StatusForbidden, fmt.Errorf("Character %d belongs to another user", id)
	}
	return row, http.StatusOK, nil
}

// handleGetSheet ports GET /characters/:id/sheet — the server-computed derived sheet
// (ComputedSheetV2), the same shape the WASM engine produces for the front. No live
// consumer today (the front derives via WASM); provided for non-WASM clients + parity.
// Active conditionals (stances) aren't applied — this is the base sheet.
func (s *Server) handleGetSheet(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	row, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	if s.catalogs == nil {
		writeError(w, http.StatusServiceUnavailable, "Rules catalog not loaded")
		return
	}
	dto, err := s.loadCharacter(r.Context(), row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	ec, err := engineCharacterFrom(dto)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not build sheet input")
		return
	}
	writeJSON(w, http.StatusOK, s.catalogs.ComputeSheetV2(ec, map[string]bool{}))
}

// assertCharacterOwner is the strict owner-only check (mirrors CharactersService.assertOwner)
// the WS vitals gate uses: a player may edit only a character they own. Transport-agnostic.
func (s *Server) assertCharacterOwner(ctx context.Context, userID, characterID int64) (int, error) {
	owner, err := s.queries.GetCharacterOwner(ctx, characterID)
	if errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, fmt.Errorf("Character %d not found", characterID)
	}
	if err != nil {
		return http.StatusInternalServerError, errors.New("Could not load character")
	}
	if owner != userID {
		return http.StatusForbidden, fmt.Errorf(
			"Caller %d can only edit their own character's vitals (character %d)", userID, characterID)
	}
	return http.StatusOK, nil
}

// loadCharacter attaches the six relations to a character row in the Prisma
// include order (races/classes/items/effects by id, expertises by name, spells by
// learnedAt).
func (s *Server) loadCharacter(ctx context.Context, c sqlcgen.Character) (CharacterDTO, error) {
	dto := characterScalarsFrom(c)

	races, err := s.queries.ListRacesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, race := range races {
		dto.Races = append(dto.Races, RaceDTO{Race: race})
	}

	classes, err := s.queries.ListClassesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, cl := range classes {
		dto.Classes = append(dto.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
	}

	exps, err := s.queries.ListExpertisesByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, e := range exps {
		dto.Expertises = append(dto.Expertises, ExpertiseDTO{
			Name: e.Name, Attribute: e.Attribute, Trained: e.Trained != 0, Custom: e.Custom != 0,
		})
	}

	items, err := s.queries.ListItemsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, it := range items {
		dto.Items = append(dto.Items, ItemDTO{
			ID: it.ID, CatalogID: nullToPtr(it.Catalogid), Name: it.Name,
			Quantity: it.Quantity, Slots: it.Slots, Equipped: nullToPtr(it.Equipped),
			Improvements: it.Improvements, Material: nullToPtr(it.Material),
		})
	}

	effects, err := s.queries.ListActiveEffectsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, ef := range effects {
		dto.ActiveEffects = append(dto.ActiveEffects, EffectDTO{
			ID: ef.ID, CatalogID: ef.Catalogid, Scope: ef.Scope,
			Modifiers: ef.Modifiers, CreatedAt: ef.Createdat,
		})
	}

	spells, err := s.queries.ListSpellsByCharacter(ctx, c.ID)
	if err != nil {
		return dto, err
	}
	for _, sp := range spells {
		dto.Spells = append(dto.Spells, SpellDTO{
			ID: sp.ID, CatalogSpellID: sp.Catalogspellid, Prepared: sp.Prepared != 0, LearnedAt: sp.Learnedat,
		})
	}
	return dto, nil
}

// intParam parses a chi :id-style path param, writing a 400 (like ParseIntPipe)
// and returning false on a non-numeric value.
func intParam(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	n, err := parseInt(chi.URLParam(r, name))
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Validation failed (numeric string is expected for %q)", name))
		return 0, false
	}
	return int64(n), true
}
