package api

import "t20engine/tabuleiro"

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
// mais autocontido do pacote: store próprio (`tabuleiro.BoardStore`), tabela própria
// (`session_boards`) e regras próprias já em `engine/board_*` — por isso é o
// candidato a virar pacote de verdade primeiro.
//
// O CORPO É LIDO COMO MAPA e passado aos parsers que já existem
// (`tabuleiro.ParseBoardToken`, `tabuleiro.ParseMarkerPatch`, `parseSquarePath`, `chosenEntries`).
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
	w http.ResponseWriter, ctx liveCtx, mutate func() (*tabuleiro.BoardState, error),
) {
	board, err := mutate()
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, board))
}

// publishBoardState transmite às duas salas por papel e persiste.
func (s *Server) publishBoardState(sessionID int64, board *tabuleiro.BoardState) {
	// O tabuleiro já numera as próprias mutações, então a ordem sai de graça —
	// `Version` sobe a cada mutação aceita. Fechar o tabuleiro manda `nil` e cai
	// no caminho "sem ordem", que reinicia o destino de propósito.
	var ordem uint64
	if board != nil {
		ordem = uint64(board.Version)
	}
	s.sse.EmitOrdered(sessionID, "gm", "board-state", ordem, board)
	s.sse.EmitOrdered(sessionID, "player", "board-state", ordem, tabuleiro.BoardForRole("player", board))
	go s.persistBoardAndWarn(sessionID)
}

func (s *Server) persistBoardAndWarn(sessionID int64) {
	if Dirty, changed := s.boards.Persist(context.Background(), sessionID); changed {
		s.warnPersistenceOnBoard(sessionID, Dirty)
	}
}

func (s *Server) warnPersistenceOnBoard(sessionID int64, Dirty bool) {
	s.sse.Emit(sessionID, "", "persistence-warning", map[string]any{
		"sessionId": sessionID, "Dirty": Dirty,
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
	board := s.boards.Open(r.Context(), ctx.sessionID,
		plataforma.StringField(body, "place"), plataforma.StringField(body, "terrain"))
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, board))
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
	if atual := s.boards.Get(r.Context(), ctx.sessionID); atual != nil {
		if err := s.boards.Archive(r.Context(), ctx.campaignID, atual); err != nil {
			log.Printf("session %d: falha ao arquivar o lugar (%v)", ctx.sessionID, err)
		}
	}
	// Um DELETE que falha deixa o tabuleiro fantasma no banco (ALE-155).
	if Dirty, changed := s.boards.Close(r.Context(), ctx.sessionID); changed {
		s.warnPersistenceOnBoard(ctx.sessionID, Dirty)
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
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, s.boards.Get(r.Context(), ctx.sessionID)))
}

// handleBoardAsPlayer devolve ao MESTRE a versão que a mesa vê — é o "espiar
// pelo olho do jogador", e por isso é do mestre e devolve o recorte de jogador.
// handleBoardCurtain fecha ou abre a CORTINA (ALE-202): o tabuleiro continua
// inteiro para o mestre e a mesa vê uma cortina no lugar dele.
//
// A trava é do SERVIDOR e não da tela. Esconder o mapa no cliente entregaria o
// estado inteiro no fio e deixaria a emboscada a um DevTools de distância — a
// cortina é feita em `tabuleiro.BoardForRole`, o mesmo lugar que já apaga a peça
// escondida (ALE-124), e o cliente recebe um tabuleiro que NÃO TEM a cena.
//
// Não existe rota para o jogador abrir a própria cortina; ela sai daqui, e daqui
// só o mestre passa.
func (s *Server) handleBoardCurtain(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	fechada, informado := body["curtained"].(bool)
	if !informado {
		plataforma.WriteError(w, http.StatusBadRequest, "curtained (bool) is required")
		return
	}
	board, mudou, err := s.boards.SetCurtain(r.Context(), ctx.sessionID, fechada)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Sem mudança não se publica: ver `SetCurtain`. O mestre ainda recebe o
	// estado, porque a resposta dele é a confirmação do gesto.
	if mudou {
		s.publishBoardState(ctx.sessionID, board)
	}
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, board))
}

func (s *Server) handleBoardAsPlayer(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole("player", s.boards.Get(r.Context(), ctx.sessionID)))
}

// --- Peças -----------------------------------------------------------------

func (s *Server) handleBoardTokenAdd(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	token, temPosicao := tabuleiro.ParseBoardToken(body)
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.AddToken(r.Context(), ctx.sessionID, token, temPosicao)
	})
}

func (s *Server) handleBoardTokenRemove(w http.ResponseWriter, r *http.Request) {
	ctx, tokenID, ok := s.boardTokenCommand(w, r)
	if !ok {
		return
	}
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.RemoveToken(r.Context(), ctx.sessionID, tokenID)
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
	patch := tabuleiro.ParseTokenPatch(body["patch"])
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.UpdateToken(r.Context(), ctx.sessionID, tokenID, patch)
	})
}

func (s *Server) handleBoardTokenDuplicate(w http.ResponseWriter, r *http.Request) {
	ctx, tokenID, ok := s.boardTokenCommand(w, r)
	if !ok {
		return
	}
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.DuplicateToken(r.Context(), ctx.sessionID, tokenID)
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
	marker := tabuleiro.BoardMarker{
		X: int(x), Y: int(y),
		Text:   plataforma.StringField(body, "text"),
		Color:  plataforma.StringField(body, "color"),
		Hidden: escondido,
	}
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.AddMarker(r.Context(), ctx.sessionID, marker)
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
	patch := tabuleiro.ParseMarkerPatch(body["patch"])
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.UpdateMarker(r.Context(), ctx.sessionID, markerID, patch)
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
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.RemoveMarker(r.Context(), ctx.sessionID, markerID)
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
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.PaintTerrain(
			r.Context(), ctx.sessionID, engine.Square{X: int(x), Y: int(y)}, difficult,
		)
	})
}

func (s *Server) handleBoardPopulate(w http.ResponseWriter, r *http.Request) {
	ctx, body, ok := s.boardGmCommand(w, r)
	if !ok {
		return
	}
	board, err := s.boards.Populate(
		r.Context(), ctx.sessionID,
		s.sessions.GetState(ctx.sessionID), tabuleiro.ChosenEntries(body, "entryIds"),
	)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if speeds := s.speedsForBoard(board); len(speeds) > 0 {
		if withSpeeds, err := s.boards.SetSpeeds(r.Context(), ctx.sessionID, speeds); err == nil {
			board = withSpeeds
		}
	}
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, board))
}

// --- Lugares (o acervo de cenas da crônica) --------------------------------

// Só o mestre: saber que existe uma "Cripta do Necromante" guardada é meio
// caminho da surpresa. O jogador vê o lugar quando ele chega à mesa.

func (s *Server) handleBoardPlaces(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok || !requireGmRole(w, ctx) {
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"Places": s.boards.Places(r.Context(), ctx.campaignID)})
}

func (s *Server) handleBoardReopen(w http.ResponseWriter, r *http.Request) {
	ctx, placeID, ok := s.boardPlaceCommand(w, r)
	if !ok {
		return
	}
	board, err := s.boards.ShowPlace(r.Context(), ctx.campaignID, ctx.sessionID, placeID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if Dirty, changed := s.boards.Persist(r.Context(), ctx.sessionID); changed {
		s.warnPersistenceOnBoard(ctx.sessionID, Dirty)
	}
	s.publishBoardState(ctx.sessionID, board)
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, board))
}

func (s *Server) handleBoardPlaceRemove(w http.ResponseWriter, r *http.Request) {
	ctx, placeID, ok := s.boardPlaceCommand(w, r)
	if !ok {
		return
	}
	if err := s.boards.RemovePlace(r.Context(), ctx.campaignID, placeID); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, "não consegui apagar o lugar")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"Places": s.boards.Places(r.Context(), ctx.campaignID)})
}

// handleBoardPlaceScene abre o lugar para MONTAR — sem pôr na mesa.
func (s *Server) handleBoardPlaceScene(w http.ResponseWriter, r *http.Request) {
	ctx, placeID, ok := s.boardPlaceCommand(w, r)
	if !ok {
		return
	}
	cena, err := s.boards.PlaceScene(r.Context(), ctx.campaignID, placeID)
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, "não consegui abrir o lugar para montar")
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, tabuleiro.BoardForRole(ctx.Role, cena))
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
	cena, err := tabuleiro.ParseScene(body["scene"])
	if err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.boards.SavePlaceScene(r.Context(), ctx.campaignID, placeID, cena); err != nil {
		plataforma.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	plataforma.WriteJSON(w, http.StatusOK, map[string]any{"Places": s.boards.Places(r.Context(), ctx.campaignID)})
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
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.ProposeMove(r.Context(), ctx.sessionID,
			s.sessions.GetState(ctx.sessionID), tokenID, path, by, speed)
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
	by, _ := s.moverFor(ctx, pendingTokenOf(s.boards.Get(r.Context(), ctx.sessionID)))
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.CommitMove(r.Context(), ctx.sessionID,
			s.sessions.GetState(ctx.sessionID), version, by)
	})
}

func (s *Server) handleBoardMoveCancel(w http.ResponseWriter, r *http.Request) {
	ctx, ok := s.liveAccess(w, r)
	if !ok {
		return
	}
	by, _ := s.moverFor(ctx, pendingTokenOf(s.boards.Get(r.Context(), ctx.sessionID)))
	s.mutateBoardAndPublish(w, ctx, func() (*tabuleiro.BoardState, error) {
		return s.boards.CancelMove(r.Context(), ctx.sessionID, by)
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
