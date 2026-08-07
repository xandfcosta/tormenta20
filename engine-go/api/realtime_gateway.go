package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"

	socket "github.com/zishang520/socket.io/servers/socket/v3"
)

// realtimeGateway is the socket.io transport for the session tracker — the thin glue that
// binds the front's socket.io-client to the transport-agnostic domain (sessionForCaller,
// sessionStore, presenceRegistry, effects). Mirrors RealtimeGateway; nothing here authors a
// rule. Built once by SocketHandler and mounted at /socket.io/.
type realtimeGateway struct {
	s  *Server
	io *socket.Server
	// lastDirty tracks the last-broadcast persistence status per session so
	// `persistence-warning` only fires when the flag flips.
	mu        sync.Mutex
	lastDirty map[int64]bool
}

// socketData is stashed on each authenticated socket: the resolved user and the caller's
// role in the session (refreshed per session-scoped message).
type socketData struct {
	user AuthUser
	role string
}

// SocketHandler builds the socket.io server wired to this Server's domain and returns the
// http.Handler to mount at /socket.io/. Called once by cmd/api.
func (s *Server) SocketHandler() http.Handler {
	g := &realtimeGateway{s: s, io: socket.NewServer(nil, nil), lastDirty: map[int64]bool{}}
	g.io.On("connection", func(clients ...any) {
		if sock, ok := clients[0].(*socket.Socket); ok {
			g.onConnect(sock)
		}
	})
	return g.io.ServeHandler(nil)
}

// onConnect authenticates the handshake (same cookie/JWT as HTTP); a bad handshake gets an
// `unauthorized` emit + disconnect. Mirrors RealtimeGateway.handleConnection.
func (g *realtimeGateway) onConnect(sock *socket.Socket) {
	user, err := g.authenticate(sock)
	if err != nil {
		_ = sock.Emit("unauthorized", map[string]any{"message": err.Error()})
		sock.Disconnect(true)
		return
	}
	sock.SetData(&socketData{user: user})
	sock.On("join-session", func(args ...any) { g.onJoin(sock, args) })
	sock.On("leave-session", func(args ...any) { g.onLeave(sock, args) })
	sock.On("get-session-state", func(args ...any) { g.onGetState(sock, args) })
	sock.On("disconnect", func(...any) { g.onDisconnect(sock) })
}

func (g *realtimeGateway) authenticate(sock *socket.Socket) (AuthUser, error) {
	h := sock.Handshake()
	authToken := ""
	if h.Auth != nil {
		if t, ok := h.Auth["token"].(string); ok {
			authToken = t
		}
	}
	authHeader := headerValue(h.Headers, "Authorization")
	cookieHeader := headerValue(h.Headers, "Cookie")
	return g.s.authenticateHandshake(context.Background(), authToken, authHeader, cookieHeader)
}

// onJoin resolves session access (stashing the role), joins the room, and tracks presence.
// Ack: {joined: room}. Mirrors RealtimeGateway.joinSession.
func (g *realtimeGateway) onJoin(sock *socket.Socket, args []any) {
	body, ack := bodyOf(args), ackOf(args)
	campaignID, ok1 := intField(body, "campaignId")
	sessionID, ok2 := intField(body, "sessionId")
	if !ok1 || !ok2 {
		g.wsError(sock, "campaignId and sessionId are required integers")
		return
	}
	data := sockData(sock)
	_, role, _, err := g.s.sessionForCaller(context.Background(), data.user.ID, campaignID, sessionID)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	data.role = role
	room := sessionRoomName(sessionID)
	sock.Join(socket.Room(room))
	g.trackPresence(sock, sessionID)
	ackOK(ack, map[string]any{"joined": room})
}

// onLeave leaves the room + drops presence. Ack: {left: room}. Mirrors leaveSession.
func (g *realtimeGateway) onLeave(sock *socket.Socket, args []any) {
	body, ack := bodyOf(args), ackOf(args)
	sessionID, ok := intField(body, "sessionId")
	if !ok {
		g.wsError(sock, "sessionId is required")
		return
	}
	room := sessionRoomName(sessionID)
	sock.Leave(socket.Room(room))
	if roster, changed := g.s.presence.leave(sessionID, string(sock.Id())); changed {
		g.emitPresence(sessionID, roster)
	}
	ackOK(ack, map[string]any{"left": room})
}

// onGetState hydrates the tracker (first pull restores the persisted state), refreshes
// character maxes from the DB, and acks the full state. Mirrors getSessionState.
func (g *realtimeGateway) onGetState(sock *socket.Socket, args []any) {
	body, ack := bodyOf(args), ackOf(args)
	campaignID, ok1 := intField(body, "campaignId")
	sessionID, ok2 := intField(body, "sessionId")
	if !ok1 || !ok2 {
		g.wsError(sock, "campaignId and sessionId are required integers")
		return
	}
	data := sockData(sock)
	_, role, _, err := g.s.sessionForCaller(context.Background(), data.user.ID, campaignID, sessionID)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	data.role = role
	if _, err := g.s.sessions.load(context.Background(), sessionID); err != nil {
		g.wsError(sock, "Could not load session state")
		return
	}
	ackOK(ack, g.s.sessions.refreshCharacterMaxes(context.Background(), sessionID))
}

// onDisconnect drops the socket from every room and broadcasts each changed roster.
func (g *realtimeGateway) onDisconnect(sock *socket.Socket) {
	for _, sr := range g.s.presence.disconnect(string(sock.Id())) {
		g.emitPresence(sr.sessionID, sr.roster)
	}
}

func (g *realtimeGateway) trackPresence(sock *socket.Socket, sessionID int64) {
	data := sockData(sock)
	name := data.user.Email
	if data.user.Name != nil && *data.user.Name != "" {
		name = *data.user.Name
	}
	role := data.role
	if role == "" {
		role = "player"
	}
	roster := g.s.presence.join(sessionID, string(sock.Id()), PresenceUser{UserID: data.user.ID, Name: name, Role: role})
	g.emitPresence(sessionID, roster)
}

func (g *realtimeGateway) emitPresence(sessionID int64, roster []PresenceUser) {
	g.io.To(socket.Room(sessionRoomName(sessionID))).Emit("presence", map[string]any{
		"sessionId": sessionID, "users": roster,
	})
}

// wsError signals a rejected message the way Nest's WsException filter does — an
// `exception` event — instead of acking (the front's mutation emits pass no callback, and
// corrupting a get-state ack with an error payload would poison the tracker).
func (g *realtimeGateway) wsError(sock *socket.Socket, message string) {
	_ = sock.Emit("exception", map[string]any{"status": "error", "message": message})
}

// ── small transport helpers ──────────────────────────────────────────

func sessionRoomName(sessionID int64) string {
	return "session:" + strconv.FormatInt(sessionID, 10)
}

func sockData(sock *socket.Socket) *socketData {
	d, _ := sock.Data().(*socketData)
	return d
}

func bodyOf(args []any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	m, _ := args[0].(map[string]any)
	return m
}

// ackOf returns the client's ack callback (the last arg when present), or nil.
func ackOf(args []any) socket.Ack {
	if len(args) == 0 {
		return nil
	}
	if cb, ok := args[len(args)-1].(socket.Ack); ok {
		return cb
	}
	return nil
}

func ackOK(ack socket.Ack, payload any) {
	if ack != nil {
		ack([]any{payload}, nil)
	}
}

// intField reads an integer body field (JSON numbers arrive as float64).
func intField(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	switch v := m[key].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	}
	return 0, false
}

// headerValue reads a header whose value is string or []string (IncomingHttpHeaders).
func headerValue(h map[string]any, key string) string {
	switch v := h[key].(type) {
	case string:
		return v
	case []string:
		return strings.Join(v, "; ")
	}
	return ""
}
