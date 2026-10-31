package combat

import (
	"context"
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O SEGUNDO TEMPO DO GOLPE: a decisão do mestre e o dano que ela pede. O
// crítico muda quantos dados se rolam (p231), e quem diz se houve crítico é o
// mestre — por isso o dano só existe depois da palavra dele.

// Rule guarda a decisão do mestre sobre o golpe proposto. No acerto e no
// crítico, ela já diz ao jogador o que rolar, com o ×N da arma.
//
//	state, err := strike.Rule(ctx, gm, "gm", req, live.RulingCritical)
func (s Strike) Rule(ctx context.Context, who app.Caller, role string, req Request, ruling string) (*live.SessionRuntimeState, error) {
	var toRoll *live.DamageToRoll
	if ruling == live.RulingHit || ruling == live.RulingCritical {
		setup, err := s.setUp(ctx, role, req)
		if err != nil {
			return nil, err
		}
		formula, err := engine.DamageToRoll(setup.weapon, engine.Ruling(ruling))
		if err != nil {
			return nil, err
		}
		toRoll = damageToRollOf(formula)
	}
	return s.tables.RuleAttack(ctx, req.SessionID, live.Attacker{UserID: who.ID, Role: role}, ruling, toRoll)
}

// RollDamage rola o dano do golpe que o mestre decidiu. `hand` são os dados que
// a mesa rolou na mão, na ordem da fórmula; nulo é o servidor rolar.
//
//	pending, err := strike.RollDamage(ctx, player, "player", req, []int{5, 3})
func (s Strike) RollDamage(ctx context.Context, who app.Caller, role string, req Request, hand []int) (live.PendingAttack, error) {
	attacker := live.Attacker{UserID: who.ID, Role: role}
	state, err := s.tables.State(ctx, req.SessionID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if state == nil {
		return live.PendingAttack{}, fmt.Errorf("a sessão %d não tem mesa aberta: %w", req.SessionID, app.ErrNotFound)
	}
	pending, err := live.AttackAwaitingDamage(state, attacker)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	setup, err := s.setUp(ctx, role, req)
	if err != nil {
		return live.PendingAttack{}, err
	}
	rollDie := s.rollDie
	var byHand *engine.HandDice
	if hand != nil {
		byHand = engine.DiceRolledByHand(hand)
		rollDie = byHand.Roll
	}
	out, err := engine.RollAttackDamage(setup.weapon, setup.target, setup.situations,
		engine.D20(pending.Roll), engine.Ruling(pending.Ruling), rollDie)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	if byHand != nil && byHand.Unused() > 0 {
		return live.PendingAttack{}, fmt.Errorf("foram digitados %d dados a mais do que o dano pede: %w",
			byHand.Unused(), app.ErrRefused)
	}
	pending.Dice, pending.Faces, pending.ExtraDice = out.Dice, out.Faces, extraDiceOf(out)
	pending.RawDamage, pending.Absorbed, pending.Damage, pending.NonLethal = out.RawDamage, out.Absorbed, out.Damage, out.NonLethal
	pending.CriticalExtra, pending.CritExtraIgnoreChance = out.CriticalExtra, out.CritExtraIgnoreChance
	pending.Announces = foreseenOf(out)
	if _, err := s.tables.RecordAttackDamage(ctx, req.SessionID, attacker, pending); err != nil {
		return live.PendingAttack{}, err
	}
	return pending, nil
}

// damageToRollOf copia a fórmula do motor para a forma que o regime entende.
func damageToRollOf(formula engine.DamageFormula) *live.DamageToRoll {
	toRoll := &live.DamageToRoll{Bonus: formula.Bonus}
	for _, group := range formula.Dice {
		toRoll.Dice = append(toRoll.Dice, live.DiceGroup{Count: group.Count, Faces: group.Faces, Type: group.Type})
	}
	return toRoll
}

func extraDiceOf(out engine.AttackOutcome) []live.RolledGroup {
	var extra []live.RolledGroup
	for _, roll := range out.Extra {
		extra = append(extra, live.RolledGroup{Dice: roll.Dice, Faces: roll.Faces, Type: roll.Type})
	}
	return extra
}
