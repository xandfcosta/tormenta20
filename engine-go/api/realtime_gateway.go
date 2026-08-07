package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	socket "github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
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
	// CORS must reflect the browser's Origin with credentials — the front connects
	// through the Vite proxy, so the forwarded Origin (:5173) differs from the Go
	// port and engine.io otherwise mishandles the WS handshake. Mirrors the Nest
	// gateway's `@WebSocketGateway({ cors: { origin: true, credentials: true } })`.
	opts := socket.DefaultServerOptions()
	opts.SetCors(&types.Cors{Origin: "*", Credentials: true})
	g := &realtimeGateway{s: s, io: socket.NewServer(nil, opts), lastDirty: map[int64]bool{}}
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
	sock.On("initiative-add", func(args ...any) { g.onInitiativeAdd(sock, args) })
	sock.On("initiative-self", func(args ...any) { g.onInitiativeSelf(sock, args) })
	sock.On("initiative-update", func(args ...any) { g.onInitiativeUpdate(sock, args) })
	sock.On("initiative-remove", func(args ...any) { g.onInitiativeRemove(sock, args) })
	sock.On("initiative-next-turn", func(args ...any) { g.onNextTurn(sock, args) })
	sock.On("initiative-reset", func(args ...any) { g.onResetInitiative(sock, args) })
	sock.On("initiative-populate", func(args ...any) { g.onPopulate(sock, args) })
	sock.On("vitals-patch", func(args ...any) { g.onVitalsPatch(sock, args) })
	sock.On("vitals-delta", func(args ...any) { g.onVitalsDelta(sock, args) })
	sock.On("session-rest", func(args ...any) { g.onSessionRest(sock, args) })
	sock.On("apply-effect", func(args ...any) { g.onApplyEffect(sock, args) })
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

// msgCtx is the resolved context of a session-scoped message: the caller, the ids, the
// per-message-resolved role, the raw body, and the client's ack callback.
type msgCtx struct {
	userID     int64
	campaignID int64
	sessionID  int64
	role       string
	body       map[string]any
	ack        socket.Ack
}

// access re-resolves the caller's role for a session-scoped message (per-message check so a
// spoofed sessionId on a stale socket can't hijack another table), stashing it on the
// socket. Returns ok=false after emitting the error. Mirrors assertSessionAccess.
func (g *realtimeGateway) access(sock *socket.Socket, args []any) (msgCtx, bool) {
	body, ack := bodyOf(args), ackOf(args)
	campaignID, ok1 := intField(body, "campaignId")
	sessionID, ok2 := intField(body, "sessionId")
	if !ok1 || !ok2 {
		g.wsError(sock, "campaignId and sessionId are required integers")
		return msgCtx{}, false
	}
	data := sockData(sock)
	_, role, _, err := g.s.sessionForCaller(context.Background(), data.user.ID, campaignID, sessionID)
	if err != nil {
		g.wsError(sock, err.Error())
		return msgCtx{}, false
	}
	data.role = role
	return msgCtx{userID: data.user.ID, campaignID: campaignID, sessionID: sessionID, role: role, body: body, ack: ack}, true
}

// requireGm gates initiative control (add/update/remove/next/reset/populate) to the GM.
// Mirrors assertGm.
func (g *realtimeGateway) requireGm(sock *socket.Socket, role string) bool {
	if role != "gm" {
		g.wsError(sock, "Only the campaign GM can control initiative")
		return false
	}
	return true
}

// onJoin resolves session access (stashing the role), joins the room, and tracks presence.
// Ack: {joined: room}. Mirrors RealtimeGateway.joinSession.
func (g *realtimeGateway) onJoin(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	room := sessionRoomName(ctx.sessionID)
	sock.Join(socket.Room(room))
	g.trackPresence(sock, ctx.sessionID)
	ackOK(ctx.ack, map[string]any{"joined": room})
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
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	if _, err := g.s.sessions.load(context.Background(), ctx.sessionID); err != nil {
		g.wsError(sock, "Could not load session state")
		return
	}
	ackOK(ctx.ack, g.s.sessions.refreshCharacterMaxes(context.Background(), ctx.sessionID))
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

// onInitiativeAdd (GM) materializes an entry (NPC or character) and appends it.
func (g *realtimeGateway) onInitiativeAdd(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryBody, _ := ctx.body["entry"].(map[string]any)
	if entryBody == nil {
		g.wsError(sock, "entry is required")
		return
	}
	entry, err := g.materializeEntry(ctx.userID, ctx.campaignID, entryBody)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.addInitiativeEntry(ctx.sessionID, entry)
	})
}

// onInitiativeSelf lets a player roll their OWN initiative (NOT GM-gated — resolveCombatant
// enforces they own the character). Upserts by characterId so a re-roll updates in place.
func (g *realtimeGateway) onInitiativeSelf(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	if _, has := intField(ctx.body, "characterId"); !has {
		g.wsError(sock, "characterId is required")
		return
	}
	entry, err := g.materializeEntry(ctx.userID, ctx.campaignID, ctx.body)
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.upsertInitiativeEntry(ctx.sessionID, entry)
	})
}

// onInitiativeUpdate (GM) patches an entry's fields.
func (g *realtimeGateway) onInitiativeUpdate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	patch := parseEntryPatch(ctx.body["patch"])
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.updateInitiativeEntry(ctx.sessionID, entryID, patch)
	})
}

// onInitiativeRemove (GM) drops an entry.
func (g *realtimeGateway) onInitiativeRemove(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.removeInitiativeEntry(ctx.sessionID, entryID)
	})
}

// onNextTurn (GM) advances to the next combatant.
func (g *realtimeGateway) onNextTurn(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.nextTurn(ctx.sessionID)
	})
}

// onResetInitiative (GM) clears the tracker.
func (g *realtimeGateway) onResetInitiative(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.reset(ctx.sessionID)
	})
}

// onPopulate (GM) pulls the campaign's player characters into the tracker (initiative 0,
// live vitals), skipping any already present. Idempotent. Mirrors initiativePopulate.
func (g *realtimeGateway) onPopulate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	combatants, err := g.s.listPlayerCombatants(context.Background(), ctx.campaignID)
	if err != nil {
		g.wsError(sock, "Could not load party")
		return
	}
	existing := map[int64]bool{}
	for _, e := range g.s.sessions.getState(ctx.sessionID).Initiative {
		if e.CharacterID != nil {
			existing[*e.CharacterID] = true
		}
	}
	var state *SessionRuntimeState
	for _, c := range combatants {
		if existing[c.characterID] {
			continue
		}
		cid, hpc, hpm, mpc, mpm := c.characterID, c.hpCurrent, c.hpMax, c.mpCurrent, c.mpMax
		st, err := g.s.sessions.addInitiativeEntry(ctx.sessionID, InitiativeEntry{
			Label: c.name, Initiative: 0, Type: "character", CharacterID: &cid,
			HpCurrent: &hpc, HpMax: &hpm, MpCurrent: &mpc, MpMax: &mpm,
		})
		if err != nil {
			g.wsError(sock, err.Error())
			break
		}
		state = st
	}
	if state == nil {
		state = g.s.sessions.getState(ctx.sessionID)
	}
	g.emitSessionState(ctx.sessionID, state)
	ackOK(ctx.ack, state)
}

// onVitalsPatch sets absolute hp/mp on an entry. The GM edits anyone; a player only their
// own character; NPC vitals are GM-only (assertVitalsEditable). Mirrors vitalsPatch.
func (g *realtimeGateway) onVitalsPatch(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	if err := g.assertVitalsEditable(ctx, entryID); err != nil {
		g.wsError(sock, err.Error())
		return
	}
	patch, _ := ctx.body["patch"].(map[string]any)
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.patchVitals(ctx.sessionID, entryID, optInt(patch, "hpCurrent"), optInt(patch, "mpCurrent"))
	})
}

// onVitalsDelta applies an hp/mp delta to an entry (same authorization as patch). Mirrors
// vitalsDelta.
func (g *realtimeGateway) onVitalsDelta(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	if err := g.assertVitalsEditable(ctx, entryID); err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.mutateAndBroadcast(sock, ctx, func() (*SessionRuntimeState, error) {
		return g.s.sessions.deltaVitals(ctx.sessionID, entryID, optInt(ctx.body, "hpDelta"), optInt(ctx.body, "mpDelta"))
	})
}

// assertVitalsEditable authorizes a vitals mutation: the GM edits any combatant; a player
// only their own character; NPC entries are GM-only. Mirrors assertVitalsEditable.
func (g *realtimeGateway) assertVitalsEditable(ctx msgCtx, entryID string) error {
	if ctx.role == "gm" {
		return nil
	}
	state := g.s.sessions.getState(ctx.sessionID)
	idx := findEntryIndex(state, entryID)
	if idx < 0 {
		return errors.New("Entry " + entryID + " not found")
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		return errors.New("Only the GM can edit NPC vitals")
	}
	_, err := g.s.assertCharacterOwner(context.Background(), ctx.userID, *entry.CharacterID)
	return err
}

// onSessionRest (GM) rests every member character: end-scene expires scene effects;
// end-day also expires day effects AND restores PV/PM, mirrored onto the live tracker.
// Broadcasts `session-rest` + (if anyone healed) `session-state`. Mirrors sessionRest.
func (g *realtimeGateway) onSessionRest(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	scope := stringField(ctx.body, "scope")
	condition := stringField(ctx.body, "condition")
	if condition == "" {
		condition = "normal"
	}
	charIDs, err := g.s.listMemberCharacterIds(context.Background(), ctx.campaignID)
	if err != nil {
		g.wsError(sock, "Could not load campaign members")
		return
	}
	user := sockData(sock).user
	healed := 0
	for _, cid := range charIDs {
		if scope == "day" {
			if _, err := g.s.endDay(context.Background(), user, cid); err != nil {
				continue
			}
			vitals, _, err := g.s.restVitals(context.Background(), user, cid, condition)
			if err != nil {
				continue
			}
			g.mirrorVitalsToTracker(ctx.sessionID, cid, vitals)
			healed++
		} else {
			_, _ = g.s.endScene(context.Background(), user, cid)
		}
	}
	if healed > 0 {
		g.emitSessionState(ctx.sessionID, g.s.sessions.getState(ctx.sessionID))
	}
	g.io.To(socket.Room(sessionRoomName(ctx.sessionID))).Emit("session-rest", map[string]any{
		"sessionId": ctx.sessionID, "scope": scope, "condition": condition,
	})
	ackOK(ctx.ack, map[string]any{"rested": scope, "characters": len(charIDs), "healed": healed})
}

// mirrorVitalsToTracker copies freshly-persisted PV/PM onto the matching live tracker entry
// (if the character is in the current initiative) so bars update without a reload.
func (g *realtimeGateway) mirrorVitalsToTracker(sessionID, characterID int64, vitals restedVitals) {
	for _, e := range g.s.sessions.getState(sessionID).Initiative {
		if e.CharacterID != nil && *e.CharacterID == characterID {
			hp, mp := vitals.hpCurrent, vitals.mpCurrent
			_, _ = g.s.sessions.patchVitals(sessionID, e.ID, &hp, &mp)
			return
		}
	}
}

// onApplyEffect (GM) applies a spell buff to a combatant's character and notifies the room
// so clients holding that sheet refetch (activeEffects aren't in tracker state). Mirrors
// applyEffect.
func (g *realtimeGateway) onApplyEffect(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	entryID := stringField(ctx.body, "entryId")
	spellID := stringField(ctx.body, "spellId")
	if entryID == "" {
		g.wsError(sock, "entryId is required")
		return
	}
	if spellID == "" {
		g.wsError(sock, "spellId is required")
		return
	}
	state := g.s.sessions.getState(ctx.sessionID)
	idx := findEntryIndex(state, entryID)
	if idx < 0 {
		g.wsError(sock, "Entry "+entryID+" not found")
		return
	}
	entry := state.Initiative[idx]
	if entry.CharacterID == nil {
		g.wsError(sock, "Only character entries can receive spell effects")
		return
	}
	var scope *string
	if s := stringField(ctx.body, "scope"); s != "" {
		scope = &s
	}
	if _, _, err := g.s.applySpellBuffEffect(context.Background(), *entry.CharacterID, spellID, scope); err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.io.To(socket.Room(sessionRoomName(ctx.sessionID))).Emit("effect-applied", map[string]any{
		"sessionId": ctx.sessionID, "characterId": *entry.CharacterID, "spellId": spellID,
	})
	ackOK(ctx.ack, map[string]any{"applied": spellID, "characterId": *entry.CharacterID})
}

// mutateAndBroadcast runs a store mutation, broadcasts the new state (+ persists), and acks.
// The shared tail of every initiative handler.
func (g *realtimeGateway) mutateAndBroadcast(sock *socket.Socket, ctx msgCtx, mutate func() (*SessionRuntimeState, error)) {
	state, err := mutate()
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.emitSessionState(ctx.sessionID, state)
	ackOK(ctx.ack, state)
}

// emitSessionState broadcasts the tracker to the room and kicks off a fire-and-forget
// persist. Mirrors RealtimeGateway.emitSessionState.
func (g *realtimeGateway) emitSessionState(sessionID int64, state *SessionRuntimeState) {
	g.io.To(socket.Room(sessionRoomName(sessionID))).Emit("session-state", state)
	go g.persistAndWarn(sessionID)
}

// persistAndWarn persists the state and broadcasts `persistence-warning` only when the
// dirty flag flips (first failure, or a retry that recovered). Mirrors the persist().then
// block. Absent previous status defaults to false (matches the Nest `?? false`).
func (g *realtimeGateway) persistAndWarn(sessionID int64) {
	dirty := g.s.sessions.persist(context.Background(), sessionID)
	g.mu.Lock()
	if g.lastDirty[sessionID] == dirty {
		g.mu.Unlock()
		return
	}
	g.lastDirty[sessionID] = dirty
	g.mu.Unlock()
	g.io.To(socket.Room(sessionRoomName(sessionID))).Emit("persistence-warning", map[string]any{
		"sessionId": sessionID, "dirty": dirty,
	})
}

// materializeEntry resolves an initiative payload into a concrete entry. NPCs (no
// characterId) require label + initiative; character entries pull name/vitals via
// resolveCombatant (membership + owner-or-GM enforced there), with optional client
// overrides. Mirrors RealtimeGateway.materializeEntry.
func (g *realtimeGateway) materializeEntry(callerID, campaignID int64, input map[string]any) (InitiativeEntry, error) {
	charID, hasChar := intField(input, "characterId")
	initiative, hasInit := intField(input, "initiative")
	if !hasChar {
		label := strings.TrimSpace(stringField(input, "label"))
		if label == "" {
			return InitiativeEntry{}, errors.New("entry.label is required for NPC entries")
		}
		if !hasInit {
			return InitiativeEntry{}, errors.New("entry.initiative is required")
		}
		typ := "npc"
		if t := stringField(input, "type"); t != "" {
			typ = t
		}
		return InitiativeEntry{Label: label, Initiative: int(initiative), Type: typ}, nil
	}
	if !hasInit {
		return InitiativeEntry{}, errors.New("entry.initiative is required")
	}
	stats, _, err := g.s.resolveCombatant(context.Background(), callerID, campaignID, charID)
	if err != nil {
		return InitiativeEntry{}, err
	}
	label := stats.name
	if l := strings.TrimSpace(stringField(input, "label")); l != "" {
		label = l
	}
	cid := charID
	return InitiativeEntry{
		Label: label, Initiative: int(initiative), Type: "character", CharacterID: &cid,
		HpCurrent: overrideInt(input, "hpCurrent", stats.hpCurrent),
		HpMax:     overrideInt(input, "hpMax", stats.hpMax),
		MpCurrent: overrideInt(input, "mpCurrent", stats.mpCurrent),
		MpMax:     overrideInt(input, "mpMax", stats.mpMax),
	}, nil
}

// parseEntryPatch reads an update patch from the raw body (only present fields become
// non-nil, so "leave unchanged" is distinct from "set to zero").
func parseEntryPatch(v any) entryPatch {
	m, _ := v.(map[string]any)
	p := entryPatch{}
	if m == nil {
		return p
	}
	if s, ok := m["label"].(string); ok {
		p.Label = &s
	}
	if i, ok := intField(m, "initiative"); ok {
		n := int(i)
		p.Initiative = &n
	}
	if s, ok := m["type"].(string); ok {
		p.Type = &s
	}
	if i, ok := intField(m, "characterId"); ok {
		p.CharacterID = &i
	}
	if i, ok := intField(m, "hpCurrent"); ok {
		p.HpCurrent = &i
	}
	if i, ok := intField(m, "hpMax"); ok {
		p.HpMax = &i
	}
	if i, ok := intField(m, "mpCurrent"); ok {
		p.MpCurrent = &i
	}
	if i, ok := intField(m, "mpMax"); ok {
		p.MpMax = &i
	}
	return p
}

// overrideInt returns the body's value for key when present, else def — as a pointer.
func overrideInt(m map[string]any, key string, def int64) *int64 {
	if v, ok := intField(m, key); ok {
		return ptrInt64(v)
	}
	return ptrInt64(def)
}

// optInt returns a pointer to the body's value for key, or nil when absent (so a vitals
// patch/delta only touches the fields the client actually sent).
func optInt(m map[string]any, key string) *int64 {
	if v, ok := intField(m, key); ok {
		return ptrInt64(v)
	}
	return nil
}

// ── small transport helpers ──────────────────────────────────────────

func sessionRoomName(sessionID int64) string {
	return "session:" + strconv.FormatInt(sessionID, 10)
}

func sockData(sock *socket.Socket) *socketData {
	d, _ := sock.Data().(*socketData)
	return d
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
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
