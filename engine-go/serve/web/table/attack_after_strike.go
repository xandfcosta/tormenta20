package table

import (
	"slices"

	"t20engine/app/character"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// O QUE OS PODERES TIRAM DO GOLPE ENCERRADO: o Roubo de Mana do ladino e o Até Acertar do
// lutador. Os dois saem do `afterTheStrike`, depois que o mestre disse o que o golpe foi.

// stealsMana é o Roubo de Mana (p74): o golpe confirmado que CAUSOU dano com o furtivo ligado dá
// ao ladino 1 PM temporário por dado, e tira 1 PM por dado da ficha atrás do alvo. Uma vez por
// cena contra a mesma criatura — a marca é da cena, e só se marca quem tem o poder.
func (st Scene) stealsMana(c commandCtx, state *live.SessionRuntimeState, pendente *live.PendingAttack) error {
	if pendente == nil || !pendente.Hit || pendente.SneakDice <= 0 || pendente.Damage <= 0 {
		return nil
	}
	attacker := entryByID(state, pendente.AttackerEntryID)
	if attacker == nil || attacker.CharacterID == nil {
		return nil
	}
	has, err := st.plays.HasPower(c.R.Context(), *attacker.CharacterID, character.ManaTheftPowerID)
	if err != nil || !has {
		return err
	}
	fresh, err := st.deps.Sessions().MarkManaStolen(c.R.Context(), c.SessionID, pendente.AttackerEntryID, pendente.TargetEntryID)
	if err != nil || !fresh {
		return err
	}
	var victim int64
	if target := entryByID(state, pendente.TargetEntryID); target != nil && target.CharacterID != nil {
		victim = *target.CharacterID
	}
	if err := st.plays.StealMana(c.R.Context(), *attacker.CharacterID, victim, pendente.SneakDice); err != nil {
		return err
	}
	st.deps.CharacterChanged(c.R.Context(), *attacker.CharacterID)
	if victim != 0 {
		st.deps.CharacterChanged(c.R.Context(), victim)
	}
	return nil
}

// countsTheUnarmedMiss é o Até Acertar (p76): o golpe desarmado que o mestre disse que ERROU soma um
// erro contra aquele oponente, e QUALQUER acerto do lutador apaga os erros dele — "os bônus
// terminam quando você acertar um ataque".
func (st Scene) countsTheUnarmedMiss(c commandCtx, state *live.SessionRuntimeState, pendente *live.PendingAttack) error {
	if pendente == nil {
		return nil
	}
	if pendente.Hit {
		if !hasUntilHitMarks(state, pendente.AttackerEntryID) {
			return nil
		}
		_, err := st.deps.Sessions().ClearUntilHit(c.R.Context(), c.SessionID, pendente.AttackerEntryID)
		return err
	}
	if pendente.Ruling != live.RulingMiss || !pendente.Unarmed {
		return nil
	}
	attacker := entryByID(state, pendente.AttackerEntryID)
	if attacker == nil || attacker.CharacterID == nil {
		return nil
	}
	has, err := st.plays.HasPower(c.R.Context(), *attacker.CharacterID, engine.UntilHitPowerID)
	if err != nil || !has {
		return err
	}
	_, err = st.deps.Sessions().MarkUnarmedMiss(c.R.Context(), c.SessionID, pendente.AttackerEntryID, pendente.TargetEntryID)
	return err
}

// hasUntilHitMarks diz se o lutador tem erro marcado contra alguém: só então o acerto precisa apagar.
func hasUntilHitMarks(state *live.SessionRuntimeState, attacker string) bool {
	if state == nil || !state.InScene() {
		return false
	}
	return slices.ContainsFunc(state.Scene.UntilHit, func(m live.UntilHitMark) bool { return m.Attacker == attacker })
}
