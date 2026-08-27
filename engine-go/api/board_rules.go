package api

import "t20engine/tabuleiro"

import (
	"context"
	"log"
	"t20engine/plataforma"

	"t20engine/engine"
)

// speedsForBoard mede o deslocamento das peças de personagem que ainda não têm
// um. Só as que faltam: recomputar a ficha de todo mundo a cada "trazer o grupo"
// seria pagar caro por um número que não muda sozinho.
func (s *Server) speedsForBoard(board *tabuleiro.BoardState) map[string]int {
	speeds := map[string]int{}
	if board == nil {
		return speeds
	}
	for _, token := range board.Tokens {
		if token.CharacterID == nil || token.SpeedSquares > 0 {
			continue
		}
		if squares := s.speedSquaresFor(*token.CharacterID); squares > 0 {
			speeds[token.ID] = squares
		}
	}
	return speeds
}

// moverFor resolve, contra o BANCO, as duas coisas que o cliente não pode
// afirmar sobre si: se a peça é de um personagem dele, e quanto ela anda.
//
// O deslocamento sai do MOTOR (`sheet.Displacement.Total`), não da coluna crua:
// a armadura pesada tira metros, e o número da mesa tem de ser o mesmo que o
// jogador lê na própria ficha. Falha de banco devolve orçamento zero, que a
// peça traduz para o padrão do livro — a mesa não para porque o disco piscou.
func (s *Server) moverFor(ctx liveCtx, tokenID string) (tabuleiro.Mover, int) {
	by := tabuleiro.Mover{UserID: ctx.UserID, Role: ctx.Role}
	if by.Role == "gm" || tokenID == "" {
		return by, 0
	}
	token := tabuleiro.FindToken(s.boards.Get(context.Background(), ctx.sessionID, aAbaPadrao), tokenID)
	if token == nil || token.CharacterID == nil {
		return by, 0
	}
	owner, err := s.queries.GetCharacterOwner(context.Background(), *token.CharacterID)
	if err != nil {
		log.Printf("board: dono do personagem %d não resolvido (%v)", *token.CharacterID, err)
		return by, 0
	}
	by.OwnsCharacter = owner == ctx.UserID
	return by, s.speedSquaresFor(*token.CharacterID)
}

// speedSquaresFor calcula o orçamento em quadrados a partir da ficha computada.
func (s *Server) speedSquaresFor(characterID int64) int {
	row, err := s.queries.GetCharacter(context.Background(), characterID)
	if err != nil {
		return 0
	}
	sheet, err := s.computeSheet(context.Background(), row)
	if err != nil {
		log.Printf("board: ficha do personagem %d não computada (%v)", characterID, err)
		return 0
	}
	return engine.SquaresForDisplacement(float64(sheet.Displacement.Total))
}

// tabuleiro.FindToken num tabuleiro possivelmente ausente — a leitura do gateway acontece
// fora da trava, e "sem tabuleiro" é resposta legítima.
func pendingTokenOf(b *tabuleiro.BoardState) string {
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
		x, okX := plataforma.IntField(square, "x")
		y, okY := plataforma.IntField(square, "y")
		if !okX || !okY {
			continue
		}
		path = append(path, engine.Square{X: int(x), Y: int(y)})
	}
	return path
}
