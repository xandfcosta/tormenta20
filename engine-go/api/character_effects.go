package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"t20engine/db/sqlcgen"
)

// handleAdjustEffect ports adjustActiveEffect: bump a temp-HP pool's amount by
// tempHpDelta; delete when it hits 0 ({removed, id}), else return the effect.
func (s *Server) handleAdjustEffect(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	effectID, ok := intParam(w, r, "effectId")
	if !ok {
		return
	}
	var body struct {
		TempHpDelta *int64 `json:"tempHpDelta"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	if body.TempHpDelta == nil {
		writeValidationError(w, FieldErrorMap{"tempHpDelta": {"tempHpDelta must be an integer number"}})
		return
	}
	eff, err := s.queries.GetActiveEffect(r.Context(), effectID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && eff.Characterid != id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Active effect %d not found for character %d", effectID, id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load effect")
		return
	}

	var mods []map[string]any
	if json.Unmarshal([]byte(eff.Modifiers), &mods) != nil {
		writeError(w, http.StatusBadRequest, "Effect modifiers are malformed")
		return
	}
	idx := -1
	for i, m := range mods {
		if isTempHpModifier(m) {
			idx = i
			break
		}
	}
	if idx < 0 {
		writeError(w, http.StatusBadRequest, "Active effect has no temp HP to adjust")
		return
	}
	amount := max(0, toInt(mods[idx]["amount"])+int(*body.TempHpDelta))
	if amount == 0 {
		if err := s.queries.DeleteEffectByID(r.Context(), effectID); err != nil {
			writeError(w, http.StatusInternalServerError, "Could not remove effect")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"removed": true, "id": effectID})
		return
	}
	mods[idx]["amount"] = amount
	next, _ := json.Marshal(mods)
	if err := s.queries.UpdateEffectModifiers(r.Context(), sqlcgen.UpdateEffectModifiersParams{Modifiers: string(next), ID: effectID}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update effect")
		return
	}
	writeJSON(w, http.StatusOK, EffectDTO{
		ID: eff.ID, CatalogID: eff.Catalogid, Scope: eff.Scope, Modifiers: string(next), CreatedAt: eff.Createdat,
	})
}

// handleDeleteEffect ports removeActiveEffect: 404 if the effect isn't on this
// character; returns {id}.
func (s *Server) handleDeleteEffect(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	effectID, ok := intParam(w, r, "effectId")
	if !ok {
		return
	}
	if _, status, err := s.authorizedCharacter(r.Context(), currentUser(r), id); err != nil {
		writeError(w, status, err.Error())
		return
	}
	meta, err := s.queries.GetActiveEffectMeta(r.Context(), effectID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && meta.Characterid != id) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Active effect %d not found", effectID))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load effect")
		return
	}
	if err := s.queries.DeleteEffectByID(r.Context(), effectID); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not remove effect")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": effectID})
}

type characterCampaignDTO struct {
	ID          int64              `json:"id"`
	CampaignID  int64              `json:"campaignId"`
	CharacterID int64              `json:"characterId"`
	Role        string             `json:"role"`
	AddedAt     string             `json:"addedAt"`
	Campaign    campaignSummaryDTO `json:"campaign"`
}

type campaignSummaryDTO struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	UpdatedAt   string  `json:"updatedAt"`
}

// handleListCharacterCampaigns ports members.listForCharacter: the campaigns a
// character has joined (owner-only, NOT GM). 404 missing, 403 not-owner.
func (s *Server) handleListCharacterCampaigns(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	owner, err := s.queries.GetCharacterOwner(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Character %d not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	if owner != currentUser(r).ID {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Character %d belongs to another user", id))
		return
	}
	rows, err := s.queries.ListCampaignsForCharacter(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list campaigns")
		return
	}
	out := make([]characterCampaignDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, characterCampaignDTO{
			ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat,
			Campaign: campaignSummaryDTO{
				ID: m.Campaignid, Name: m.Campaignname,
				Description: nullToPtr(m.Campaigndescription), UpdatedAt: m.Campaignupdatedat,
			},
		})
	}
	writeJSON(w, http.StatusOK, out)
}
