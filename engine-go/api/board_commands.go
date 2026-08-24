package api

import (
	"context"
	"log"
	"net/http"
	"t20engine/plataforma"

	"github.com/go-chi/chi/v5"

	"t20engine/engine"
)

// CONTEXTO: TABULEIRO — peças, marcadores, terreno, lugares e movimento.
//
// Traduzido do socket para HTTP na ALE-253, e agrupado por assunto como
// primeiro passo do trabalho de bounded context (ALE-254). Este é o contexto
// mais autocontido do pacote: store próprio (`boardStore`), tabela própria
// (`session_boards`) e regras próprias já em `engine/board_*` — por isso é o
// candidato a virar pacote de verdade primeiro.
//
// O CORPO É LIDO COMO MAPA e passado aos parsers que já existem
// (`parseBoardToken`, `parseMarkerPatch`, `parseSquarePath`, `chosenEntries`).
// Não é preguiça: esses parsers carregam decisões — o marcador que nasce
// ESCONDIDO quando o campo não vem, o terreno que assume difícil — e reescrevê-
// las em structs seria refazer regra provada durante uma troca de transporte,
// que é exatamente o jeito de perder uma delas em silêncio. Trocar por structs
// tipados é limpeza válida, depois, com os testes de tabuleiro no ar.

// boardBody lê o corpo como mapa para os parsers do socket continuarem valendo.
// Corpo ausente é mapa vazio: vários comandos não têm corpo nenhum.
func boardBody(w http.ResponseWriter, r *http.Request) (map[string]any, bool) {
	body := map[string]any{}
	if r.ContentLength == 0 {
		return body, true
	}
	if !plataforma.DecodeJSON(w, r, &body) {
		return nil, false
	}
	return body, true
}

// mutateBoardAndPublish espelha o `mutateBoard` do gateway.
func (s *Server) mutateBoardAndPublish(
	w http.ResponseWriter, ctx liveCtx, mutate func() (*BoardState, error),
) {
	board, err := mutate()
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, boardForRole(ctx.role, board))
}

// publishBoardState transmite às duas salas por papel e persiste.
func (s *Server) publishBoardState(sessionID int64, board *BoardState) {
	// O tabuleiro já numera as próprias mutações, então a ordem sai de graça —
	// `Version` sobe a cada mutação aceita. Fechar o tabuleiro manda `nil` e cai
	// no caminho "sem ordem", que reinicia o destino de propósito.
	var ordem uint64
	if board != nil {
		ordem = uint64(board.Version)
	}
	s.sse.emitOrdered(sessionID, "gm", "board-state", ordem, board)
	s.sse.emitOrdered(sessionID, "player", "board-state", ordem, boardForRole("player", board))
	go s.persistBoardAndWarn(sessionID)
}

func (s *Server) persistBoardAndWarn(sessionID int64) {
	if dirty, changed := s.boards.persist(context.Background(), sessionID); changed {
		s.warnPersistenceOnBoard(sessionID, dirty)
	}
}

func (s *Server) warnPersistenceOnBoard(sessionID int64, dirty bool) {
	s.sse.emit(sessionID, "", "persistence-warning", map[string]any{
		"sessionId": sessionID, "dirty": dirty,
	})
}

// --- Abrir, fechar e ler ---------------------------------------------------

func (s *Server) handleBoardOpen(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	body, ok := boardBody(w, r)
	if !ok {
		return
	}
	board := s.boards.open(r.Context(), ctx.sessionID,
		plataforma.StringField(body, "place"), plataforma.StringField(body, "terrain"))
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, boardForRole(ctx.role, board))
}

// handleBoardClose ARQUIVA e fecha (ALE-124, fatia 5): a cena vira um Lugar da
// crônica com as peças onde estavam, para reabrir na semana seguinte sem
// remontar nada.
//
// A falha ao arquivar NÃO impede o fechar: o mestre mandou tirar a cena da
// mesa, e recusar isso porque o acervo falhou deixaria a mesa presa numa cena
// que já acabou. Quem conta a verdade é o aviso de persistência.
func (s *Server) handleBoardClose(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	if atual := s.boards.get(r.Context(), ctx.sessionID); atual != nil {
		if err := s.boards.archive(r.Context(), ctx.campaignID, atual); err != nil {
			log.Printf("session %d: falha ao arquivar o lugar (%v)", ctx.sessionID, err)
		}
	}
	// Um DELETE que falha deixa o tabuleiro fantasma no banco (ALE-155).
	if dirty, changed := s.boards.close(r.Context(), ctx.sessionID); changed {
		s.warnPersistenceOnBoard(ctx.sessionID, dirty)
	}
	// `nil` é a mensagem: "esta sessão não tem tabuleiro" é estado de verdade, e
	// não uma grade vazia.
	s.publishBoardState(ctx.sessionID, nil)
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"closed": true})
}

func (s *Server) handleBoardGet(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, boardForRole(ctx.role, s.boards.get(r.Context(), ctx.sessionID)))
}

// handleBoardAsPlayer devolve ao MESTRE a versão que a mesa vê — é o "espiar
// pelo olho do jogador", e por isso é do mestre e devolve o recorte de jogador.
func (s *Server) handleBoardAsPlayer(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, boardForRole("player", s.boards.get(r.Context(), ctx.sessionID)))
}

// --- Peças -----------------------------------------------------------------

func (s *Server) handleBoardTokenAdd(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	token, temPosicao := parseBoardToken(body)
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.addToken(r.Context(), ctx.sessionID, token, temPosicao)
	})
}

func (s *Server) handleBoardTokenRemove(w http.ResponseWriter, r *http.Request) {
	ctx, tokenID, ok := s.boardTokenCommand(w, r)
	if !ok {
		return
	}
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.removeToken(r.Context(), ctx.sessionID, tokenID)
	})
}

func (s *Server) handleBoardTokenUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	tokenID := pathToken(r)
	if tokenID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "tokenId is required")
		return
	}
	patch := parseTokenPatch(body["patch"])
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.updateToken(r.Context(), ctx.sessionID, tokenID, patch)
	})
}

func (s *Server) handleBoardTokenDuplicate(w http.ResponseWriter, r *http.Request) {
	ctx, tokenID, ok := s.boardTokenCommand(w, r)
	if !ok {
		return
	}
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.duplicateToken(r.Context(), ctx.sessionID, tokenID)
	})
}

// --- Marcadores ------------------------------------------------------------

func (s *Server) handleBoardMarkerAdd(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	x, temX := plataforma.IntField(body, "x")
	y, temY := plataforma.IntField(body, "y")
	if !temX || !temY {
		plataforma.WriteError(w, http.StatusBadRequest, "x and y are required")
		return
	}
	// Nasce ESCONDIDO quando ninguém disse o contrário: marcador é preparação
	// do mestre, e revelar é gesto.
	escondido, informado := body["hidden"].(bool)
	if !informado {
		escondido = true
	}
	marker := BoardMarker{
		X: int(x), Y: int(y),
		Text:   plataforma.StringField(body, "text"),
		Color:  plataforma.StringField(body, "color"),
		Hidden: escondido,
	}
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.addMarker(r.Context(), ctx.sessionID, marker)
	})
}

func (s *Server) handleBoardMarkerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	markerID := pathMarker(r)
	if markerID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "markerId is required")
		return
	}
	patch := parseMarkerPatch(body["patch"])
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.updateMarker(r.Context(), ctx.sessionID, markerID, patch)
	})
}

func (s *Server) handleBoardMarkerRemove(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	markerID := pathMarker(r)
	if markerID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "markerId is required")
		return
	}
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.removeMarker(r.Context(), ctx.sessionID, markerID)
	})
}

// --- Terreno e povoamento --------------------------------------------------

func (s *Server) handleBoardTerrainPaint(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	x, temX := plataforma.IntField(body, "x")
	y, temY := plataforma.IntField(body, "y")
	if !temX || !temY {
		plataforma.WriteError(w, http.StatusBadRequest, "x and y are required")
		return
	}
	difficult, informado := body["difficult"].(bool)
	if !informado {
		difficult = true
	}
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.paintTerrain(
			r.Context(), ctx.sessionID, engine.Square{X: int(x), Y: int(y)}, difficult,
		)
	})
}

func (s *Server) handleBoardPopulate(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	board, err := s.boards.populate(
		r.Context(), ctx.sessionID,
		s.sessions.getState(ctx.sessionID), chosenEntries(body, "entryIds"),
	)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if speeds := s.speedsForBoard(board); len(speeds) > 0 {
		if withSpeeds, err := s.boards.setSpeeds(r.Context(), ctx.sessionID, speeds); err == nil {
			board = withSpeeds
		}
	}
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, boardForRole(ctx.role, board))
}

// --- Lugares (o acervo de cenas da crônica) --------------------------------

// Só o mestre: saber que existe uma "Cripta do Necromante" guardada é meio
// caminho da surpresa. O jogador vê o lugar quando ele chega à mesa.

func (s *Server) handleBoardPlaces(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"places": s.boards.places(r.Context(), ctx.campaignID)})
}

func (s *Server) handleBoardReopen(w http.ResponseWriter, r *http.Request) {
	ctx, placeID, ok := s.boardPlaceCommand(w, r)
	if !ok {
		return
	}
	board, err := s.boards.showPlace(r.Context(), ctx.campaignID, ctx.sessionID, placeID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if dirty, changed := s.boards.persist(r.Context(), ctx.sessionID); changed {
		s.warnPersistenceOnBoard(ctx.sessionID, dirty)
	}
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, boardForRole(ctx.role, board))
}

func (s *Server) handleBoardPlaceRemove(w http.ResponseWriter, r *http.Request) {
	ctx, placeID, ok := s.boardPlaceCommand(w, r)
	if !ok {
		return
	}
	if err := s.boards.removePlace(r.Context(), ctx.campaignID, placeID); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, "não consegui apagar o lugar")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"places": s.boards.places(r.Context(), ctx.campaignID)})
}

// handleBoardPlaceScene abre o lugar para MONTAR — sem pôr na mesa.
func (s *Server) handleBoardPlaceScene(w http.ResponseWriter, r *http.Request) {
	ctx, placeID, ok := s.boardPlaceCommand(w, r)
	if !ok {
		return
	}
	cena, err := s.boards.placeScene(r.Context(), ctx.campaignID, placeID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, "não consegui abrir o lugar para montar")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, boardForRole(ctx.role, cena))
}

func (s *Server) handleBoardPlaceSave(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	placeID, temID := intParam(w, r, "placeId")
	if !temID {
		return
	}
	cena, err := parseScene(body["scene"])
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.boards.savePlaceScene(r.Context(), ctx.campaignID, placeID, cena); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"places": s.boards.places(r.Context(), ctx.campaignID)})
}

// --- Movimento -------------------------------------------------------------
//
// Os três NÃO exigem mestre: a regra é mais fina e mora no `moverFor` — o
// mestre move qualquer peça, o jogador move a própria na vez dela, e fora de
// combate cada um anda com a sua.

func (s *Server) handleBoardMovePropose(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return
	}
	body, ok := boardBody(w, r)
	if !ok {
		return
	}
	tokenID := pathToken(r)
	path := parseSquarePath(body["path"])
	if tokenID == "" || len(path) < 2 {
		plataforma.WriteError(w, http.StatusBadRequest, "tokenId e um caminho com origem e destino são obrigatórios")
		return
	}
	by, speed := s.moverFor(ctx, tokenID)
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.proposeMove(r.Context(), ctx.sessionID,
			s.sessions.getState(ctx.sessionID), tokenID, path, by, speed)
	})
}

func (s *Server) handleBoardMoveCommit(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return
	}
	body, ok := boardBody(w, r)
	if !ok {
		return
	}
	version, _ := plataforma.IntField(body, "version")
	by, _ := s.moverFor(ctx, pendingTokenOf(s.boards.get(r.Context(), ctx.sessionID)))
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.commitMove(r.Context(), ctx.sessionID,
			s.sessions.getState(ctx.sessionID), version, by)
	})
}

func (s *Server) handleBoardMoveCancel(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return
	}
	by, _ := s.moverFor(ctx, pendingTokenOf(s.boards.get(r.Context(), ctx.sessionID)))
	s.mutateBoardAndPublish(w, ctx, func() (*BoardState, error) {
		return s.boards.cancelMove(r.Context(), ctx.sessionID, by)
	})
}

// boardPlaceCommand: mesa, papel de mestre e o lugar vindo do caminho.
func (s *Server) boardPlaceCommand(w http.ResponseWriter, r *http.Request) (liveCtx, int64, bool) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return liveCtx{}, 0, false
	}
	placeID, ok := intParam(w, r, "placeId")
	if !ok {
		return liveCtx{}, 0, false
	}
	return ctx, placeID, true
}

// --- Preâmbulos comuns -----------------------------------------------------

// boardGmCommand é o começo repetido: mesa, papel de mestre e corpo.
func (s *Server) boardGmCommand(w http.ResponseWriter, r *http.Request) (liveCtx, map[string]any, bool) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return liveCtx{}, nil, false
	}
	body, ok := boardBody(w, r)
	if !ok {
		return liveCtx{}, nil, false
	}
	return ctx, body, true
}

// boardTokenCommand acrescenta a peça do caminho, para os comandos sem corpo.
func (s *Server) boardTokenCommand(w http.ResponseWriter, r *http.Request) (liveCtx, string, bool) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return liveCtx{}, "", false
	}
	tokenID := pathToken(r)
	if tokenID == "" {
		plataforma.WriteError(w, http.StatusBadRequest, "tokenId is required")
		return liveCtx{}, "", false
	}
	return ctx, tokenID, true
}

func pathToken(r *http.Request) string  { return chi.URLParam(r, "tokenId") }
func pathMarker(r *http.Request) string { return chi.URLParam(r, "markerId") }
