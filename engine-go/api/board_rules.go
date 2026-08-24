package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"t20engine/engine"
)

// As regras do tabuleiro: quem se move, quanto anda, o que vira peça.
//
// Este arquivo é o que SOBROU de `realtime_board.go` quando o socket.io foi
// apagado (ALE-253). O corte foi pelo receptor: o que era `(g *realtimeGateway)`
// era transporte e morreu junto; o que está aqui é aplicação, e não mudou uma
// linha ao mudar de vizinho. As rotas HTTP em `session_commands.go` e
// `board_commands.go` chamam exatamente as mesmas funções que os eventos
// chamavam.

// parseScene lê a cena montada do corpo da mensagem. Passa pelo JSON de novo
// porque o corpo chega como `map[string]any` genérico, e reconstruir o
// `BoardState` campo a campo aqui seria uma segunda definição do formato de fio.
func parseScene(raw any) (*BoardState, error) {
	blob, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("cena ilegível: %w", err)
	}
	var cena BoardState
	if err := json.Unmarshal(blob, &cena); err != nil {
		return nil, fmt.Errorf("cena ilegível: %w", err)
	}
	if cena.Tokens == nil {
		cena.Tokens = []BoardToken{}
	}
	return &cena, nil
}

// chosenEntries lê do corpo as linhas que o mestre escolheu trazer (ALE-204).
//
// Ausente devolve nil — TODAS, o significado que o evento sempre teve e que uma
// aba aberta antes desta mudança ainda manda. Lista vazia devolve conjunto
// vazio, que não traz ninguém: os dois casos são diferentes de propósito.
func chosenEntries(body map[string]any, key string) entrySelection {
	raw, ok := body[key].([]any)
	if !ok {
		return nil
	}
	chosen := entrySelection{}
	for _, item := range raw {
		if id, ok := item.(string); ok {
			chosen[id] = true
		}
	}
	return chosen
}

// speedsForBoard mede o deslocamento das peças de personagem que ainda não têm
// um. Só as que faltam: recomputar a ficha de todo mundo a cada "trazer o grupo"
// seria pagar caro por um número que não muda sozinho.
func (s *Server) speedsForBoard(board *BoardState) map[string]int {
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
func (s *Server) moverFor(ctx liveCtx, tokenID string) (mover, int) {
	by := mover{userID: ctx.userID, role: ctx.role}
	if by.role == "gm" || tokenID == "" {
		return by, 0
	}
	token := findToken(s.boards.get(context.Background(), ctx.sessionID), tokenID)
	if token == nil || token.CharacterID == nil {
		return by, 0
	}
	owner, err := s.queries.GetCharacterOwner(context.Background(), *token.CharacterID)
	if err != nil {
		log.Printf("board: dono do personagem %d não resolvido (%v)", *token.CharacterID, err)
		return by, 0
	}
	by.ownsCharacter = owner == ctx.userID
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
