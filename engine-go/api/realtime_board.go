package api

import (
	"context"
	"log"

	socket "github.com/zishang520/socket.io/servers/socket/v3"

	"t20engine/engine"
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
	board := g.s.boards.open(context.Background(), ctx.sessionID,
		stringField(ctx.body, "place"), stringField(ctx.body, "terrain"))
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
	// O aviso de gravação vale para o fechar também: um DELETE que falha deixa
	// o tabuleiro fantasma no banco, e a mesa precisa saber (ALE-155).
	if dirty, changed := g.s.boards.close(context.Background(), ctx.sessionID); changed {
		g.warnPersistence(ctx.sessionID, dirty)
	}
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
	token, temPosicao := parseBoardToken(ctx.body)
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.addToken(context.Background(), ctx.sessionID, token, temPosicao)
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

// onBoardPopulate (mestre) traz para o tabuleiro quem já está na iniciativa, já
// com o deslocamento de cada personagem medido.
//
// O orçamento entra AQUI e não na tela: sem ele a peça nasceria sem alcance, e
// o jogador só descobriria quanto anda ao tentar mover. A conta é do motor
// (`Displacement.Total`), então a armadura pesada já chega descontada.
func (g *realtimeGateway) onBoardPopulate(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok || !g.requireGm(sock, ctx.role) {
		return
	}
	board, err := g.s.boards.populate(context.Background(), ctx.sessionID, g.s.sessions.getState(ctx.sessionID))
	if err != nil {
		g.wsError(sock, err.Error())
		return
	}
	if speeds := g.speedsForBoard(board); len(speeds) > 0 {
		if withSpeeds, err := g.s.boards.setSpeeds(context.Background(), ctx.sessionID, speeds); err == nil {
			board = withSpeeds
		}
	}
	g.emitBoardState(ctx.sessionID, board)
	ackOK(ctx.ack, boardForRole(ctx.role, board))
}

// speedsForBoard mede o deslocamento das peças de personagem que ainda não têm
// um. Só as que faltam: recomputar a ficha de todo mundo a cada "trazer o grupo"
// seria pagar caro por um número que não muda sozinho.
func (g *realtimeGateway) speedsForBoard(board *BoardState) map[string]int {
	speeds := map[string]int{}
	if board == nil {
		return speeds
	}
	for _, token := range board.Tokens {
		if token.CharacterID == nil || token.SpeedSquares > 0 {
			continue
		}
		if squares := g.speedSquaresFor(*token.CharacterID); squares > 0 {
			speeds[token.ID] = squares
		}
	}
	return speeds
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
	go g.persistBoardAndWarn(sessionID)
}

// persistBoardAndWarn grava e avisa a mesa quando a gravação começa (ou deixa)
// de falhar — o MESMO evento do rastreador, então a tela que já mostra o aviso
// cobre o tabuleiro sem uma linha nova.
//
// Sem isto, uma falha permanente de gravação fica invisível: foi assim que o
// tabuleiro passou um dia vivendo só em memória, com a tela impecável (ALE-124).
func (g *realtimeGateway) persistBoardAndWarn(sessionID int64) {
	if dirty, changed := g.s.boards.persist(context.Background(), sessionID); changed {
		g.warnPersistence(sessionID, dirty)
	}
}

// warnPersistence conta à sala que a gravação começou (ou deixou) de falhar. É
// o MESMO evento do rastreador, então a tela que já mostra o aviso cobre o
// tabuleiro sem uma linha nova.
func (g *realtimeGateway) warnPersistence(sessionID int64, dirty bool) {
	g.io.To(socket.Room(sessionRoomName(sessionID))).Emit("persistence-warning", map[string]any{
		"sessionId": sessionID, "dirty": dirty,
	})
}

// parseBoardToken lê a peça do corpo da mensagem. Posição ausente vira o
// primeiro quadrado livre da fileira de entrada — antes era (0,0) fixo, e com o
// "+ Peça" da ALE-178 duas peças criadas seguidas nasciam UMA EM CIMA DA OUTRA
// (ALE-166).
func parseBoardToken(body map[string]any) (BoardToken, bool) {
	token := BoardToken{
		Label: stringField(body, "label"),
		Kind:  stringField(body, "kind"),
	}
	x, temX := intField(body, "x")
	y, temY := intField(body, "y")
	if temX {
		token.X = int(x)
	}
	if temY {
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
	return token, temX && temY
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

// Movimento (ALE-124, fatia 3). Estes três NÃO passam pelo `requireGm`: a regra
// é mais fina e mora no `assertMovable` — o mestre move qualquer peça, o
// jogador move a própria na vez dela, e fora de combate cada um anda com a sua.
// É a mesma forma dos vitais, onde a porta larga esconderia a regra que importa.

// onBoardMovePropose mede o caminho e guarda o provisório para a mesa ver.
func (g *realtimeGateway) onBoardMovePropose(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	tokenID := stringField(ctx.body, "tokenId")
	path := parseSquarePath(ctx.body["path"])
	if tokenID == "" || len(path) < 2 {
		g.wsError(sock, "tokenId e um caminho com origem e destino são obrigatórios")
		return
	}
	by, speed := g.moverFor(ctx, tokenID)
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.proposeMove(context.Background(), ctx.sessionID,
			g.s.sessions.getState(ctx.sessionID), tokenID, path, by, speed)
	})
}

// onBoardMoveCommit pousa a peça. A versão vem do cliente e é o que recusa um
// commit escrito sobre uma cena que já mudou.
func (g *realtimeGateway) onBoardMoveCommit(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	version, _ := intField(ctx.body, "version")
	by, _ := g.moverFor(ctx, pendingTokenOf(g.s.boards.get(context.Background(), ctx.sessionID)))
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.commitMove(context.Background(), ctx.sessionID,
			g.s.sessions.getState(ctx.sessionID), version, by)
	})
}

// onBoardMoveCancel joga fora o provisório sem mexer na peça.
func (g *realtimeGateway) onBoardMoveCancel(sock *socket.Socket, args []any) {
	ctx, ok := g.access(sock, args)
	if !ok {
		return
	}
	by, _ := g.moverFor(ctx, pendingTokenOf(g.s.boards.get(context.Background(), ctx.sessionID)))
	g.mutateBoard(sock, ctx, func() (*BoardState, error) {
		return g.s.boards.cancelMove(context.Background(), ctx.sessionID, by)
	})
}

// moverFor resolve, contra o BANCO, as duas coisas que o cliente não pode
// afirmar sobre si: se a peça é de um personagem dele, e quanto ela anda.
//
// O deslocamento sai do MOTOR (`sheet.Displacement.Total`), não da coluna crua:
// a armadura pesada tira metros, e o número da mesa tem de ser o mesmo que o
// jogador lê na própria ficha. Falha de banco devolve orçamento zero, que a
// peça traduz para o padrão do livro — a mesa não para porque o disco piscou.
func (g *realtimeGateway) moverFor(ctx msgCtx, tokenID string) (mover, int) {
	by := mover{userID: ctx.userID, role: ctx.role}
	if by.role == "gm" || tokenID == "" {
		return by, 0
	}
	token := findToken(g.s.boards.get(context.Background(), ctx.sessionID), tokenID)
	if token == nil || token.CharacterID == nil {
		return by, 0
	}
	owner, err := g.s.queries.GetCharacterOwner(context.Background(), *token.CharacterID)
	if err != nil {
		log.Printf("board: dono do personagem %d não resolvido (%v)", *token.CharacterID, err)
		return by, 0
	}
	by.ownsCharacter = owner == ctx.userID
	return by, g.speedSquaresFor(*token.CharacterID)
}

// speedSquaresFor calcula o orçamento em quadrados a partir da ficha computada.
func (g *realtimeGateway) speedSquaresFor(characterID int64) int {
	row, err := g.s.queries.GetCharacter(context.Background(), characterID)
	if err != nil {
		return 0
	}
	sheet, err := g.s.computeSheet(context.Background(), row)
	if err != nil {
		log.Printf("board: ficha do personagem %d não computada (%v)", characterID, err)
		return 0
	}
	return engine.SquaresForDisplacement(float64(sheet.Displacement.Total))
}

// findToken num tabuleiro possivelmente ausente — a leitura do gateway acontece
// fora da trava, e "sem tabuleiro" é resposta legítima.
func pendingTokenOf(b *BoardState) string {
	if b == nil || b.Pending == nil {
		return ""
	}
	return b.Pending.TokenID
}

// parseSquarePath lê o caminho do corpo. Item que não é um par de números é
// DESCARTADO em silêncio, e o caminho encurtado será recusado pela medição —
// derrubar a mensagem inteira por causa de um item deixaria a peça presa.
func parseSquarePath(raw any) []engine.Square {
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	path := make([]engine.Square, 0, len(list))
	for _, item := range list {
		square, ok := item.(map[string]any)
		if !ok {
			continue
		}
		x, okX := intField(square, "x")
		y, okY := intField(square, "y")
		if !okX || !okY {
			continue
		}
		path = append(path, engine.Square{X: int(x), Y: int(y)})
	}
	return path
}
