package api

import (
	"sort"
	"sync"
)

// PresenceUser is one connected participant in a session room. Rosters dedupe by userId
// (multiple browser tabs of the same user collapse to one chip; a user counts as GM if any
// of their connections is a GM). Mirrors the frontend PresenceUser + the `presence` broadcast.
type PresenceUser struct {
	UserID int64  `json:"userId"`
	Name   string `json:"name"`
	Role   string `json:"role"` // "gm" | "player"
}

// sessionRoster pairs a session with its recomputed roster — one per room a disconnect
// touched, so the gateway can broadcast each.
type sessionRoster struct {
	sessionID int64
	roster    []PresenceUser
}

// presenceRegistry is the in-memory "who's online" tracker for session rooms. Pure
// bookkeeping (the SSE hub owns the broadcast); rebuilt from live connections, so a
// server restart starts empty and refills as clients reconnect. Mirrors PresenceRegistry.
type presenceRegistry struct {
	mu sync.Mutex
	// sessionID → (connID → presence)
	presence map[int64]map[string]PresenceUser
	// connID → set of sessions it joined, so a disconnect cleans every room without
	// relying on a transport library's post-disconnect room set.
	connSessions map[string]map[int64]bool
}

func newPresenceRegistry() *presenceRegistry {
	return &presenceRegistry{
		presence:       map[int64]map[string]PresenceUser{},
		connSessions: map[string]map[int64]bool{},
	}
}

// join records a connection in a session and returns the session's deduped roster.
func (p *presenceRegistry) join(sessionID int64, connID string, user PresenceUser) []PresenceUser {
	p.mu.Lock()
	defer p.mu.Unlock()
	room := p.presence[sessionID]
	if room == nil {
		room = map[string]PresenceUser{}
		p.presence[sessionID] = room
	}
	room[connID] = user
	sessions := p.connSessions[connID]
	if sessions == nil {
		sessions = map[int64]bool{}
		p.connSessions[connID] = sessions
	}
	sessions[sessionID] = true
	return p.rosterLocked(sessionID)
}

// leave removes a connection from one session. Returns the new roster and true, or (nil,false)
// when the socket wasn't present (nothing to announce).
func (p *presenceRegistry) leave(sessionID int64, connID string) ([]PresenceUser, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if sessions := p.connSessions[connID]; sessions != nil {
		delete(sessions, sessionID)
	}
	if !p.dropLocked(connID, sessionID) {
		return nil, false
	}
	return p.rosterLocked(sessionID), true
}

// disconnect removes a socket from every session it joined, returning one {sessionID,
// roster} per room that actually changed so the gateway can broadcast each.
func (p *presenceRegistry) disconnect(connID string) []sessionRoster {
	p.mu.Lock()
	defer p.mu.Unlock()
	sessions := p.connSessions[connID]
	if sessions == nil {
		return nil
	}
	changed := []sessionRoster{}
	for _, sessionID := range sortedInt64Keys(sessions) {
		if p.dropLocked(connID, sessionID) {
			changed = append(changed, sessionRoster{sessionID: sessionID, roster: p.rosterLocked(sessionID)})
		}
	}
	delete(p.connSessions, connID)
	return changed
}

// dropLocked removes a socket from a session's room, cleaning up the empty room. Returns
// false when the socket wasn't there. Caller holds mu.
func (p *presenceRegistry) dropLocked(connID string, sessionID int64) bool {
	room := p.presence[sessionID]
	if room == nil {
		return false
	}
	if _, ok := room[connID]; !ok {
		return false
	}
	delete(room, connID)
	if len(room) == 0 {
		delete(p.presence, sessionID)
	}
	return true
}

// rosterLocked returns the session's roster deduped by userId — multi-tab collapses to one,
// and a user counts as GM if any of their sockets is a GM. Sorted by userId for a
// deterministic broadcast (order is cosmetic, but a stable one keeps diffs readable).
// Caller holds mu.
func (p *presenceRegistry) rosterLocked(sessionID int64) []PresenceUser {
	room := p.presence[sessionID]
	byUser := make(map[int64]PresenceUser, len(room))
	for _, u := range room {
		existing, ok := byUser[u.UserID]
		if !ok || (existing.Role != "gm" && u.Role == "gm") {
			byUser[u.UserID] = u
		}
	}
	users := make([]PresenceUser, 0, len(byUser))
	for _, u := range byUser {
		users = append(users, u)
	}
	sort.Slice(users, func(i, j int) bool { return users[i].UserID < users[j].UserID })
	return users
}

func sortedInt64Keys(m map[int64]bool) []int64 {
	keys := make([]int64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}
