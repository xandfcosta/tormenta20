package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"t20engine/plataforma"

	"t20engine/db/sqlcgen"
)

const defaultRuntimeState = `{"initiative":[],"round":0,"turnIndex":-1}`

type SessionDTO struct {
	ID            int64   `json:"id"`
	CampaignID    int64   `json:"campaignId"`
	Title         *string `json:"title"`
	SessionNumber int64   `json:"sessionNumber"`
	Notes         *string `json:"notes"`
	Status        string  `json:"status"`
	StartedAt     *string `json:"startedAt"`
	EndedAt       *string `json:"endedAt"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
	RuntimeState  string  `json:"runtimeState"`
}

func sessionDTO(s sqlcgen.Session) SessionDTO {
	return SessionDTO{
		ID: s.ID, CampaignID: s.Campaignid, Title: plataforma.NullToPtr(s.Title), SessionNumber: s.Sessionnumber,
		Notes: plataforma.NullToPtr(s.Notes), Status: s.Status, StartedAt: plataforma.NullToPtr(s.Startedat), EndedAt: plataforma.NullToPtr(s.Endedat),
		CreatedAt: s.Createdat, UpdatedAt: s.Updatedat, RuntimeState: s.Runtimestate,
	}
}

// loadSessionInCampaign loads a session and asserts it belongs to the campaign —
// transport-agnostic, no access check of its own. Ela era compartilhada pelo
// `ownedSession` (só o dono) e pelo `sessionForCaller` do gateway, para a regra
// "a sessão é desta campanha" morar num lugar só. O `ownedSession` foi apagado
// com as rotas JSON na ALE-277; o `sessionForCaller` ficou, e é por ele que as
// cenas passam.
func (s *Server) loadSessionInCampaign(ctx context.Context, campaignID, sessionID int64) (sqlcgen.Session, int, error) {
	sess, err := s.queries.GetSession(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && sess.Campaignid != campaignID) {
		return sqlcgen.Session{}, http.StatusNotFound, fmt.Errorf("Session %d not found", sessionID)
	}
	if err != nil {
		return sqlcgen.Session{}, http.StatusInternalServerError, errors.New("Could not Load session")
	}
	return sess, http.StatusOK, nil
}

// sessionForCaller is the member-aware session resolver the WS gateway runs on every
// session-scoped message: resolve the caller's Role (gm/player) then Load the session and
// assert it belongs to the campaign. — the Role is
// stashed on socket.data for per-action GM gating. Transport-agnostic (WS maps status/err).
func (s *Server) sessionForCaller(ctx context.Context, user AuthUser, campaignID, sessionID int64) (sqlcgen.Session, string, int, error) {
	Role, status, err := s.resolveRole(ctx, user, campaignID)
	if err != nil {
		return sqlcgen.Session{}, "", status, err
	}
	sess, status, err := s.loadSessionInCampaign(ctx, campaignID, sessionID)
	if err != nil {
		return sqlcgen.Session{}, "", status, err
	}
	return sess, Role, http.StatusOK, nil
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	var body struct {
		SessionNumber *int64  `json:"sessionNumber"`
		Title         *string `json:"title"`
		Notes         *string `json:"notes"`
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedCampaign(w, r, cid); !ok {
		return
	}
	if body.SessionNumber == nil || *body.SessionNumber < 1 {
		plataforma.WriteValidationError(w, plataforma.FieldErrorMap{"sessionNumber": {"sessionNumber must not be less than 1"}})
		return
	}
	now := plataforma.NowISO()
	sess, err := s.queries.CreateSession(r.Context(), sqlcgen.CreateSessionParams{
		Campaignid: cid, Sessionnumber: *body.SessionNumber, Title: trimOrNull(body.Title), Notes: trimOrNull(body.Notes),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		plataforma.WriteError(w, http.StatusInternalServerError, "Could not create session")
		return
	}
	plataforma.WriteJSON(w, http.StatusCreated, sessionDTO(sess))
}

// trimOrNull trims a string pointer, treating nil AND whitespace-only as NULL.
// The one spelling of "blank" for every nullable TEXT column: campaigns used to
// have a second one (`trimmedNull`) that stored an empty string instead, so the
// same input produced "" on create and null on update.
func trimOrNull(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	t := strings.TrimSpace(*p)
	if t == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: t, Valid: true}
}

// nullableArg converts a NullString to a driver arg (nil when invalid) for the
// dynamic UPDATE builders.
func nullableArg(ns sql.NullString) any {
	if !ns.Valid {
		return nil
	}
	return ns.String
}
