package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"t20engine/db/sqlcgen"
)

var campaignMemberRoles = map[string]bool{"player": true, "gm": true}

type memberDTO struct {
	ID          int64               `json:"id"`
	CampaignID  int64               `json:"campaignId"`
	CharacterID int64               `json:"characterId"`
	Role        string              `json:"role"`
	AddedAt     string              `json:"addedAt"`
	Character   *memberCharacterDTO `json:"character,omitempty"`
}

type memberCharacterDTO struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Level     int64      `json:"level"`
	HpCurrent int64      `json:"hpCurrent"`
	HpMax     int64      `json:"hpMax"`
	MpCurrent int64      `json:"mpCurrent"`
	MpMax     int64      `json:"mpMax"`
	Classes   []ClassDTO `json:"classes"`
}

func memberScalars(m sqlcgen.CampaignMember) memberDTO {
	return memberDTO{ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat}
}

func (s *Server) classDTOs(r *http.Request, characterID int64) []ClassDTO {
	classes, _ := s.queries.ListClassesByCharacter(r.Context(), characterID)
	out := make([]ClassDTO, 0, len(classes))
	for _, cl := range classes {
		out = append(out, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
	}
	return out
}

// campaignAccess enforces owner-or-member read access (resolveAccess), writing
// the 404/403 and returning false.
func (s *Server) campaignAccess(w http.ResponseWriter, r *http.Request, campaignID int64) bool {
	c, err := s.queries.GetCampaign(r.Context(), campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Campaign %d not found", campaignID))
		return false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load campaign")
		return false
	}
	if c.Ownerid == currentUser(r).ID {
		return true
	}
	isMember, _ := s.queries.IsCampaignMember(r.Context(), sqlcgen.IsCampaignMemberParams{Campaignid: campaignID, Ownerid: currentUser(r).ID})
	if !isMember {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Campaign %d is not accessible", campaignID))
		return false
	}
	return true
}

// handleListMembers ports members.list: any player in the campaign sees the
// roster with live vitals.
func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	if !s.campaignAccess(w, r, cid) {
		return
	}
	rows, err := s.queries.ListMembers(r.Context(), cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list members")
		return
	}
	out := make([]memberDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, memberDTO{
			ID: m.ID, CampaignID: m.Campaignid, CharacterID: m.Characterid, Role: m.Role, AddedAt: m.Addedat,
			Character: &memberCharacterDTO{
				ID: m.Characterid, Name: m.Charname, Level: m.Charlevel,
				HpCurrent: m.Charhpcurrent, HpMax: m.Charhpmax, MpCurrent: m.Charmpcurrent, MpMax: m.Charmpmax,
				Classes: s.classDTOs(r, m.Characterid),
			},
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// handleAddMember ports members.add: caller must own the character; owner joins
// freely, others need a valid invite token; one player-PC per user per campaign.
func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	var body struct {
		CharacterID *int64  `json:"characterId"`
		Role        *string `json:"role"`
		InviteToken *string `json:"inviteToken"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	user := currentUser(r)

	c, err := s.queries.GetCampaign(r.Context(), cid)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Campaign %d not found", cid))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load campaign")
		return
	}
	if c.Ownerid != user.ID {
		token := derefStr(body.InviteToken, "")
		if !c.Invitetoken.Valid || token == "" || token != c.Invitetoken.String {
			writeError(w, http.StatusForbidden, fmt.Sprintf("A valid invite token is required to join campaign %d", cid))
			return
		}
	}
	if body.CharacterID == nil {
		writeValidationError(w, FieldErrorMap{"characterId": {"characterId must be an integer number"}})
		return
	}
	owner, err := s.queries.GetCharacterOwner(r.Context(), *body.CharacterID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"statusCode": http.StatusBadRequest, "error": "Bad Request",
			"message":     fmt.Sprintf("Character %d not found", *body.CharacterID),
			"fieldErrors": FieldErrorMap{"characterId": {"Character does not exist"}},
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load character")
		return
	}
	if owner != user.ID {
		writeError(w, http.StatusForbidden, fmt.Sprintf("Cannot add a character you don't own (character %d)", *body.CharacterID))
		return
	}
	role := derefStr(body.Role, "player")
	if !campaignMemberRoles[role] {
		writeValidationError(w, FieldErrorMap{"role": {"role must be one of: player, gm"}})
		return
	}
	if role == "player" {
		if hasPc, _ := s.queries.HasPlayerPc(r.Context(), sqlcgen.HasPlayerPcParams{Campaignid: cid, Ownerid: user.ID}); hasPc {
			writeJSON(w, http.StatusConflict, map[string]any{
				"statusCode": http.StatusConflict, "error": "Conflict",
				"message":     fmt.Sprintf("You already have a character in campaign %d", cid),
				"fieldErrors": FieldErrorMap{"characterId": {"Você já tem um personagem nesta campanha"}},
			})
			return
		}
	}
	if isMember, _ := s.queries.IsCharacterMember(r.Context(), sqlcgen.IsCharacterMemberParams{Campaignid: cid, Characterid: *body.CharacterID}); isMember {
		writeJSON(w, http.StatusConflict, map[string]any{
			"statusCode": http.StatusConflict, "error": "Conflict",
			"message":     fmt.Sprintf("Character %d already in campaign %d", *body.CharacterID, cid),
			"fieldErrors": FieldErrorMap{"characterId": {"Already a member"}},
		})
		return
	}
	m, err := s.queries.CreateMember(r.Context(), sqlcgen.CreateMemberParams{
		Campaignid: cid, Characterid: *body.CharacterID, Role: role, Addedat: nowISO(),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not add member")
		return
	}
	writeJSON(w, http.StatusCreated, memberScalars(m))
}

// handleUpdateMemberRole ports updateRole (owner-only).
func (s *Server) handleUpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	mid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Role string `json:"role"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedCampaign(w, r, cid); !ok {
		return
	}
	if !campaignMemberRoles[body.Role] {
		writeValidationError(w, FieldErrorMap{"role": {"role must be one of: player, gm"}})
		return
	}
	m, err := s.queries.GetMember(r.Context(), mid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && m.Campaignid != cid) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Member %d not found", mid))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load member")
		return
	}
	updated, err := s.queries.SetMemberRole(r.Context(), sqlcgen.SetMemberRoleParams{Role: body.Role, ID: mid})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update member")
		return
	}
	writeJSON(w, http.StatusOK, memberScalars(updated))
}

// handleRemoveMember ports members.remove: GM or the character's owner may remove.
func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	mid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	owners, err := s.queries.GetMemberOwners(r.Context(), mid)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && owners.Campaignid != cid) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Member %d not found", mid))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load member")
		return
	}
	uid := currentUser(r).ID
	if owners.Campaignowner != uid && owners.Characterowner != uid {
		writeError(w, http.StatusForbidden, "You are neither the GM of this campaign nor the character's owner")
		return
	}
	if err := s.queries.DeleteMember(r.Context(), mid); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not remove member")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": mid})
}
