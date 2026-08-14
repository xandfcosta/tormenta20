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
// assert it belongs to the campaign. — the role is
// stashed on socket.data for per-action GM gating. Transport-agnostic (WS maps status/err).
func (s *Server) sessionForCaller(ctx context.Context, user AuthUser, campaignID, sessionID int64) (sqlcgen.Session, string, int, error) {
	role, status, err := s.resolveRole(ctx, user, campaignID)
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
	// Member-aware (ALE-19): a player who is a member must be able to load the
	// session before the socket connects — the WS gateway already gates on
	// sessionForCaller, so mirror it here instead of the owner-only ownedSession
	// (which 403'd invited players with "belongs to another user").
	sess, _, status, err := s.sessionForCaller(r.Context(), currentUser(r), cid, sid)
	if err != nil {
		writeError(w, status, err.Error())
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
	var set setBuilder
	if body.SessionNumber != nil {
		if *body.SessionNumber < 1 {
			writeValidationError(w, FieldErrorMap{"sessionNumber": {"sessionNumber must not be less than 1"}})
			return
		}
		set.add("sessionNumber = ?", *body.SessionNumber)
	}
	if body.Title != nil {
		set.add("title = ?", nullableArg(trimOrNull(body.Title)))
	}
	if body.Notes != nil {
		set.add("notes = ?", nullableArg(trimOrNull(body.Notes)))
	}
	if set.empty() {
		writeError(w, http.StatusBadRequest, "No fields to update")
		return
	}
	if err := set.execTouched(r.Context(), s.db, "UPDATE sessions", sid); err != nil {
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
	// Drop the in-memory tracker too — otherwise a live session's cached state
	// would shadow the cleared DB row until the next cold load (the realtime
	// store hydrates only on first access). Code-review finding (B.6 fase 2).
	s.sessions.forget(sid)
	writeJSON(w, http.StatusOK, map[string]int64{"id": sid})
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
