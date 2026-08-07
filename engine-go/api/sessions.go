package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

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
		ID: s.ID, CampaignID: s.Campaignid, Title: nullToPtr(s.Title), SessionNumber: s.Sessionnumber,
		Notes: nullToPtr(s.Notes), Status: s.Status, StartedAt: nullToPtr(s.Startedat), EndedAt: nullToPtr(s.Endedat),
		CreatedAt: s.Createdat, UpdatedAt: s.Updatedat, RuntimeState: s.Runtimestate,
	}
}

// ownedSession loads a session under an owned campaign (findOne), writing the
// 404/403 and returning ok=false.
// loadSessionInCampaign loads a session and asserts it belongs to the campaign —
// transport-agnostic, no access check of its own. Shared by ownedSession (owner-only)
// and the WS gateway's member-aware sessionForCaller so the "session belongs to the
// campaign" rule lives in one place.
func (s *Server) loadSessionInCampaign(ctx context.Context, campaignID, sessionID int64) (sqlcgen.Session, int, error) {
	sess, err := s.queries.GetSession(ctx, sessionID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && sess.Campaignid != campaignID) {
		return sqlcgen.Session{}, http.StatusNotFound, fmt.Errorf("Session %d not found", sessionID)
	}
	if err != nil {
		return sqlcgen.Session{}, http.StatusInternalServerError, errors.New("Could not load session")
	}
	return sess, http.StatusOK, nil
}

// sessionForCaller is the member-aware session resolver the WS gateway runs on every
// session-scoped message: resolve the caller's role (gm/player) then load the session and
// assert it belongs to the campaign. Mirrors SessionsService.findOneForCaller — the role is
// stashed on socket.data for per-action GM gating. Transport-agnostic (WS maps status/err).
func (s *Server) sessionForCaller(ctx context.Context, userID, campaignID, sessionID int64) (sqlcgen.Session, string, int, error) {
	role, status, err := s.resolveRole(ctx, userID, campaignID)
	if err != nil {
		return sqlcgen.Session{}, "", status, err
	}
	sess, status, err := s.loadSessionInCampaign(ctx, campaignID, sessionID)
	if err != nil {
		return sqlcgen.Session{}, "", status, err
	}
	return sess, role, http.StatusOK, nil
}

func (s *Server) ownedSession(w http.ResponseWriter, r *http.Request, campaignID, sessionID int64) (sqlcgen.Session, bool) {
	if _, ok := s.ownedCampaign(w, r, campaignID); !ok {
		return sqlcgen.Session{}, false
	}
	sess, status, err := s.loadSessionInCampaign(r.Context(), campaignID, sessionID)
	if err != nil {
		writeError(w, status, err.Error())
		return sqlcgen.Session{}, false
	}
	return sess, true
}

// handleListSessions ports listForCaller: member-aware, ordered by sessionNumber.
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	if !s.campaignAccess(w, r, cid) {
		return
	}
	rows, err := s.queries.ListSessions(r.Context(), cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not list sessions")
		return
	}
	out := make([]SessionDTO, 0, len(rows))
	for _, sess := range rows {
		out = append(out, sessionDTO(sess))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	sess, ok := s.ownedSession(w, r, cid, sid)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, sessionDTO(sess))
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
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedCampaign(w, r, cid); !ok {
		return
	}
	if body.SessionNumber == nil || *body.SessionNumber < 1 {
		writeValidationError(w, FieldErrorMap{"sessionNumber": {"sessionNumber must not be less than 1"}})
		return
	}
	now := nowISO()
	sess, err := s.queries.CreateSession(r.Context(), sqlcgen.CreateSessionParams{
		Campaignid: cid, Sessionnumber: *body.SessionNumber, Title: trimOrNull(body.Title), Notes: trimOrNull(body.Notes),
		Createdat: now, Updatedat: now,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not create session")
		return
	}
	writeJSON(w, http.StatusCreated, sessionDTO(sess))
}

func (s *Server) handleUpdateSession(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	var body struct {
		SessionNumber *int64  `json:"sessionNumber"`
		Title         *string `json:"title"`
		Notes         *string `json:"notes"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if _, ok := s.ownedSession(w, r, cid, sid); !ok {
		return
	}
	sets := []string{}
	args := []any{}
	if body.SessionNumber != nil {
		if *body.SessionNumber < 1 {
			writeValidationError(w, FieldErrorMap{"sessionNumber": {"sessionNumber must not be less than 1"}})
			return
		}
		sets, args = append(sets, "sessionNumber = ?"), append(args, *body.SessionNumber)
	}
	if body.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, nullableArg(trimOrNull(body.Title)))
	}
	if body.Notes != nil {
		sets = append(sets, "notes = ?")
		args = append(args, nullableArg(trimOrNull(body.Notes)))
	}
	if len(sets) == 0 {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}
	sets = append(sets, "updatedAt = ?")
	args = append(args, nowISO(), sid)
	//nolint:gosec // fixed column allowlist.
	if _, err := s.db.ExecContext(r.Context(), "UPDATE sessions SET "+strings.Join(sets, ", ")+" WHERE id = ?", args...); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not update session")
		return
	}
	sess, _ := s.queries.GetSession(r.Context(), sid)
	writeJSON(w, http.StatusOK, sessionDTO(sess))
}

func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.ownedSession(w, r, cid, sid); !ok {
		return
	}
	if err := s.queries.DeleteSession(r.Context(), sid); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not delete session")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": sid})
}

// handleStartSession ports start: active is a no-op, ended reopens, planned starts.
func (s *Server) handleStartSession(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	sess, ok := s.ownedSession(w, r, cid, sid)
	if !ok {
		return
	}
	now := nowISO()
	var updated sqlcgen.Session
	var err error
	switch sess.Status {
	case "active":
		writeJSON(w, http.StatusOK, sessionDTO(sess))
		return
	case "ended":
		updated, err = s.queries.ReopenSession(r.Context(), sqlcgen.ReopenSessionParams{UpdatedAt: now, ID: sid})
	default:
		updated, err = s.queries.StartSessionFresh(r.Context(), sqlcgen.StartSessionFreshParams{StartedAt: sql.NullString{String: now, Valid: true}, UpdatedAt: now, ID: sid})
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not start session")
		return
	}
	writeJSON(w, http.StatusOK, sessionDTO(updated))
}

// handleEndSession ports end: planned → 400, ended → no-op, active → ends. The WS
// vitals write-through is deferred to B.6.
func (s *Server) handleEndSession(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	sess, ok := s.ownedSession(w, r, cid, sid)
	if !ok {
		return
	}
	switch sess.Status {
	case "planned":
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Session %d was never started; nothing to end", sid))
		return
	case "ended":
		writeJSON(w, http.StatusOK, sessionDTO(sess))
		return
	}
	now := nowISO()
	updated, err := s.queries.EndSession(r.Context(), sqlcgen.EndSessionParams{EndedAt: sql.NullString{String: now, Valid: true}, UpdatedAt: now, ID: sid})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Could not end session")
		return
	}
	writeJSON(w, http.StatusOK, sessionDTO(updated))
}

// handleClearTracker resets the initiative runtime state to empty.
func (s *Server) handleClearTracker(w http.ResponseWriter, r *http.Request) {
	cid, ok := intParam(w, r, "campaignId")
	if !ok {
		return
	}
	sid, ok := intParam(w, r, "id")
	if !ok {
		return
	}
	if _, ok := s.ownedSession(w, r, cid, sid); !ok {
		return
	}
	if err := s.queries.ResetSessionTracker(r.Context(), sqlcgen.ResetSessionTrackerParams{
		RuntimeState: defaultRuntimeState, UpdatedAt: nowISO(), ID: sid,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Could not clear tracker")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"id": sid})
}

// trimOrNull trims a string pointer, treating nil/empty as NULL.
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
