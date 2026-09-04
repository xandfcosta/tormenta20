package api

import (
	"context"
	"log"
	"t20engine/engine"
	"t20engine/plataforma"
	"t20engine/tabuleiro"
)

// speedsForBoard mede o deslocamento das peças de personagem que ainda não têm
// um. Só as que faltam: recomputar a ficha de todo mundo a cada "trazer o grupo"
// seria pagar caro por um número que não muda sozinho.
func (tr tableRules) speedsForBoard(board *tabuleiro.BoardState) map[string]int {
	speeds := map[string]int{}
	if board == nil {
		return speeds
	}
	for _, token := range board.Tokens {
		if token.CharacterID == nil || token.SpeedSquares > 0 {
			continue
		}
		if squares := tr.speedSquaresFor(*token.CharacterID); squares > 0 {
			speeds[token.ID] = squares
		}
	}
	return speeds
}

// speedSquaresFor calcula o orçamento em quadrados a partir da ficha computada.
func (tr tableRules) speedSquaresFor(characterID int64) int {
	row, err := tr.queries.GetCharacter(context.Background(), characterID)
	if err != nil {
		return 0
	}
	sheet, err := tr.sheet.ComputeSheet(context.Background(), row)
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
