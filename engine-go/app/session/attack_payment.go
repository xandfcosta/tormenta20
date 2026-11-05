package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// A PROPOSTA PAGA AO NASCER, e o cancelamento do mestre devolve (decisão do dono).
//
// Por que não no ✓: até lá o jogador podia gastar a mesma ação numa postura e o mesmo PM
// no Sortudo. O ✓ e o "errou" recusavam por falta de ação, sobrando só o cancelamento, e
// o impulso saía de graça com o poço em zero. Paga na proposta, a postura que não couber
// é recusada na hora dela, e a resposta do mestre nunca falha por um gesto posterior.

// ProposeAttack guarda o ataque rolado, e a ação e o impulso saem na mesma transação.
// Ninguém perde PV aqui.
func (st *Store) ProposeAttack(ctx context.Context, sessionID int64, attack live.PendingAttack) (*live.SessionRuntimeState, error) {
	var moved *int64
	clone, err := st.unitGesture(ctx, sessionID,
		func(inUnit context.Context, u Unit, s *live.SessionRuntimeState) error {
			if err := live.ProposeAttack(s, attack); err != nil {
				return err
			}
			if err := paysForTheProposal(inUnit, u, s); err != nil {
				return err
			}
			moved = s.PendingAttack.Paid.CharacterID
			return stampTheSheet(inUnit, u, moved)
		})
	if err != nil {
		return nil, err
	}
	st.bus.Publish(events.AttackRolled{SessionID: sessionID})
	st.wakeTheSheet(moved)
	return clone, nil
}

// RerollAttack troca o dado da proposta pelo do "rolar de novo" (p226), e o PM dele sai na
// mesma transação. Trocados em dois passos, dois toques com PM para um só listariam dois.
func (st *Store) RerollAttack(ctx context.Context, sessionID int64, attack live.PendingAttack) (*live.SessionRuntimeState, error) {
	var moved *int64
	clone, err := st.unitGesture(ctx, sessionID,
		func(inUnit context.Context, u Unit, s *live.SessionRuntimeState) error {
			if err := live.ReplaceAttackRoll(s, attack); err != nil {
				return err
			}
			moved = attackerCharacterOf(s)
			paid := attack.Rerolls[len(attack.Rerolls)-1]
			if err := spendsOnTheAttack(inUnit, u, s, moved, paid.PM); err != nil {
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

// paysForTheProposal cobra a ação, de quem ataca SE ele está na vez, e o PM do impulso.
func paysForTheProposal(ctx context.Context, u Unit, s *live.SessionRuntimeState) error {
	attack := s.PendingAttack
	paid := &live.AttackPaid{CharacterID: attackerCharacterOf(s), Round: s.Round}
	attack.Paid = paid
	if s.CountsRounds() && attackerIsOnTurn(s, *attack) {
		before := *s.Scene
		if err := chargeTurn(s, engine.StrikeAction(attack.Charge)); err != nil {
			return err
		}
		paid.Scene = s.Scene.Number
		paid.Standard = before.StandardLeft && !s.Scene.StandardLeft
		paid.Movement = before.MovementLeft && !s.Scene.MovementLeft
		paid.ExtraMovement = before.ExtraMovementLeft && !s.Scene.ExtraMovementLeft
	}
	return spendsOnTheAttack(ctx, u, s, paid.CharacterID, attack.BoostPM)
}

// spendsOnTheAttack cobra PM de quem ataca e espelha na fila. Ele confere sob a trava: o
// poço tem piso em zero, e sem isto o gasto que a ficha deixou de poder pagar sairia de
// graça.
func spendsOnTheAttack(ctx context.Context, u Unit, s *live.SessionRuntimeState, characterID *int64, pm int) error {
	if pm <= 0 || characterID == nil {
		return nil
	}
	pools, err := u.Sheet.PoolsOf(ctx, []int64{*characterID})
	if err != nil {
		return fmt.Errorf("ler o PM de quem ataca: %w", err)
	}
	pool := pools[*characterID]
	if available := pool.ManaAvailable(); int64(pm) > available {
		return fmt.Errorf("o golpe pede %d PM, e sobram %d", pm, available)
	}
	hp, mp, receipt, err := u.Sheet.SpendManaKeepingReceipt(ctx, *characterID, pm)
	if err != nil {
		return fmt.Errorf("cobrar o PM do golpe: %w", err)
	}
	attack := s.PendingAttack
	if attack.Paid == nil {
		attack.Paid = &live.AttackPaid{CharacterID: characterID}
	}
	attack.Paid.Mana = append(attack.Paid.Mana, receipt)
	entry := attack.AttackerEntryID
	return live.PatchEntryVitals(s, entry, hp, mp, hpFloorOf(s, entry))
}

// attackerIsOnTurn diz se quem atacou é a linha da vez em curso.
func attackerIsOnTurn(s *live.SessionRuntimeState, attack live.PendingAttack) bool {
	if s == nil || s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return false
	}
	return s.Initiative[s.TurnIndex].ID == attack.AttackerEntryID
}

// CancelAttack descarta o provisório sem ferir ninguém, e o que ele pagou volta.
func (st *Store) CancelAttack(ctx context.Context, sessionID int64, who live.Attacker) (*live.SessionRuntimeState, error) {
	var moved *int64
	clone, err := st.unitGesture(ctx, sessionID,
		func(inUnit context.Context, u Unit, s *live.SessionRuntimeState) error {
			open := s.PendingAttack
			if err := live.CancelAttack(s, who); err != nil {
				return err
			}
			var err error
			moved, err = givesTheProposalBack(inUnit, u, s, open)
			return err
		})
	if err != nil {
		return nil, err
	}
	st.bus.Publish(events.AttackSettled{SessionID: sessionID})
	st.wakeTheSheet(moved)
	return clone, nil
}

// givesTheProposalBack desfaz o que a proposta pagou, e devolve a ficha que mudou. O golpe
// que sai da mesa sem resposta não aconteceu: a ação, o PM e a dose voltam. O erro já
// aconteceu, e fechar a leitura dele não devolve nada.
func givesTheProposalBack(ctx context.Context, u Unit, s *live.SessionRuntimeState, open *live.PendingAttack) (*int64, error) {
	if open == nil || open.Ruling == live.RulingMiss {
		return nil, nil
	}
	givesTheTurnBack(s, *open)
	if err := returnsTheDose(ctx, u, open.Dose); err != nil {
		return nil, err
	}
	if err := givesTheManaBack(ctx, u, s, *open); err != nil {
		return nil, err
	}
	if open.Paid == nil {
		return nil, nil
	}
	return open.Paid.CharacterID, stampTheSheet(ctx, u, open.Paid.CharacterID)
}

// givesTheTurnBack devolve as ações só ao turno de onde elas saíram. Na vez de outra
// pessoa, ou na rodada seguinte, elas seriam uma ação a mais que ninguém pagou.
func givesTheTurnBack(s *live.SessionRuntimeState, open live.PendingAttack) {
	paid := open.Paid
	if paid == nil || !s.CountsRounds() || !attackerIsOnTurn(s, open) {
		return
	}
	if s.Scene.Number != paid.Scene || s.Round != paid.Round {
		return
	}
	s.Scene.StandardLeft = s.Scene.StandardLeft || paid.Standard
	s.Scene.MovementLeft = s.Scene.MovementLeft || paid.Movement
	s.Scene.ExtraMovementLeft = s.Scene.ExtraMovementLeft || paid.ExtraMovement
}

// givesTheManaBack desfaz cada recibo: o PM do impulso e dos "rolar de novo" volta de onde
// saiu, ao poço ou à poça temporária.
func givesTheManaBack(ctx context.Context, u Unit, s *live.SessionRuntimeState, open live.PendingAttack) error {
	if open.Paid == nil || open.Paid.CharacterID == nil {
		return nil
	}
	for _, receipt := range open.Paid.Mana {
		hp, mp, err := u.Sheet.GiveManaBack(ctx, *open.Paid.CharacterID, receipt)
		// A FICHA APAGADA não tem para onde receber, e isso não pode prender o golpe na mesa.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("devolver o PM do golpe cancelado: %w", err)
		}
		if live.FindEntryIndex(s, open.AttackerEntryID) < 0 {
			continue
		}
		if err := live.PatchEntryVitals(s, open.AttackerEntryID, hp, mp, hpFloorOf(s, open.AttackerEntryID)); err != nil {
			return err
		}
	}
	return nil
}
