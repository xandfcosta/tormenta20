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
//	state, err := strike.Rule(ctx, gm, "gm", sessionID, live.RulingCritical)
func (s Strike) Rule(ctx context.Context, who app.Caller, role string, sessionID int64, ruling string) (*live.SessionRuntimeState, error) {
	var toRoll *live.DamageToRoll
	if ruling == live.RulingHit || ruling == live.RulingCritical {
		state, err := s.tables.State(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		// Sem proposta, ou numa manobra, quem responde é a regra lá embaixo.
		if state != nil && state.PendingAttack != nil && state.PendingAttack.Maneuver == nil {
			frozen, err := strikeProposedIn(*state.PendingAttack)
			if err != nil {
				return nil, err
			}
			toRoll, err = damageLinesOf(frozen.Weapon, engine.Ruling(ruling))
			if err != nil {
				return nil, err
			}
		}
	}
	return s.tables.RuleAttack(ctx, sessionID, live.Attacker{UserID: who.ID, Role: role}, ruling, toRoll)
}

// strikeProposedIn lê o golpe congelado na proposta. A que veio de antes de ele existir
// não tem o que ler, e recalcular pela ficha de agora seria mudar a proposta.
func strikeProposedIn(pending live.PendingAttack) (engine.FrozenStrike, error) {
	if len(pending.Frozen) == 0 {
		return engine.FrozenStrike{}, fmt.Errorf(
			"este ataque foi proposto antes de uma atualização do app: o mestre cancela, e ele é rolado de novo: %w", app.ErrRefused)
	}
	return engine.ThawStrike(pending.Frozen)
}

// RollDamage rola o dano do golpe que o mestre decidiu. `hand` são os dados que
// a mesa rolou na mão, na ordem da fórmula; nulo é o servidor rolar.
//
//	pending, err := strike.RollDamage(ctx, player, "player", sessionID, []int{5, 3})
func (s Strike) RollDamage(ctx context.Context, who app.Caller, role string, sessionID int64, hand []int) (live.PendingAttack, error) {
	attacker := live.Attacker{UserID: who.ID, Role: role}
	state, err := s.tables.State(ctx, sessionID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if state == nil {
		return live.PendingAttack{}, fmt.Errorf("a sessão %d não tem mesa aberta: %w", sessionID, app.ErrNotFound)
	}
	pending, err := live.AttackAwaitingDamage(state, attacker)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	frozen, err := strikeProposedIn(pending)
	if err != nil {
		return live.PendingAttack{}, err
	}
	// A LINHA ESCOLHIDA é a carta que se rola: o dado um passo acima, e o resto igual (ALE-655).
	if pending.DamageToRoll != nil {
		if frozen.Weapon, err = engine.WithDamageVariant(frozen.Weapon, pending.DamageToRoll.Line); err != nil {
			return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
		}
	}
	rollDie := s.rollDie
	var byHand *engine.HandDice
	if hand != nil {
		byHand = engine.DiceRolledByHand(hand)
		rollDie = byHand.Roll
	}
	out, err := engine.RollAttackDamage(frozen.Weapon, frozen.Target, frozen.Situations,
		engine.D20(pending.Roll), engine.Ruling(pending.Ruling), rollDie)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	if byHand != nil && byHand.Unused() > 0 {
		return live.PendingAttack{}, fmt.Errorf("foram digitados %d dados a mais do que o dano pede: %w",
			byHand.Unused(), app.ErrRefused)
	}
	pending = withRolledDamage(pending, out)
	if _, err := s.tables.RecordAttackDamage(ctx, sessionID, attacker, pending); err != nil {
		return live.PendingAttack{}, err
	}
	return pending, nil
}

// withRolledDamage é o provisório com o dano que o motor rolou: os dados, a conta e o que o livro prevê.
func withRolledDamage(pending live.PendingAttack, out engine.AttackOutcome) live.PendingAttack {
	pending.Dice, pending.Faces, pending.ExtraDice = out.Dice, out.Faces, extraDiceOf(out)
	pending.RawDamage, pending.Absorbed, pending.Damage, pending.NonLethal = out.RawDamage, out.Absorbed, out.Damage, out.NonLethal
	pending.CriticalExtra, pending.CritExtraIgnoreChance = out.CriticalExtra, out.CritExtraIgnoreChance
	pending.Announces = foreseenOf(out)
	pending.Taken = nil
	for _, row := range out.Taken {
		pending.Taken = append(pending.Taken, live.TakenRow{Label: row.Label, Amount: row.Amount})
	}
	return pending
}

// damageLinesOf é o dano a rolar com as linhas da carta: o normal marcado, e uma fórmula por situação (ALE-655).
// Sem situação na carta, só o normal, e sem a lista.
func damageLinesOf(card engine.WeaponCard, ruling engine.Ruling) (*live.DamageToRoll, error) {
	formula, err := engine.DamageToRoll(card, ruling)
	if err != nil {
		return nil, err
	}
	toRoll := damageToRollOf(formula)
	if len(card.DamageVariants) == 0 {
		return toRoll, nil
	}
	toRoll.Lines = []live.DamageLine{{Label: "normal", Dice: toRoll.Dice, Bonus: toRoll.Bonus}}
	for _, variant := range card.DamageVariants {
		varied, err := engine.WithDamageVariant(card, variant.ID)
		if err != nil {
			return nil, err
		}
		formula, err := engine.DamageToRoll(varied, ruling)
		if err != nil {
			return nil, err
		}
		line := damageToRollOf(formula)
		toRoll.Lines = append(toRoll.Lines, live.DamageLine{
			ID: variant.ID, Label: variant.Label, Source: variant.Source, Dice: line.Dice, Bonus: line.Bonus,
		})
	}
	return toRoll, nil
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

// ChooseDamageLine marca a linha do dano que quem atacou vai rolar: o normal, ou uma situação da carta (a Arma
// Improvisada, a Chave, p76). O "Rolar" e o "rolei na mão" seguem a linha marcada.
//
//	state, err := strike.ChooseDamageLine(ctx, player, "player", sessionID, "class.lutador.chave")
func (s Strike) ChooseDamageLine(ctx context.Context, who app.Caller, role string, sessionID int64, line string) (*live.SessionRuntimeState, error) {
	state, err := s.tables.ChooseDamageLine(ctx, sessionID, live.Attacker{UserID: who.ID, Role: role}, line)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	return state, nil
}
