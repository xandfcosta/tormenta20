package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"t20engine/db/sqlcgen"
)

// CampaignDTO is the base campaign row (create/update responses).
type CampaignDTO struct {
	ID          int64   `json:"id"`
	OwnerID     int64   `json:"ownerId"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

type campaignCharacterDTO struct {
	ID      int64      `json:"id"`
	Name    string     `json:"name"`
	Level   int64      `json:"level"`
	Classes []ClassDTO `json:"classes"`
}

// campaignListDTO adds the caller's role + own member character (GET /campaigns).
type campaignListDTO struct {
	CampaignDTO
	Role      string                `json:"role"`
	Character *campaignCharacterDTO `json:"character"`
}

type campaignDetailDTO struct {
	CampaignDTO
	Role string `json:"role"`
}

func campaignScalars(c sqlcgen.Campaign) CampaignDTO {
	return CampaignDTO{
		ID: c.ID, OwnerID: c.Ownerid, Name: c.Name, Description: nullToPtr(c.Description),
		CreatedAt: c.Createdat, UpdatedAt: c.Updatedat,
	}
}

func (s *Server) handleListCampaigns(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	rows, err := s.queries.ListCampaignsForUser(r.Context(), user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list campaigns")
		return
	}
	out := make([]campaignListDTO, 0, len(rows))
	for _, c := range rows {
		item := campaignListDTO{CampaignDTO: campaignScalars(c), Role: "player"}
		if c.Ownerid == user.ID {
			item.Role = "gm"
		}
		char, err := s.queries.CallerCharacterInCampaign(r.Context(), sqlcgen.CallerCharacterInCampaignParams{Campaignid: c.ID, Ownerid: user.ID})
		if err == nil {
			classes, _ := s.queries.ListClassesByCharacter(r.Context(), char.ID)
			cc := &campaignCharacterDTO{ID: char.ID, Name: char.Name, Level: char.Level, Classes: []ClassDTO{}}
			for _, cl := range classes {
				cc.Classes = append(cc.Classes, ClassDTO{ClassName: cl.Classname, Level: cl.Level})
			}
			item.Character = cc
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, out)
}

// handleGetCampaign ports resolveAccess: owner → gm, member → player, else 403.
func (s *Server) handleGetCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	user := currentUser(r)
	c, err := s.queries.GetCampaign(r.Context(), id)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Campaign %d not found", id))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not load campaign")
		return
	}
	role := "gm"
	if c.Ownerid != user.ID {
		isMember, _ := s.queries.IsCampaignMember(r.Context(), sqlcgen.IsCampaignMemberParams{Campaignid: id, Ownerid: user.ID})
		if !isMember {
			writeError(w, http.StatusForbidden, fmt.Sprintf("Campaign %d is not accessible", id))
			return
		}
		role = "player"
	}
	writeJSON(w, http.StatusOK, campaignDetailDTO{CampaignDTO: campaignScalars(c), Role: role})
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string  `json:"name"`
		Description *string `json:"description"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" || len([]rune(name)) > 120 {
		writeValidationError(w, FieldErrorMap{"name": {"name must be between 1 and 120 characters"}})
		return
	}
	now := nowISO()
	c, err := s.queries.CreateCampaign(r.Context(), sqlcgen.CreateCampaignParams{
		Ownerid: currentUser(r).ID, Name: name, Description: trimOrNull(body.Description),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create campaign")
		return
	}
	writeJSON(w, http.StatusCreated, campaignScalars(c))
}

func (s *Server) handleUpdateCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	var set setBuilder
	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if name == "" || len([]rune(name)) > 120 {
			writeValidationError(w, FieldErrorMap{"name": {"name must be between 1 and 120 characters"}})
			return
		}
		set.add("name = ?", name)
	}
	if body.Description != nil {
		// Same helper as create: a whitespace-only description is NULL on both
		// paths, or the client reads "" from one and null from the other for the
		// very same input.
		set.add("description = ?", nullableArg(trimOrNull(body.Description)))
	}
	if set.empty() {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}
	if err := set.execTouched(r.Context(), s.db, "UPDATE campaigns", id); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update campaign")
		return
	}
	c, _ := s.queries.GetCampaign(r.Context(), id)
	writeJSON(w, http.StatusOK, campaignScalars(c))
}

func (s *Server) handleDeleteCampaign(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	if err := s.queries.DeleteCampaign(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not delete campaign")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": id})
}

// handleRotateInvite ports rotateInviteToken (owner-only): {campaignId, token}.
func (s *Server) handleRotateInvite(w http.ResponseWriter, r *http.Request) {
	id, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.ownedCampaign(w, r, id); !ok {
		return
	}
	row, err := s.queries.SetInviteToken(r.Context(), sqlcgen.SetInviteTokenParams{
		InviteToken: sql.NullString{String: generateInviteToken(), Valid: true}, UpdatedAt: nowISO(), ID: id,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not rotate invite")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaignId": row.ID, "token": row.Invitetoken.String})
}

// handleResolveInvite resolves a shared token to {campaignId, campaignName}
// (public). The frontend's CampaignInvitePreview expects camelCase keys —
// returning {id, name} left the join form with an undefined campaignId, so the
// "Entrar" button stayed disabled forever (ALE-18). Mirrors handleRotateInvite,
// which already returns campaignId.
//
// An unknown or rotated token is a 404. It used to answer 200 with a `null`
// body, which made a dead invite arrive at the client as a SUCCESS carrying no
// campaign — indistinguishable from one still loading, so the join screen sat
// there with a disabled button and no explanation (ALE-80). A missing thing is
// a 404; only a genuine lookup failure is a 500.
func (s *Server) handleResolveInvite(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeError(w, http.StatusNotFound, "Invite not found")
		return
	}
	c, err := s.queries.GetCampaignByToken(r.Context(), sql.NullString{String: token, Valid: true})
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Invite not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not resolve invite")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campaignId": c.ID, "campaignName": c.Name})
}

// resolveRole is the campaign-access domain rule,
// transport-agnostic so both the HTTP handlers and the WS gateway can gate on it: the
// owner is the "gm"; a user who owns a member character is a "player"; anyone else is
// forbidden. Returns the role + an HTTP-ish status the caller maps to its transport.
func (s *Server) resolveRole(ctx context.Context, userID, campaignID int64) (string, int, error) {
	c, err := s.queries.GetCampaign(ctx, campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", http.StatusNotFound, fmt.Errorf("Campaign %d not found", campaignID)
	}
	if err != nil {
		return "", http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	if c.Ownerid == userID {
		return "gm", http.StatusOK, nil
	}
	isMember, _ := s.queries.IsCampaignMember(ctx, sqlcgen.IsCampaignMemberParams{Campaignid: campaignID, Ownerid: userID})
	if !isMember {
		return "", http.StatusForbidden, fmt.Errorf("Campaign %d is not accessible", campaignID)
	}
	return "player", http.StatusOK, nil
}

// loadOwnedCampaign is the owner-only campaign rule, transport-agnostic. The GM (owner)
// alone passes; everyone else gets Forbidden.
func (s *Server) loadOwnedCampaign(ctx context.Context, userID, id int64) (sqlcgen.Campaign, int, error) {
	c, err := s.queries.GetCampaign(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return c, http.StatusNotFound, fmt.Errorf("Campaign %d not found", id)
	}
	if err != nil {
		return c, http.StatusInternalServerError, errors.New("Could not load campaign")
	}
	if c.Ownerid != userID {
		return c, http.StatusForbidden, fmt.Errorf("Campaign %d belongs to another user", id)
	}
	return c, http.StatusOK, nil
}

func (s *Server) ownedCampaign(w http.ResponseWriter, r *http.Request, id int64) (sqlcgen.Campaign, bool) {
	c, status, err := s.loadOwnedCampaign(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeError(w, status, err.Error())
		return c, false
	}
	return c, true
}

func generateInviteToken() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
