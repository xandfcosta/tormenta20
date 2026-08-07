package api

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"

	"t20engine/db/sqlcgen"
)

// newUUID generates a random v4 UUID string for initiative entry ids (randomUUID() in the
// Nest service). Injected into the store so tests can swap a deterministic generator.
func newUUID() string { return uuid.NewString() }

// sessionStore holds each session's in-memory runtime state, guarding the pure mutations
// (session_state.go) with a mutex and persisting fire-and-forget to Session.runtimeState.
// Mirrors SessionStateService — a server restart wipes trackers until the first load()
// re-hydrates from the DB. Mutation methods return a snapshot (deep-enough copy) so the
// gateway can serialize/broadcast it without racing a concurrent message on the session.
type sessionStore struct {
	mu     sync.Mutex
	states map[int64]*SessionRuntimeState
	dirty  map[int64]bool
	newID  func() string
	q      *sqlcgen.Queries
	// persistMus holds a per-session mutex (sessionID → *sync.Mutex) serializing that
	// session's DB writes (runtime-state persist + vitals write-through) so concurrent
	// mutations can't land out of order — WITHOUT coupling latency across sessions.
	persistMus sync.Map
	// liveWriteThrough mirrors vitals patch/delta back to the Character row (opt-in).
	liveWriteThrough bool
}

// persistLock returns the per-session DB-write mutex, creating it on first use.
func (st *sessionStore) persistLock(sessionID int64) *sync.Mutex {
	m, _ := st.persistMus.LoadOrStore(sessionID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func newSessionStore(q *sqlcgen.Queries, newID func() string, liveWriteThrough bool) *sessionStore {
	return &sessionStore{
		states:           map[int64]*SessionRuntimeState{},
		dirty:            map[int64]bool{},
		newID:            newID,
		q:                q,
		liveWriteThrough: liveWriteThrough,
	}
}

// cloneState copies the state for broadcast. The entry structs are copied by value; their
// *int64 vitals are shared but never mutated in place (patch/delta always assign a fresh
// pointer), so the snapshot is safe to serialize outside the lock.
func cloneState(s *SessionRuntimeState) *SessionRuntimeState {
	entries := make([]InitiativeEntry, len(s.Initiative))
	copy(entries, s.Initiative)
	return &SessionRuntimeState{Initiative: entries, Round: s.Round, TurnIndex: s.TurnIndex}
}

func (st *sessionStore) getOrCreateLocked(sessionID int64) *SessionRuntimeState {
	s := st.states[sessionID]
	if s == nil {
		s = emptyRuntimeState()
		st.states[sessionID] = s
	}
	return s
}

// getState returns a snapshot of the current state (an empty tracker if never loaded).
func (st *sessionStore) getState(sessionID int64) *SessionRuntimeState {
	st.mu.Lock()
	defer st.mu.Unlock()
	return cloneState(st.getOrCreateLocked(sessionID))
}

// apply runs a pure mutation under the lock and returns a snapshot for broadcast.
func (st *sessionStore) apply(sessionID int64, fn func(*SessionRuntimeState) error) (*SessionRuntimeState, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.getOrCreateLocked(sessionID)
	if err := fn(s); err != nil {
		return nil, err
	}
	return cloneState(s), nil
}

func (st *sessionStore) addInitiativeEntry(sessionID int64, e InitiativeEntry) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return addEntry(s, e, st.newID) })
}

func (st *sessionStore) upsertInitiativeEntry(sessionID int64, e InitiativeEntry) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return upsertCharacterEntry(s, e, st.newID) })
}

func (st *sessionStore) updateInitiativeEntry(sessionID int64, entryID string, patch entryPatch) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return updateEntry(s, entryID, patch) })
}

func (st *sessionStore) removeInitiativeEntry(sessionID int64, entryID string) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { return removeEntry(s, entryID) })
}

func (st *sessionStore) nextTurn(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { advanceTurn(s); return nil })
}

func (st *sessionStore) reset(sessionID int64) (*SessionRuntimeState, error) {
	return st.apply(sessionID, func(s *SessionRuntimeState) error { resetInitiative(s); return nil })
}

func (st *sessionStore) patchVitals(sessionID int64, entryID string, hpCurrent, mpCurrent *int64) (*SessionRuntimeState, error) {
	snap, err := st.apply(sessionID, func(s *SessionRuntimeState) error { return patchEntryVitals(s, entryID, hpCurrent, mpCurrent) })
	st.maybeWriteThrough(sessionID, entryID, err)
	return snap, err
}

func (st *sessionStore) deltaVitals(sessionID int64, entryID string, hpDelta, mpDelta *int64) (*SessionRuntimeState, error) {
	snap, err := st.apply(sessionID, func(s *SessionRuntimeState) error { return deltaEntryVitals(s, entryID, hpDelta, mpDelta) })
	st.maybeWriteThrough(sessionID, entryID, err)
	return snap, err
}

// maybeWriteThrough fires a fire-and-forget Character.update mirroring the entry's PV/PM
// back to the DB, when opt-in via WS_VITALS_WRITETHROUGH_LIVE. Mirrors maybeLiveWriteThrough.
func (st *sessionStore) maybeWriteThrough(sessionID int64, entryID string, err error) {
	if err == nil && st.liveWriteThrough {
		go st.writeThroughVitals(sessionID, entryID)
	}
}

// writeThroughVitals mirrors the entry's current PV/PM onto the Character row, clamped to
// the fresh DB max (the entry's cached max can drift after a mid-session level-up). Only
// character entries with both vitals present; failures are logged, never surfaced.
func (st *sessionStore) writeThroughVitals(sessionID int64, entryID string) {
	// Same per-session lock as persist: serialize write-throughs so two concurrent vitals
	// edits can't land out of order. Whoever runs last reads the CURRENT entry (below) and
	// writes it, so the DB converges to the latest value.
	pm := st.persistLock(sessionID)
	pm.Lock()
	defer pm.Unlock()

	st.mu.Lock()
	var entry *InitiativeEntry
	if s := st.states[sessionID]; s != nil {
		if idx := findEntryIndex(s, entryID); idx >= 0 {
			e := s.Initiative[idx]
			entry = &e
		}
	}
	st.mu.Unlock()
	if entry == nil || entry.CharacterID == nil || entry.HpCurrent == nil || entry.MpCurrent == nil {
		return
	}
	ctx := context.Background()
	rows, err := st.q.ListCharacterMaxes(ctx, []int64{*entry.CharacterID})
	if err != nil || len(rows) == 0 {
		if err != nil {
			log.Printf("session %d write-through: load maxes failed (%v)", sessionID, err)
		}
		return
	}
	m := rows[0]
	if err := st.q.SetVitalsCurrent(ctx, sqlcgen.SetVitalsCurrentParams{
		HpCurrent: clampVital(*entry.HpCurrent, &m.Hpmax),
		MpCurrent: clampVital(*entry.MpCurrent, &m.Mpmax),
		UpdatedAt: nowISO(), ID: *entry.CharacterID,
	}); err != nil {
		log.Printf("session %d write-through failed for character %d: %v", sessionID, *entry.CharacterID, err)
	}
}

// load hydrates the session from Session.runtimeState on first access, then serves the
// cached copy. Mirrors SessionStateService.load/hydrate.
func (st *sessionStore) load(ctx context.Context, sessionID int64) (*SessionRuntimeState, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if s := st.states[sessionID]; s != nil {
		return cloneState(s), nil
	}
	sess, err := st.q.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	s := parseRuntimeBlob(sess.Runtimestate)
	st.states[sessionID] = s
	return cloneState(s), nil
}

// parseRuntimeBlob parses a persisted blob, falling back to an empty tracker on malformed
// input (mirrors the zod safeParse fallback). Blobs are always full (our Marshal + the
// column default carry round/turnIndex), so partial-blob defaulting isn't needed.
func parseRuntimeBlob(blob string) *SessionRuntimeState {
	if blob == "" {
		return emptyRuntimeState()
	}
	var parsed SessionRuntimeState
	if err := json.Unmarshal([]byte(blob), &parsed); err != nil {
		return emptyRuntimeState()
	}
	if parsed.Initiative == nil {
		parsed.Initiative = []InitiativeEntry{}
	}
	return &parsed
}

// persist serializes the current state to Session.runtimeState. Fire-and-forget: never
// returns an error — it returns (dirty, changed), where `changed` is true only when the
// persistence health flipped since the last persist, so the gateway can broadcast
// `persistence-warning` exactly on the transitions. The store is the single owner of the
// dirty flag (pruned by forget) — the gateway no longer tracks it. Code-review finding.
//
// Serialized so overlapping persists for one session write in order: whichever runs last
// snapshots the newest state, so the DB converges to the latest instead of a stale capture.
func (st *sessionStore) persist(ctx context.Context, sessionID int64) (dirty, changed bool) {
	pm := st.persistLock(sessionID)
	pm.Lock()
	defer pm.Unlock()

	st.mu.Lock()
	s := st.states[sessionID]
	if s == nil {
		st.mu.Unlock()
		return false, false
	}
	blob, _ := json.Marshal(cloneState(s))
	st.mu.Unlock()

	err := st.q.ResetSessionTracker(ctx, sqlcgen.ResetSessionTrackerParams{
		RuntimeState: string(blob), UpdatedAt: nowISO(), ID: sessionID,
	})

	st.mu.Lock()
	defer st.mu.Unlock()
	prev := st.dirty[sessionID] // absent ⇒ false (healthy), matching the Nest `?? false`
	dirty = err != nil
	changed = prev != dirty
	if dirty {
		st.dirty[sessionID] = true
		log.Printf("session %d: persist failed (%v); marked dirty for retry", sessionID, err)
	} else {
		delete(st.dirty, sessionID)
	}
	return dirty, changed
}

func (st *sessionStore) isDirty(sessionID int64) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.dirty[sessionID]
}

// forget drops a session from memory (e.g. its DB row was deleted).
// forget drops a session's in-memory tracker (e.g. on clear-tracker). It does NOT clear
// the dirty flag: that would swallow the dirty→healthy recovery — a session left dirty
// must still emit persistence-warning{dirty:false} on the next successful persist. The
// dirty map self-prunes on that success, so it stays small (only currently-dirty sessions).
func (st *sessionStore) forget(sessionID int64) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.states, sessionID)
}

// refreshCharacterMaxes refreshes hpMax/mpMax on every entry carrying a characterId from
// the DB rows (ceilings only; current untouched) so a mid-session level-up isn't capped at
// the stale max. Best-effort like the Nest version: a DB blip logs and returns the current
// snapshot rather than failing the get-session-state pull.
func (st *sessionStore) refreshCharacterMaxes(ctx context.Context, sessionID int64) *SessionRuntimeState {
	st.mu.Lock()
	ids := uniqueCharacterIDs(st.getOrCreateLocked(sessionID))
	st.mu.Unlock()
	if len(ids) == 0 {
		return st.getState(sessionID)
	}
	rows, err := st.q.ListCharacterMaxes(ctx, ids)
	if err != nil {
		log.Printf("session %d: hpMax refresh failed (%v)", sessionID, err)
		return st.getState(sessionID)
	}
	maxes := make(map[int64]sqlcgen.ListCharacterMaxesRow, len(rows))
	for _, r := range rows {
		maxes[r.ID] = r
	}
	st.mu.Lock()
	defer st.mu.Unlock()
	s := st.getOrCreateLocked(sessionID)
	for i := range s.Initiative {
		e := &s.Initiative[i]
		if e.CharacterID == nil {
			continue
		}
		if fresh, ok := maxes[*e.CharacterID]; ok {
			e.HpMax = ptrInt64(fresh.Hpmax)
			e.MpMax = ptrInt64(fresh.Mpmax)
		}
	}
	return cloneState(s)
}

func uniqueCharacterIDs(s *SessionRuntimeState) []int64 {
	seen := map[int64]bool{}
	ids := []int64{}
	for _, e := range s.Initiative {
		if e.CharacterID != nil && !seen[*e.CharacterID] {
			seen[*e.CharacterID] = true
			ids = append(ids, *e.CharacterID)
		}
	}
	return ids
}
