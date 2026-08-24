package api

import (
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	socket "github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

// realtimeGateway is the socket.io transport for the session tracker — the thin glue that
// binds the front's socket.io-client to the transport-agnostic domain (sessionForCaller,
// sessionStore, presenceRegistry, effects). Mirrors RealtimeGateway; nothing here authors a
// rule. Built once by SocketHandler and mounted at /socket.io/. The initiative handlers live
// in realtime_initiative.go and the vitals/rest/apply handlers in realtime_vitals.go.
type realtimeGateway struct {
	s  *Server
	io *socket.Server
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
	// port and engine.io otherwise mishandles the WS handshake. Matches the
	// gateway's `@WebSocketGateway({ cors: { origin: true, credentials: true } })`.
	opts := socket.DefaultServerOptions()
	opts.SetCors(&types.Cors{Origin: "*", Credentials: true})
	g := &realtimeGateway{s: s, io: socket.NewServer(nil, opts)}
	// As DUAS linhas, e não uma escolha entre elas: elas fazem coisas
	// diferentes que o merge só empilhou no mesmo lugar. O `s.rt` é a
	// referência que a Mesa do piloto usa para ler o estado ao vivo (ALE-219);
	// o gancho abaixo é a voz do servidor na sala.
	s.rt = g
	// O gancho da ALE-245: é AQUI que o servidor ganha voz na sala, porque é
	// aqui que o gateway existe. Sem esta linha, uma condição aplicada pela
	// ficha não chega a mais ninguém.
	s.notifyCharacterChanged = g.emitCharacterChanged
	g.io.On("connection", func(clients ...any) {
		if sock, ok := clients[0].(*socket.Socket); ok {
			g.onConnect(sock)
		}
	})
	// A política de origem do socket passa a ser a MESMA do HTTP (ALE-158): o
	// `SetCors` acima reflete qualquer origem, e refletir com credenciais deixa
	// um site de terceiros abrir o handshake com o cookie do usuário.
	return s.guardSocketOrigin(g.io.ServeHandler(nil))
}

// guardSocketOrigin recusa o handshake de uma origem que a política não
// autoriza, ANTES de o engine.io ver a requisição.
//
// O `Router()` já tinha o guarda cuidadoso do ALE-119 e o socket não tinha
// nenhum, o que é contradição dentro do mesmo binário: o mesmo cookie que o
// HTTP protege abria uma sala ao vivo pelo socket (cross-site WebSocket
// hijacking). Numa LAN doméstica o risco prático é baixo, mas a política é uma
// só ou não é política.
func (s *Server) guardSocketOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.socketOriginAllowed(r.Header.Get("Origin"), r.Host) {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// socketOriginAllowed espelha a decisão do HTTP: com `CORS_ORIGIN` declarado
// (desenvolvimento, atrás do proxy do Vite), valem aquelas origens; sem ele
// (produção, onde este binário serve a própria SPA), vale a MESMA origem do
// pedido — e é por esse caminho que a mesa na LAN passa, porque quem abre
// `http://192.168.0.10:3001` manda essa origem e bate com o próprio Host
// (ALE-185).
//
// Requisição SEM `Origin` passa, e isso é deliberado: o navegador não manda
// esse cabeçalho em GET de mesma origem, que é justamente o transporte de
// polling do socket.io em produção — exigi-lo derrubaria o caminho normal. Quem
// continua guardando a sala nesse caso é o JWT do handshake, que não mudou.
func (s *Server) socketOriginAllowed(origin, host string) bool {
	if origin == "" {
		return true
	}
	if len(s.cfg.CORSOrigins) > 0 {
		return slices.Contains(s.cfg.CORSOrigins, origin)
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Host == host
}

// onConnect authenticates the handshake (same cookie/JWT as HTTP); a bad handshake gets an
// `unauthorized` emit + disconnect.
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
	sock.On("initiative-previous-turn", func(args ...any) { g.onPreviousTurn(sock, args) })
	sock.On("initiative-reset", func(args ...any) { g.onResetInitiative(sock, args) })
	sock.On("initiative-populate", func(args ...any) { g.onPopulate(sock, args) })
	// `session-*` e não `scene-*` (ALE-210): "cena" já é palavra ocupada no fio
	// pelo lado do tabuleiro, onde `board-place-scene` significa um mapa GUARDADO
	// da crônica. O ciclo vive no namespace da sessão, ao lado do `session-rest`,
	// que é o outro verbo de sessão inteira.
	sock.On("session-scene-start", func(args ...any) { g.onSceneStart(sock, args) })
	sock.On("session-scene-end", func(args ...any) { g.onSceneEnd(sock, args) })
	sock.On("vitals-patch", func(args ...any) { g.onVitalsPatch(sock, args) })
	sock.On("vitals-delta", func(args ...any) { g.onVitalsDelta(sock, args) })
	sock.On("session-rest", func(args ...any) { g.onSessionRest(sock, args) })
	sock.On("apply-effect", func(args ...any) { g.onApplyEffect(sock, args) })
	sock.On("board-open", func(args ...any) { g.onBoardOpen(sock, args) })
	sock.On("board-close", func(args ...any) { g.onBoardClose(sock, args) })
	sock.On("get-board-state", func(args ...any) { g.onGetBoardState(sock, args) })
	sock.On("board-as-player", func(args ...any) { g.onBoardAsPlayer(sock, args) })
	sock.On("board-token-add", func(args ...any) { g.onBoardTokenAdd(sock, args) })
	sock.On("board-token-remove", func(args ...any) { g.onBoardTokenRemove(sock, args) })
	sock.On("board-token-duplicate", func(args ...any) { g.onBoardTokenDuplicate(sock, args) })
	sock.On("board-token-update", func(args ...any) { g.onBoardTokenUpdate(sock, args) })
	sock.On("board-marker-add", func(args ...any) { g.onBoardMarkerAdd(sock, args) })
	sock.On("board-marker-update", func(args ...any) { g.onBoardMarkerUpdate(sock, args) })
	sock.On("board-marker-remove", func(args ...any) { g.onBoardMarkerRemove(sock, args) })
	sock.On("board-populate", func(args ...any) { g.onBoardPopulate(sock, args) })
	sock.On("board-terrain-paint", func(args ...any) { g.onBoardTerrainPaint(sock, args) })
	sock.On("board-places", func(args ...any) { g.onBoardPlaces(sock, args) })
	sock.On("board-reopen", func(args ...any) { g.onBoardReopen(sock, args) })
	sock.On("board-place-scene", func(args ...any) { g.onBoardPlaceScene(sock, args) })
	sock.On("board-place-save", func(args ...any) { g.onBoardPlaceSave(sock, args) })
	sock.On("board-place-remove", func(args ...any) { g.onBoardPlaceRemove(sock, args) })
	sock.On("board-move-propose", func(args ...any) { g.onBoardMovePropose(sock, args) })
	sock.On("board-move-commit", func(args ...any) { g.onBoardMoveCommit(sock, args) })
	sock.On("board-move-cancel", func(args ...any) { g.onBoardMoveCancel(sock, args) })
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
	_, role, _, err := g.s.sessionForCaller(context.Background(), data.user, campaignID, sessionID)
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
// Ack: {joined: room}.
func (g *realtimeGateway) onJoin(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	room := sessionRoomName(ctx.sessionID)
	sock.Join(socket.Room(room))
	// Sala por PAPEL: o estado sai duas vezes, inteiro para o mestre e redigido
	// para os jogadores, em vez de um payload por socket. O papel é o do momento
	// da entrada — quem for promovido no meio da sessão reentra (ALE-122).
	sock.Join(socket.Room(roleRoomName(ctx.sessionID, ctx.role)))
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
// character maxes from the DB, and acks the state COMO O PAPEL PODE VER — o ack é
// um segundo caminho do estado até a tela, e redigir só o broadcast deixaria o PV
// oculto sair inteiro na primeira carga.
func (g *realtimeGateway) onGetState(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	if _, err := g.s.sessions.load(context.Background(), ctx.sessionID); err != nil {
		g.wsError(sock, "Could not load session state")
		return
	}
	ackOK(ctx.ack, stateForRole(ctx.role, g.s.sessions.refreshCharacterMaxes(context.Background(), ctx.sessionID)))
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

// wsError signals a rejected message to the client — an
// `exception` event — instead of acking (the front's mutation emits pass no callback, and
// corrupting a get-state ack with an error payload would poison the tracker).
func (g *realtimeGateway) wsError(sock *socket.Socket, message string) {
	_ = sock.Emit("exception", map[string]any{"status": "error", "message": message})
}

// mutateAndBroadcast runs a store mutation, broadcasts the new state (+ persists), and acks.
// The shared tail of every initiative/vitals handler.
func (g *realtimeGateway) mutateAndBroadcast(sock *socket.Socket, ctx msgCtx, mutate func() (*SessionRuntimeState, error)) {
	state, err := mutate()
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.emitSessionState(ctx.sessionID, state)
	ackOK(ctx.ack, stateForRole(ctx.role, state))
}

// emitSessionState broadcasts the tracker to the room and kicks off a fire-and-forget
// persist.
func (g *realtimeGateway) emitSessionState(sessionID int64, state *SessionRuntimeState) {
	g.io.To(socket.Room(roleRoomName(sessionID, "gm"))).Emit("session-state", state)
	g.io.To(socket.Room(roleRoomName(sessionID, "player"))).
		Emit("session-state", redactForPlayers(state))
	go g.persistAndWarn(sessionID)
}

// emitCharacterChanged avisa TODA mesa ao vivo que tem este personagem na fila
// de que a ficha dele mudou (ALE-245).
//
// Manda só o id, e isso é escolha: cada cliente refaz a busca pela rota que ele
// já tem permissão de chamar, então a mensagem não carrega ficha de ninguém
// para dentro da sala. Quem nunca buscou aquele personagem não tem a query no
// cache e o aviso é um no-op — não vira requisição.
//
// O mesmo payload vai para as duas salas: saber que um combatente MUDOU não é
// informação secreta, e a fila já mostra que ele existe. O que é secreto está
// atrás da rota que cada um chama depois.
//
// Percorre as sessões vivas em vez de perguntar ao banco: é o que o store tem
// em memória, são poucas mesas com no máximo 50 entradas, e o custo é o de uma
// escrita de ficha — não de um laço quente.
func (g *realtimeGateway) emitCharacterChanged(characterID int64) {
	for _, sessionID := range g.s.sessions.liveSessionsWithCharacter(characterID) {
		payload := map[string]any{"characterId": characterID}
		g.io.To(socket.Room(roleRoomName(sessionID, "gm"))).Emit("character-changed", payload)
		g.io.To(socket.Room(roleRoomName(sessionID, "player"))).Emit("character-changed", payload)
	}
}

// persistAndWarn persists the state and broadcasts `persistence-warning` only when the
// store reports the dirty flag flipped (first failure, or a retry that recovered). The
// store owns the flag now — the gateway just relays the transitions.
func (g *realtimeGateway) persistAndWarn(sessionID int64) {
	dirty, changed := g.s.sessions.persist(context.Background(), sessionID)
	if !changed {
		return
	}
	g.io.To(socket.Room(sessionRoomName(sessionID))).Emit("persistence-warning", map[string]any{
		"sessionId": sessionID, "dirty": dirty,
	})
}

// ── small transport helpers ──────────────────────────────────────────

func sessionRoomName(sessionID int64) string {
	return "session:" + strconv.FormatInt(sessionID, 10)
}

// roleRoomName separa mestre de jogador dentro da sessão: é o que deixa o mesmo
// evento sair com conteúdos diferentes sem montar um payload por socket.
func roleRoomName(sessionID int64, role string) string {
	if role != "gm" {
		role = "player"
	}
	return sessionRoomName(sessionID) + ":" + role
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

// optInt returns a pointer to the body's value for key, or nil when absent (so a vitals
// patch/delta only touches the fields the client actually sent).
func optInt(m map[string]any, key string) *int64 {
	if v, ok := intField(m, key); ok {
		return ptrInt64(v)
	}
	return nil
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
