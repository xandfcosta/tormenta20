package combat

import (
	"context"
	"fmt"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// BoardReader é o tabuleiro ABERTO de uma sessão, e só isso.
//
// Porta de uma função porque é uma pergunta só: a id vazia quer dizer "a aba
// padrão", que é a convenção do `boards.Store`.
type BoardReader interface {
	Get(ctx context.Context, sessionID int64, boardID string) (*board.BoardState, error)
}

// BoardSituations é o adaptador que faz o tabuleiro responder em REGRA.
//
// Ele existe para o `combat` não importar o `app/boards`: a tradução de
// "posição das peças" para "linhas da Tabela 5-3" é do `domain/board`, e este
// tipo é só quem busca o estado e repassa.
type BoardSituations struct{ boards BoardReader }

func NewBoardSituations(b BoardReader) BoardSituations { return BoardSituations{boards: b} }

// Between são as situações especiais que o tabuleiro produz entre dois
// combatentes.
//
// Sem tabuleiro aberto a resposta é VAZIA e não é erro — a maior parte de uma
// sessão não tem mapa, e o ataque acontece do mesmo jeito.
//
// @example combat.NewBoardSituations(store).Between(ctx, 7, "atk", board.AttackTargetOnTheBoard{EntryID: "alvo"})
func (b BoardSituations) Between(
	ctx context.Context, sessionID int64, attackerEntry string, target board.AttackTargetOnTheBoard,
) ([]engine.SpecialSituation, error) {
	state, err := b.boards.Get(ctx, sessionID, "")
	if err != nil {
		return nil, fmt.Errorf("ler o tabuleiro da sessão %d para o ataque: %w", sessionID, err)
	}
	return board.SituationsBetween(state, attackerEntry, target), nil
}
