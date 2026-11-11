package session

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

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
	if s.Scene != nil {
		paid.Scene = s.Scene.Number
	}
	if err := takesTheEnchantBoostOfTheRound(s, *attack); err != nil {
		return err
	}
	if attack.ExtraOf != "" {
		if err := takesTheExtraStrike(s, attack); err != nil {
			return err
		}
	} else if s.CountsRounds() && attackerIsOnTurn(s, *attack) {
		before := *s.Scene
		if err := chargeStrike(s, *attack); err != nil {
			return err
		}
		paid.Standard = before.StandardLeft && !s.Scene.StandardLeft
		paid.Movement = before.MovementLeft && !s.Scene.MovementLeft
		paid.ExtraMovement = before.ExtraMovementLeft && !s.Scene.ExtraMovementLeft
	}
	if err := spendsOnTheAttack(ctx, u, s, paid.CharacterID, attack.BoostPM); err != nil {
		return err
	}
	return losesTheBoostHitPoints(ctx, u, s, paid)
}

// losesTheBoostHitPoints tira de quem ataca o PV que o gasto cobra — o 1d8 da Tumular, "você perde 1d8
// pontos de vida" (p336) —, pela ordem de toda perda: os temporários saem primeiro (p106), e quem chega a 0
// cai (p236). O recibo de onde cada ponto saiu fica na proposta, para o cancelamento devolver.
func losesTheBoostHitPoints(ctx context.Context, u Unit, s *live.SessionRuntimeState, paid *live.AttackPaid) error {
	attack := s.PendingAttack
	if attack.BoostHitPoints <= 0 || paid.CharacterID == nil {
		return nil
	}
	hp, mp, receipt, err := u.Sheet.LoseHitPointsKeepingReceipt(ctx, *paid.CharacterID, attack.BoostHitPoints)
	if err != nil {
		return fmt.Errorf("tirar o PV do gasto do golpe: %w", err)
	}
	paid.HitPoints = receipt
	entry := attack.AttackerEntryID
	return live.PatchEntryVitals(s, entry, hp, mp, hpFloorOf(s, entry))
}

// takesTheEnchantBoostOfTheRound confere e conta o gasto de 2 PM do encanto: "uma vez por rodada" (p335-336),
// recusado com a iniciativa correndo (decisão do dono). Fora dela não há rodada para contar. Ele vem antes
// da cobrança da ação, para o segundo golpe pago da vez ouvir a regra do encanto, e não "falta ação".
func takesTheEnchantBoostOfTheRound(s *live.SessionRuntimeState, attack live.PendingAttack) error {
	if _, _, isEnchant := engine.EnchantBoostParts(attack.BoostPowerID); !isEnchant || !s.CountsRounds() {
		return nil
	}
	if !live.UsesOncePerRound(s, attack.AttackerEntryID, attack.BoostPowerID) {
		return fmt.Errorf("o gasto de 2 PM do encanto de %s é uma vez por rodada (p335-336), e já foi feito nesta: ataque sem ele, ou espere a próxima vez", attack.Weapon)
	}
	return nil
}

// chargeStrike cobra a ação do golpe. O Disparo Rápido é agredir com a ação completa (p125), e a
// recusa diz isso: quem já andou ainda pode atirar, desligando o interruptor.
func chargeStrike(s *live.SessionRuntimeState, attack live.PendingAttack) error {
	if !attack.FullAction {
		return chargeTurn(s, engine.StrikeAction(attack.Charge))
	}
	if err := chargeTurn(s, engine.ActionFull); err != nil {
		return fmt.Errorf("o Disparo Rápido gasta a ação completa, e ela já não cabe nesta vez: desligue-o para atirar com a ação padrão: %w", err)
	}
	return nil
}

// takesTheExtraStrike confere e conta o ataque extra na vez, sob a trava. Ele não gasta ação.
func takesTheExtraStrike(s *live.SessionRuntimeState, attack *live.PendingAttack) error {
	if !s.CountsRounds() || s.Scene.ExtraStrikeEntry == "" || s.Scene.ExtraStrikeEntry != attack.AttackerEntryID ||
		!attackerIsOnTurn(s, *attack) {
		return fmt.Errorf("o ataque com %s vem depois de agredir nesta vez, e da resposta do mestre", attack.Weapon)
	}
	if attack.ExtraIsStyle {
		if s.Scene.ExtraStyle != attack.ExtraOf {
			return fmt.Errorf("o ataque adicional com %s pede o estilo ligado antes de agredir (p125), porque o -2 vale nos dois", attack.Weapon)
		}
		if sameWeapon := s.Scene.ExtraStyleItem == attack.StyleItem; sameWeapon == attack.ExtraOtherWeapon {
			if attack.ExtraOtherWeapon {
				return fmt.Errorf("o ataque adicional é um com cada arma (p125), e %s acabou de atacar: use a outra arma", attack.Weapon)
			}
			return fmt.Errorf("o disparo adicional é com a mesma arma do primeiro (p125), e ele não foi com %s", attack.Weapon)
		}
	} else if s.Scene.ExtraStrikeBarred == attack.ExtraOf {
		return fmt.Errorf("o ataque extra com %s pede que o golpe tenha sido com outra arma", attack.Weapon)
	}
	if timesIn(s.Scene.ExtraStrikesUsed, attack.ExtraOf) >= attack.ExtraLimit {
		return fmt.Errorf("o ataque extra com %s é uma vez por rodada", attack.Weapon)
	}
	// A PATA entra na mesma lista, numa entrada só dela: a contagem pela fonte continua exata, e as
	// Ações leem as duas do mesmo sinal.
	if attack.ExtraLimb != "" && timesIn(s.Scene.ExtraStrikesUsed, attack.ExtraLimb) >= attack.ExtraLimbLimit {
		return fmt.Errorf("o ataque extra das patas é um com cada uma (p137), e %s já atacou nesta rodada", attack.Weapon)
	}
	s.Scene.ExtraStrikesUsed = append(s.Scene.ExtraStrikesUsed, attack.ExtraOf)
	if attack.ExtraLimb != "" {
		s.Scene.ExtraStrikesUsed = append(s.Scene.ExtraStrikesUsed, attack.ExtraLimb)
	}
	return nil
}

// timesIn é quantas vezes a entrada aparece nos extras da vez.
func timesIn(used []string, entry string) int {
	times := 0
	for _, each := range used {
		if each == entry {
			times++
		}
	}
	return times
}

// opensTheExtraStrike é o que a resposta do mestre deixa: agredir abre o ataque extra das outras
// armas naturais (p25), se a vez ainda é a do turno em que o golpe nasceu.
func opensTheExtraStrike(s *live.SessionRuntimeState, attack live.PendingAttack) {
	if attack.OpensExtra && paidInThisTurn(s, attack) {
		s.Scene.ExtraStrikeEntry, s.Scene.ExtraStrikeBarred = attack.AttackerEntryID, attack.NaturalSource
		s.Scene.ExtraStyle, s.Scene.ExtraStyleItem = attack.OpensStyle, attack.StyleItem
	}
}

// paidInThisTurn diz se a proposta nasceu neste turno. Na vez de outra pessoa, ou na rodada
// seguinte, a vez que volta já é outro turno.
func paidInThisTurn(s *live.SessionRuntimeState, attack live.PendingAttack) bool {
	paid := attack.Paid
	return paid != nil && s.CountsRounds() && attackerIsOnTurn(s, attack) &&
		s.Scene.Number == paid.Scene && s.Round == paid.Round
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
	live.GivesOncePerRoundBack(s, open.AttackerEntryID, open.BoostPowerID)
	if err := returnsTheDose(ctx, u, open.Dose); err != nil {
		return nil, err
	}
	if err := givesTheManaBack(ctx, u, s, *open); err != nil {
		return nil, err
	}
	if err := givesTheHitPointsBack(ctx, u, s, *open); err != nil {
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
	if !paidInThisTurn(s, open) {
		return
	}
	for _, entry := range []string{open.ExtraOf, open.ExtraLimb} {
		if at := slices.Index(s.Scene.ExtraStrikesUsed, entry); entry != "" && at >= 0 {
			s.Scene.ExtraStrikesUsed = slices.Delete(s.Scene.ExtraStrikesUsed, at, at+1)
		}
	}
	paid := open.Paid
	s.Scene.StandardLeft = s.Scene.StandardLeft || paid.Standard
	s.Scene.MovementLeft = s.Scene.MovementLeft || paid.Movement
	s.Scene.ExtraMovementLeft = s.Scene.ExtraMovementLeft || paid.ExtraMovement
}

// givesTheHitPointsBack devolve o PV que o gasto tirou (a Tumular), pelo recibo: o que saiu da poça
// temporária volta para ela, e não ao PV comum, onde viraria ponto que não expira.
func givesTheHitPointsBack(ctx context.Context, u Unit, s *live.SessionRuntimeState, open live.PendingAttack) error {
	if open.Paid == nil || open.Paid.CharacterID == nil || len(open.Paid.HitPoints) == 0 {
		return nil
	}
	hp, mp, err := u.Sheet.GiveHitPointsBack(ctx, *open.Paid.CharacterID, open.Paid.HitPoints)
	// A FICHA APAGADA não tem para onde receber, e isso não pode prender o golpe na mesa.
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("devolver o PV do golpe cancelado: %w", err)
	}
	if live.FindEntryIndex(s, open.AttackerEntryID) < 0 {
		return nil
	}
	return live.PatchEntryVitals(s, open.AttackerEntryID, hp, mp, hpFloorOf(s, open.AttackerEntryID))
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
