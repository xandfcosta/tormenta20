package aovivo

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

// PresenceRegistry is the in-memory "who's online" tracker for session rooms. Pure
// bookkeeping (the SSE hub owns the broadcast); rebuilt from live connections, so a
// server restart starts empty and refills as clients reconnect. Mirrors PresenceRegistry.
type PresenceRegistry struct {
	Mu sync.Mutex
	// sessionID → (connID → presence)
	presence map[int64]map[string]PresenceUser
	// connID → set of sessions it joined, so a disconnect cleans every room without
	// relying on a transport library's post-disconnect room set.
	connSessions map[string]map[int64]bool
}

func NewPresenceRegistry() *PresenceRegistry {
	return &PresenceRegistry{
		presence:     map[int64]map[string]PresenceUser{},
		connSessions: map[string]map[int64]bool{},
	}
}

// Join records a connection in a session and returns the session's deduped roster.
func (p *PresenceRegistry) Join(sessionID int64, connID string, user PresenceUser) []PresenceUser {
	p.Mu.Lock()
	defer p.Mu.Unlock()
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

// Leave removes a connection from one session. Returns the new roster and true, or (nil,false)
// when the socket wasn't present (nothing to announce).
func (p *PresenceRegistry) Leave(sessionID int64, connID string) ([]PresenceUser, bool) {
	p.Mu.Lock()
	defer p.Mu.Unlock()
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
func (p *PresenceRegistry) disconnect(connID string) []sessionRoster {
	p.Mu.Lock()
	defer p.Mu.Unlock()
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
// false when the socket wasn't there. Caller holds Mu.
func (p *PresenceRegistry) dropLocked(connID string, sessionID int64) bool {
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
// Caller holds Mu.
func (p *PresenceRegistry) rosterLocked(sessionID int64) []PresenceUser {
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

// Roster é quem está na sessão AGORA.
//
// Existe porque a cena do mestre em Datastar precisa marcar quem está com a aba
// aberta, e ela lê o estado a cada desenho em vez de receber avisos — é a mesma
// diferença de desenho que o `sse_hub.go` registra entre publicar-o-clone e
// avisar-e-reler. O registro é do pacote e o campo é interno, então de fora
// ninguém alcançaria; sem isto a cena marcaria "ninguém online" por não
// conseguir olhar, que é o pior tipo de verde.
func (p *PresenceRegistry) Roster(sessionID int64) []PresenceUser {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	return p.rosterLocked(sessionID)
}
