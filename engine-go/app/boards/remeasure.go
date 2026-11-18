package boards

import (
	"context"
	"slices"

	"t20engine/app/session"
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
	// A GRAVAÇÃO É DO STORE, e nunca da unidade que o contexto carrega: quem chama é o aviso de que
	// a ficha mudou, que pode sair DEPOIS do commit de uma unidade cujo contexto ainda a leva. Com
	// ela ali, o escritor seria o da transação fechada e a troca na memória ficaria para um "depois
	// do commit" que já passou — a peça não mudava, e o teste do voo que cai o mediu.
	ctx = context.WithValue(session.AfterTheUnit(ctx), snapshotsKey{}, BoardSnapshots(nil))
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
			if token.SpeedSquares != movement.Squares || token.IgnoresDifficult != movement.IgnoresDifficult ||
				!slices.Equal(token.OtherModes, movement.Modes) {
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

// SessionsWithPieceOf devolve as sessões com tabuleiro aberto em memória que têm peça deste
// personagem. Só a trava dos tabuleiros: quem pergunta pode estar dentro da trava da sessão.
func (bs *Store) SessionsWithPieceOf(characterID int64) []int64 {
	bs.Mu.Lock()
	defer bs.Mu.Unlock()
	var found []int64
	for sessionID, open := range bs.boards {
		if hasPieceOf(open, characterID) {
			found = append(found, sessionID)
		}
	}
	return found
}

func hasPieceOf(open []*board.BoardState, characterID int64) bool {
	for _, b := range open {
		for _, token := range b.Tokens {
			if token.CharacterID != nil && *token.CharacterID == characterID {
				return true
			}
		}
	}
	return false
}
