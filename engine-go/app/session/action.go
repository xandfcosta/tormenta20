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
func (st *Store) SpendAction(sessionID int64, cost engine.ActionCost) (*live.SessionRuntimeState, error) {
	return st.apply(sessionID, events.TurnAdvanced{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error {
			left, charges, err := spendsFromTurn(s, cost)
			if err != nil || !charges {
				return err
			}
			s.Scene.StandardLeft, s.Scene.MovementLeft = left.Standard, left.Movement
			return nil
		})
}

// ActionFits diz se o custo CABERIA no turno, sem cobrar nada.
//
// Existe para quem precisa RECUSAR o gesto antes de aplicá-lo. Cobrar depois de
// aplicar parece equivalente e não é: a peça pousa na casa nova e a recusa vira
// só uma frase vermelha embaixo do mapa — a mesa lê o erro e vê o movimento
// feito. Medido na tela, e prendido por `TestMovingOnYourTurnSpendsTheMovementAction`.
func (st *Store) ActionFits(sessionID int64, cost engine.ActionCost) error {
	_, _, err := spendsFromTurn(st.GetState(sessionID), cost)
	return err
}

// spendsFromTurn é a decisão que as duas compartilham: o segundo valor diz se a
// cena COBRA — fora de uma cena de ação, nada cabe porque nada custa (p252).
func spendsFromTurn(s *live.SessionRuntimeState, cost engine.ActionCost) (engine.TurnBudget, bool, error) {
	if s == nil || s.Scene == nil || !s.Scene.CountsRounds() {
		return engine.TurnBudget{}, false, nil
	}
	left, err := engine.TurnBudget{
		Standard: s.Scene.StandardLeft, Movement: s.Scene.MovementLeft,
	}.Spend(cost)
	if err != nil {
		return left, true, fmt.Errorf("%s: %w", whoIsOnTurn(s), err)
	}
	return left, true, nil
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
func (st *Store) CharacterActionFits(characterID int64, cost engine.ActionCost) error {
	sessionID, moment, inScene := st.momentOf(characterID)
	if !inScene {
		return nil
	}
	if err := engine.UsableNow(cost, moment); err != nil {
		return err
	}
	return st.ActionFits(sessionID, cost)
}

// SpendCharacterAction cobra do turno o que o gesto FEITO custou.
func (st *Store) SpendCharacterAction(characterID int64, cost engine.ActionCost) error {
	sessionID, _, inScene := st.momentOf(characterID)
	if !inScene {
		return nil
	}
	_, err := st.SpendAction(sessionID, cost)
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
		return sessionID, st.characterMoment(characterID, isOnTurn(s, characterID)), true
	}
	return 0, engine.ActionMoment{}, false
}

// isOnTurn diz se a vez em curso é deste personagem.
func isOnTurn(s *live.SessionRuntimeState, characterID int64) bool {
	if s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return false
	}
	onTurn := s.Initiative[s.TurnIndex].CharacterID
	return onTurn != nil && *onTurn == characterID
}

// characterMoment monta o instante do personagem pelo PV e pelas condições da
// ficha — quem decide o que cada uma tira é o `engine.MomentFor`.
//
// Sem porta para a ficha, ou com a leitura falhando, a resposta é DE PÉ. O erro
// pende para o lado de deixar jogar: recusar o gesto de alguém porque o banco
// tossiu troca um número errado por uma mesa parada, e é o mesmo caminho que o
// `payUpkeep` escolheu pela mesma razão.
func (st *Store) characterMoment(characterID int64, onTurn bool) engine.ActionMoment {
	standing := engine.ActionMoment{OnTurn: onTurn, CanAct: true, CanReact: true}
	if st.sheet == nil || st.turnEffects == nil {
		return standing
	}
	pools, err := st.sheet.PoolsOf(context.Background(), []int64{characterID})
	if err != nil {
		return standing
	}
	pool, found := pools[characterID]
	if !found {
		return standing
	}
	conditions, err := st.turnEffects.ConditionsOf(context.Background(), characterID)
	if err != nil {
		conditions = nil
	}
	return engine.MomentFor(onTurn, pool.HpCurrent, conditions)
}
