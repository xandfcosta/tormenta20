package session

import (
	"context"
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

// O TURNO DE UM PERSONAGEM, para quem chega pela FICHA.
//
// Conjurar sai da ficha, que não conhece sessão nenhuma: ela pergunta por um
// personagem, e achar a mesa em que ele está é trabalho desta camada — a única
// que conhece as duas.
//
// São DUAS funções pelo mesmo motivo que o `ActionFits` e o `SpendAction`:
// quem precisa RECUSAR o gesto tem de perguntar ANTES de aplicá-lo. Uma
// conjuração recusada por falta de PM depois de a padrão já ter sido cobrada
// tiraria a vez de alguém por uma magia que nunca saiu.
//
// FORA DE UMA CENA DE AÇÃO nenhuma das duas cobra ou recusa (p252), e "não está
// em mesa nenhuma" é o mesmo caso: a ficha aberta sozinha não tem turno.

// CharacterActionFits diz se o gesto caberia agora, sem cobrar nada.
//
// As DUAS perguntas são feitas em ordem, e a ordem é a das frases: primeiro se é
// a hora (`UsableNow`), depois se sobrou (`ActionFits`). Invertida, quem tenta
// agir fora da vez com o turno cheio ouviria "não sobrou ação".
func (st *Store) CharacterActionFits(characterID int64, custo engine.ActionCost) error {
	sessionID, quando, emCena := st.momentOf(characterID)
	if !emCena {
		return nil
	}
	if err := engine.UsableNow(custo, quando); err != nil {
		return err
	}
	return st.ActionFits(sessionID, custo)
}

// SpendCharacterAction cobra do turno o que o gesto FEITO custou.
func (st *Store) SpendCharacterAction(characterID int64, custo engine.ActionCost) error {
	sessionID, _, emCena := st.momentOf(characterID)
	if !emCena {
		return nil
	}
	_, err := st.SpendAction(sessionID, custo)
	return err
}

// momentOf acha a cena de AÇÃO em que este personagem está e monta o instante
// dele. O terceiro valor diz se há cena — sem ela os outros dois não querem
// dizer nada.
func (st *Store) momentOf(characterID int64) (int64, engine.ActionMoment, bool) {
	for _, sessionID := range st.LiveSessionsWithCharacter(characterID) {
		s := st.GetState(sessionID)
		if !s.CountsRounds() {
			continue
		}
		return sessionID, engine.ActionMoment{
			OnTurn: isOnTurn(s, characterID),
			CanAct: st.canAct(characterID),
		}, true
	}
	return 0, engine.ActionMoment{}, false
}

// isOnTurn diz se a vez em curso é deste personagem.
func isOnTurn(s *live.SessionRuntimeState, characterID int64) bool {
	if s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return false
	}
	naVez := s.Initiative[s.TurnIndex].CharacterID
	return naVez != nil && *naVez == characterID
}

// canAct é ter PV: a 0 "você cai inconsciente" (p236), e o poço do app tem piso
// em zero.
//
// Sem porta para a ficha, ou com a leitura falhando, a resposta é SIM. O erro
// pende para o lado de deixar jogar: recusar o gesto de alguém porque o banco
// tossiu troca um número errado por uma mesa parada, e é o mesmo caminho que o
// `payUpkeep` escolheu pela mesma razão.
func (st *Store) canAct(characterID int64) bool {
	if st.ficha == nil {
		return true
	}
	pocos, err := st.ficha.PoolsOf(context.Background(), []int64{characterID})
	if err != nil {
		return true
	}
	poco, tem := pocos[characterID]
	return !tem || poco.HpCurrent > 0
}
