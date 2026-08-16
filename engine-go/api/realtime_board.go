package api

import (
	"context"

	socket "github.com/zishang520/socket.io/servers/socket/v3"
)

// Handlers do tabuleiro tático (ALE-124). Todos do MESTRE nesta fatia: quem move
// a própria peça entra na fatia do movimento, com gate por turno.
//
// O estado do tabuleiro sai por um evento PRÓPRIO (`board-state`) e não dentro do
// `session-state`: as duas taxas de escrita não têm relação, e juntá-las faria
// cada tique de PV carregar o tabuleiro inteiro para quem nem o abriu.

// onBoardOpen (mestre) abre o tabuleiro da sessão.
func (g *realtimeGateway) onBoardOpen(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	cols, _ := intField(ctx.body, "cols")
	rows, _ := intField(ctx.body, "rows")
	board, err := g.s.boards.open(context.Background(), ctx.sessionID,
		stringField(ctx.body, "place"), int(cols), int(rows), stringField(ctx.body, "terrain"))
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.emitBoardState(ctx.sessionID, board)
	ackOK(ctx.ack, boardForRole(ctx.role, board))
}

// onBoardClose (mestre) encerra o tabuleiro. O `nil` no broadcast é a mensagem:
// "esta sessão não tem tabuleiro" é estado de verdade, e não uma grade vazia.
func (g *realtimeGateway) onBoardClose(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.s.boards.close(context.Background(), ctx.sessionID)
	g.emitBoardState(ctx.sessionID, nil)
	ackOK(ctx.ack, map[string]any{"closed": true})
}

// onGetBoardState hidrata a tela de quem acabou de entrar. Responde COMO O PAPEL
// PODE VER — o ack é o segundo caminho do estado até a tela, e redigir só o
// broadcast deixaria a peça escondida sair inteira na primeira carga.
func (g *realtimeGateway) onGetBoardState(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	ackOK(ctx.ack, boardForRole(ctx.role, g.s.boards.get(context.Background(), ctx.sessionID)))
}

// onBoardTokenAdd (mestre) põe uma peça no tabuleiro.
func (g *realtimeGateway) onBoardTokenAdd(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.addToken(context.Background(), ctx.sessionID, parseBoardToken(ctx.body))
	})
}

// onBoardTokenRemove (mestre) tira uma peça do tabuleiro.
func (g *realtimeGateway) onBoardTokenRemove(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	tokenID := stringField(ctx.body, "tokenId")
	if tokenID == "" {
		g.wsError(sock, "tokenId is required")
		return
	}
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.removeToken(context.Background(), ctx.sessionID, tokenID)
	})
}

// onBoardTokenUpdate (mestre) altera rótulo, tamanho, posição ou o ocultamento.
func (g *realtimeGateway) onBoardTokenUpdate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	tokenID := stringField(ctx.body, "tokenId")
	if tokenID == "" {
		g.wsError(sock, "tokenId is required")
		return
	}
	patch := parseTokenPatch(ctx.body["patch"])
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.updateToken(context.Background(), ctx.sessionID, tokenID, patch)
	})
}

// onBoardPopulate (mestre) traz para o tabuleiro quem já está na iniciativa.
func (g *realtimeGateway) onBoardPopulate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.populate(context.Background(), ctx.sessionID, g.s.sessions.getState(ctx.sessionID))
	})
}

// mutateBoard é a cauda comum: muta, transmite, responde e persiste. Espelha o
// `mutateAndBroadcast` do rastreador — o erro sai como `exception`, nunca no ack.
func (g *realtimeGateway) mutateBoard(sock *socket.Socket, ctx msgCtx, mutate func() (*BoardState, error)) {
	board, err := mutate()
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	g.emitBoardState(ctx.sessionID, board)
	ackOK(ctx.ack, boardForRole(ctx.role, board))
}

// emitBoardState transmite o tabuleiro inteiro para a sala do mestre e a versão
// redigida para a dos jogadores, e persiste em seguida (fire-and-forget, como o
// rastreador: a mesa não para porque o disco piscou).
func (g *realtimeGateway) emitBoardState(sessionID int64, board *BoardState) {
	g.io.To(socket.Room(roleRoomName(sessionID, "gm"))).Emit("board-state", board)
	g.io.To(socket.Room(roleRoomName(sessionID, "player"))).Emit("board-state", boardForRole("player", board))
	go g.s.boards.persist(context.Background(), sessionID)
}

// parseBoardToken lê a peça do corpo da mensagem. Posição ausente é (0,0) — o
// canto é um lugar honesto para uma peça que ninguém posicionou.
func parseBoardToken(body map[string]any) BoardToken {
	token := BoardToken{
		Label: stringField(body, "label"),
		Kind:  stringField(body, "kind"),
	}
	if x, ok := intField(body, "x"); ok {
		token.X = int(x)
	}
	if y, ok := intField(body, "y"); ok {
		token.Y = int(y)
	}
	if footprint, ok := intField(body, "footprint"); ok {
		token.Footprint = int(footprint)
	}
	if entryID := stringField(body, "entryId"); entryID != "" {
		token.EntryID = &entryID
	}
	if characterID, ok := intField(body, "characterId"); ok {
		token.CharacterID = &characterID
	}
	if hidden, ok := body["hidden"].(bool); ok {
		token.Hidden = hidden
	}
	return token
}

// parseTokenPatch lê só os campos PRESENTES: ausente é "não mexa", não "zere".
func parseTokenPatch(raw any) tokenPatch {
	patch := tokenPatch{}
	m, ok := raw.(map[string]any)
	if !ok {
		return patch
	}
	if label, ok := m["label"].(string); ok {
		patch.Label = &label
	}
	if hidden, ok := m["hidden"].(bool); ok {
		patch.Hidden = &hidden
	}
	if footprint, ok := intField(m, "footprint"); ok {
		side := int(footprint)
		patch.Footprint = &side
	}
	if x, ok := intField(m, "x"); ok {
		col := int(x)
		patch.X = &col
	}
	if y, ok := intField(m, "y"); ok {
		row := int(y)
		patch.Y = &row
	}
	return patch
}
