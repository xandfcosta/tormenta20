package boards

import (
	"context"

	"t20engine/domain/board"
)

// RemeasureCharacter regrava o deslocamento das peças deste personagem em todo tabuleiro aberto da
// sessão, e SÓ onde o número mudou.
//
// A peça media a ficha uma vez, ao entrar no mapa, e a ficha muda no meio da cena: o interruptor
// "em terreno natural" acende a isenção de terreno difícil (p63), a condição Lento corta o
// deslocamento. Quem chama é o aviso de que a ficha mudou, e por ele passa também o PV de cada
// golpe — gravar sem comparar mandaria a mesa inteira redesenhar o tabuleiro a cada dano.
//
// Uso: `bs.RemeasureCharacter(ctx, 3, 7, board.TokenMovement{Squares: 6})` acerta as peças do
// personagem 7 nos tabuleiros da sessão 3.
func (bs *Store) RemeasureCharacter(ctx context.Context, sessionID, characterID int64, movement board.TokenMovement) error {
	if movement.Squares <= 0 {
		return nil
	}
	open, err := bs.OpenBoards(ctx, sessionID)
	if err != nil {
		return err
	}
	for _, b := range open {
		stale := map[string]board.TokenMovement{}
		for _, token := range b.Tokens {
			if token.CharacterID == nil || *token.CharacterID != characterID {
				continue
			}
			if token.SpeedSquares != movement.Squares || token.IgnoresDifficult != movement.IgnoresDifficult {
				stale[token.ID] = movement
			}
		}
		if len(stale) == 0 {
			continue
		}
		if _, err := bs.SetSpeeds(ctx, sessionID, b.ID, stale); err != nil {
			return err
		}
	}
	return nil
}
