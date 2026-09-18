package api

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/httpio"
	"t20engine/infra/wire"

	"t20engine/infra/db/sqlcgen"
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
		ID: s.ID, CampaignID: s.Campaignid, Title: dbvalue.NullToPtr(s.Title), SessionNumber: s.Sessionnumber,
		Notes: dbvalue.NullToPtr(s.Notes), Status: s.Status, StartedAt: dbvalue.NullToPtr(s.Startedat), EndedAt: dbvalue.NullToPtr(s.Endedat),
		CreatedAt: s.Createdat, UpdatedAt: s.Updatedat, RuntimeState: s.Runtimestate,
	}
}

// sessionForCaller is the member-aware session resolver the WS gateway runs on every
// session-scoped message: resolve the caller's Role (gm/player) then Load the session and
// assert it belongs to the campaign. — the Role is
// stashed on socket.data for per-action GM gating. Transport-agnostic (WS maps status/err).
func (rules campaignRules) sessionForCaller(ctx context.Context, user AuthUser, campaignID, sessionID int64) (sqlcgen.Session, string, int, error) {
	sess, papel, err := rules.access().Session(ctx, callerOf(user), campaignID, sessionID)
	if err != nil {
		return sqlcgen.Session{}, "", statusForAccess(err), err
	}
	return sess, papel, http.StatusOK, nil
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
	if !httpio.DecodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.campaignRules().ownedCampaign(w, r, cid); !ok {
		return
	}
	if body.SessionNumber == nil || *body.SessionNumber < 1 {
		httpio.WriteValidationError(w, wire.FieldErrorMap{"sessionNumber": {"sessionNumber must not be less than 1"}})
		return
	}
	now := dbvalue.NowISO()
	sess, err := s.queries.CreateSession(r.Context(), sqlcgen.CreateSessionParams{
		Campaignid: cid, Sessionnumber: *body.SessionNumber, Title: trimOrNull(body.Title), Notes: trimOrNull(body.Notes),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		httpio.WriteError(w, http.StatusInternalServerError, "Could not create session")
		return
	}
	httpio.WriteJSON(w, http.StatusCreated, sessionDTO(sess))
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
