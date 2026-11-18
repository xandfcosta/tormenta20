package boards

import (
	"context"
	"fmt"
	"slices"

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
	// a ficha mudou, que pode sair DEPOIS do commit de uma unidade cujo contexto ainda leva o
	// escritor dela — e gravar por uma transação fechada falha. Zerar a chave devolve o
	// `snapshotsFor` ao escritor do store.
	ctx = context.WithValue(ctx, snapshotsKey{}, BoardSnapshots(nil))
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

// SessionsWithPieceOf devolve as sessões não encerradas com tabuleiro aberto que tem peça deste
// personagem, lidas do banco numa consulta. Não pede trava nenhuma: quem pergunta pode estar dentro
// da trava da sessão (o aviso de depois do commit).
func (bs *Store) SessionsWithPieceOf(ctx context.Context, characterID int64) ([]int64, error) {
	rows, err := bs.q.ListLiveOpenBoards(ctx)
	if err != nil {
		return nil, fmt.Errorf("ler os tabuleiros abertos: %w", err)
	}
	var found []int64
	for _, row := range rows {
		if slices.Contains(found, row.Sessionid) {
			continue
		}
		parsed, err := parseBoardBlob(row.State)
		if err != nil {
			continue
		}
		if hasPieceOf(parsed, characterID) {
			found = append(found, row.Sessionid)
		}
	}
	return found, nil
}

func hasPieceOf(b *board.BoardState, characterID int64) bool {
	for _, token := range b.Tokens {
		if token.CharacterID != nil && *token.CharacterID == characterID {
			return true
		}
	}
	return false
}
