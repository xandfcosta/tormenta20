package session

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// FireBlast guarda o que um encanto dispara em vez do golpe — a bola de fogo da Flamejante (p335) —, e cobra
// na MESMA transação o que o gesto custa, como a proposta do golpe (decisão do dono): a vez da rodada do
// encanto, a ação padrão do ataque que ele troca e o PM. Uma cobrança pela metade deixaria a bola de fogo de
// graça, ou o PM gasto sem bola de fogo. Ninguém perde PV aqui: o dano e a resistência são do mestre.
//
// A vez da rodada vem antes da ação, pela razão do gasto do golpe: a segunda bola de fogo da vez ouve a regra
// do encanto, e não "falta ação".
func (st *Store) FireBlast(ctx context.Context, sessionID int64, blast live.BlastRoll) (*live.SessionRuntimeState, error) {
	moved := &blast.CharacterID
	clone, err := st.unitGesture(ctx, sessionID,
		func(inUnit context.Context, u Unit, s *live.SessionRuntimeState) error {
			if err := live.RecordBlast(s, blast); err != nil {
				return err
			}
			if s.CountsRounds() {
				if !live.UsesOncePerRound(s, blast.AttackerEntryID, blast.BoostID) {
					return fmt.Errorf("a %s de %s é uma vez por rodada (p335), e já saiu nesta: espere a próxima vez",
						blast.Name, blast.Weapon)
				}
				if s.TurnIndex >= 0 && s.TurnIndex < len(s.Initiative) && s.Initiative[s.TurnIndex].ID == blast.AttackerEntryID {
					if err := chargeTurn(s, engine.ActionStandard); err != nil {
						return fmt.Errorf("a %s é em vez do ataque, e a ação padrão dele já não cabe nesta vez: %w", blast.Name, err)
					}
				}
			}
			if err := spendsOnTheBlast(inUnit, u, s, blast); err != nil {
				return err
			}
			return stampTheSheet(inUnit, u, moved)
		})
	if err != nil {
		return nil, err
	}
	st.bus.Publish(events.AttackRolled{SessionID: sessionID})
	st.wakeTheSheet(moved)
	return clone, nil
}

// spendsOnTheBlast tira o PM do disparo da ficha de quem dispara, e a linha da fila acompanha. Sem recibo: o
// disparo não tem cancelamento — ele aconteceu, e o mestre aplica ou não o que ele causa.
func spendsOnTheBlast(ctx context.Context, u Unit, s *live.SessionRuntimeState, blast live.BlastRoll) error {
	pools, err := u.Sheet.PoolsOf(ctx, []int64{blast.CharacterID})
	if err != nil {
		return fmt.Errorf("ler o PM de %s: %w", blast.Who, err)
	}
	if available := pools[blast.CharacterID].ManaAvailable(); int64(blast.PM) > available {
		return fmt.Errorf("a %s pede %d PM, e sobram %d", blast.Name, blast.PM, available)
	}
	hp, mp, err := u.Sheet.SpendMana(ctx, blast.CharacterID, blast.PM)
	if err != nil {
		return fmt.Errorf("cobrar o PM da %s: %w", blast.Name, err)
	}
	return live.PatchEntryVitals(s, blast.AttackerEntryID, hp, mp, hpFloorOf(s, blast.AttackerEntryID))
}
