package combat

import (
	"context"
	"fmt"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// BoardReader são os tabuleiros ABERTOS de uma sessão, na ordem das abas, e só isso.
//
// Porta de uma função porque é uma pergunta só. Ela devolve todos, e não o da aba padrão: o golpe lê o tabuleiro
// em que as duas peças estão, ou o do alvo (`board.BoardForTheStrike`).
type BoardReader interface {
	OpenBoards(ctx context.Context, sessionID int64) ([]*board.BoardState, error)
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
// Sem tabuleiro aberto, ou sem um que tenha o alvo, a resposta é VAZIA e não é erro — a maior parte de uma
// sessão não tem mapa, e o ataque acontece do mesmo jeito.
//
// @example combat.NewBoardSituations(store).Between(ctx, 7, "atk", board.AttackTargetOnTheBoard{EntryID: "alvo"})
func (b BoardSituations) Between(
	ctx context.Context, sessionID int64, attackerEntry string, target board.AttackTargetOnTheBoard,
) ([]engine.SpecialSituation, error) {
	open, err := b.boards.OpenBoards(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ler os tabuleiros da sessão %d para o ataque: %w", sessionID, err)
	}
	return board.SituationsBetween(board.BoardForTheStrike(open, attackerEntry, target), attackerEntry, target), nil
}
