package session

import (
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// A ECONOMIA DE AÇÃO, ligada: o regime GUARDA o que sobrou do turno e o motor
// DECIDE se o custo cabe (ALE-365).
//
// Os dois lados existem separados porque o regime não alcança o motor — a
// fronteira do `domain/live` é explícita sobre isso —, e esta camada é a única
// que conhece os dois. É o mesmo desenho do ataque: a regra do livro de um
// lado, o estado da mesa do outro, e o caso de uso costurando.

// SpendAction cobra do turno em curso o custo de uma ação.
//
// FORA DA CENA DE AÇÃO ela não cobra nada e não recusa: numa conversa na corte
// não há turno, e um gesto que fosse recusado ali estaria cobrando uma regra
// que o livro não aplica (p252).
func (st *Store) SpendAction(sessionID int64, custo engine.ActionCost) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.TurnAdvanced{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error {
			sobrou, cobra, err := spendsFromTurn(s, custo)
			if err != nil || !cobra {
				return err
			}
			s.Scene.StandardLeft, s.Scene.MovementLeft = sobrou.Standard, sobrou.Movement
			return nil
		})
}

// ActionFits diz se o custo CABERIA no turno, sem cobrar nada.
//
// Existe para quem precisa RECUSAR o gesto antes de aplicá-lo. Cobrar depois de
// aplicar parece equivalente e não é: a peça pousa na casa nova e a recusa vira
// só uma frase vermelha embaixo do mapa — a mesa lê o erro e vê o movimento
// feito. Medido na tela, e prendido por `TestMovingOnYourTurnSpendsTheMovementAction`.
func (st *Store) ActionFits(sessionID int64, custo engine.ActionCost) error {
	_, _, err := spendsFromTurn(st.GetState(sessionID), custo)
	return err
}

// spendsFromTurn é a decisão que as duas compartilham: o segundo valor diz se a
// cena COBRA — fora de uma cena de ação, nada cabe porque nada custa (p252).
func spendsFromTurn(s *live.SessionRuntimeState, custo engine.ActionCost) (engine.TurnBudget, bool, error) {
	if s == nil || s.Scene == nil || !s.Scene.CountsRounds() {
		return engine.TurnBudget{}, false, nil
	}
	sobrou, err := engine.TurnBudget{
		Standard: s.Scene.StandardLeft, Movement: s.Scene.MovementLeft,
	}.Spend(custo)
	if err != nil {
		return sobrou, true, fmt.Errorf("%s: %w", whoIsOnTurn(s), err)
	}
	return sobrou, true, nil
}

// whoIsOnTurn nomeia quem ficou sem ação, porque "não sobrou ação neste
// turno" sem sujeito manda procurar em nove linhas da fila.
func whoIsOnTurn(s *live.SessionRuntimeState) string {
	if s.TurnIndex >= 0 && s.TurnIndex < len(s.Initiative) {
		return s.Initiative[s.TurnIndex].Label
	}
	return "quem está na vez"
}
