package combat

import (
	"context"
	"fmt"
	"slices"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// RerollLowDamage é o Destruidor (p41, p65): rola de novo os 1 e 2 dos dados da ARMA do dano já rolado, e refaz
// a conta inteira — a RD, o corpo do alvo, o dano final — com os dados novos da arma e os extras como saíram.
// `hand` são os dados que a mesa rolou de novo na mão, um por dado a rolar, na ordem; nulo é o servidor rolar.
//
//	pending, err := strike.RerollLowDamage(ctx, player, "player", sessionID, []int{5})
func (s Strike) RerollLowDamage(ctx context.Context, who app.Caller, role string, sessionID int64, hand []int) (live.PendingAttack, error) {
	attacker := live.Attacker{UserID: who.ID, Role: role}
	state, err := s.tables.State(ctx, sessionID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if state == nil || state.PendingAttack == nil {
		return live.PendingAttack{}, fmt.Errorf("não há ataque na mesa para rolar o dano de novo: %w", app.ErrRefused)
	}
	pending := *state.PendingAttack
	if pending.DamageRerolledFrom != nil {
		return live.PendingAttack{}, fmt.Errorf("o dano deste ataque já foi rolado de novo, e vale o segundo resultado: %w",
			app.ErrRefused)
	}
	frozen, err := strikeProposedIn(pending)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if !frozen.Weapon.RerollsLowDice {
		return live.PendingAttack{}, fmt.Errorf("o %s deste golpe não rola de novo os 1 e 2: o Destruidor é de arma corpo a corpo "+
			"de duas mãos: %w", frozen.Weapon.Name, app.ErrRefused)
	}
	low := engine.LowWeaponDice(pending.Dice)
	if low == 0 {
		return live.PendingAttack{}, fmt.Errorf("nenhum dado da arma saiu 1 ou 2 (%v): não há o que rolar de novo: %w",
			pending.Dice, app.ErrRefused)
	}
	rollDie := s.rollDie
	if hand != nil {
		if len(hand) != low {
			return live.PendingAttack{}, fmt.Errorf("foram digitados %d dados, e são %d dado(s) da arma a rolar de novo: %w",
				len(hand), low, app.ErrRefused)
		}
		rollDie = engine.DiceRolledByHand(hand).Roll
	}
	weapon, err := engine.RerollLowWeaponDice(pending.Dice, pending.Faces, rollDie)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	// A CONTA REFEITA é a mesma do dano rolado, com os dados entregues na ordem da fórmula: a arma primeiro, os
	// extras depois, como saíram.
	replay := slices.Clone(weapon)
	for _, group := range pending.ExtraDice {
		replay = append(replay, group.Dice...)
	}
	out, err := engine.RollAttackDamage(frozen.Weapon, frozen.Target, frozen.Situations,
		engine.D20(pending.Roll), engine.Ruling(pending.Ruling), engine.DiceRolledByHand(replay).Roll)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	before := pending.Dice
	pending = withRolledDamage(pending, out)
	pending.DamageRerolledFrom = before
	if _, err := s.tables.RerollAttackDamage(ctx, sessionID, attacker, pending); err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	return pending, nil
}
