package table

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O DESTRUIDOR NA CARTA (bárbaro p41, guerreiro p65): depois do dano rolado e antes do ✓ do mestre, quem rolou
// rola de novo os 1 e 2 dos dados da arma — no app, ou na mão, um campo por dado (decisões do dono). A regra é do
// motor e a conta é do `combat.Strike.RerollLowDamage`; aqui moram o botão, os campos e a frase.

// destroyerFields são os campos da mão do Destruidor para quem rolou, ou nil quando ele não se oferece: a arma
// não rola de novo, já rolou, ou nenhum dado dela saiu 1 ou 2.
func destroyerFields(pa live.PendingAttack, mine bool) []handGroup {
	low := engine.LowWeaponDice(pa.Dice)
	if !mine || !pa.DamageRolled || !pa.RerollsLowDice || pa.DamageRerolledFrom != nil || low == 0 || pa.Faces == 0 {
		return nil
	}
	group := handGroup{Label: fmt.Sprintf("%dd%d", low, pa.Faces), Faces: pa.Faces}
	for i := range low {
		group.Signals = append(group.Signals, handDamageSignal(i+1))
	}
	return []handGroup{group}
}

// rerolledWritten é o que o Destruidor mudou, dado a dado: "1 → 5, 2 → 1". Vazio quando ele não foi usado.
func rerolledWritten(before, after []int) string {
	var changed []string
	for i, value := range before {
		if i < len(after) && (value == 1 || value == 2) {
			changed = append(changed, strconv.Itoa(value)+" → "+strconv.Itoa(after[i]))
		}
	}
	return strings.Join(changed, ", ")
}

// rerollsLowDamage é o Destruidor rolado no app.
func rerollsLowDamage(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	return st.rerollLowDamage(c, nil)
}

// rerollsLowDamageByHand é o Destruidor com os dados que a mesa rolou de novo e digitou, um por dado a rolar.
func rerollsLowDamageByHand(st Scene, c commandCtx) (*live.SessionRuntimeState, error) {
	state, err := st.deps.Sessions().State(c.R.Context(), c.SessionID)
	if err != nil {
		return nil, err
	}
	if state == nil || state.PendingAttack == nil {
		return nil, errors.New("não há ataque na mesa para rolar o dano de novo")
	}
	hand, err := handDamageDice(c, engine.LowWeaponDice(state.PendingAttack.Dice))
	if err != nil {
		return nil, err
	}
	return st.rerollLowDamage(c, hand)
}

func (st Scene) rerollLowDamage(c commandCtx, hand []int) (*live.SessionRuntimeState, error) {
	if _, err := st.strike.RerollLowDamage(c.R.Context(), app.Caller{ID: c.User}, c.Role, c.SessionID, hand); err != nil {
		return nil, err
	}
	return st.deps.Sessions().State(c.R.Context(), c.SessionID)
}
