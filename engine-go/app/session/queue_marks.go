package session

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// OS GESTOS DA MARCA NA FILA (ver GLOSSARY, **marca na fila**): o mestre marca quem falhou na resistência, tira a
// marca, e rola na linha o dado que se rola na vez dela. A regra — o que se rola, quando, e por quantos turnos — é
// do `live`; aqui estão a vez, as condições da ficha e o dado.

// MarkEntry põe na linha a marca da última magia conjurada que deixa marca.
func (st *Store) MarkEntry(ctx context.Context, sessionID int64, entryID string) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.CombatantChanged{SessionID: sessionID, EntryID: entryID},
		func(s *live.SessionRuntimeState) error { return live.MarkEntry(s, entryID) })
}

// UnmarkEntry tira da linha a marca da magia.
func (st *Store) UnmarkEntry(ctx context.Context, sessionID int64, entryID, spell string) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.CombatantChanged{SessionID: sessionID, EntryID: entryID},
		func(s *live.SessionRuntimeState) error { return live.UnmarkEntry(s, entryID, spell) })
}

// DueRollsOf são os dados que se rolam na vez da linha (`live.DueRolls`), com as condições de onde elas moram: as
// do PC na ficha, as do NPC na linha.
func (st *Store) DueRollsOf(ctx context.Context, entry live.InitiativeEntry) ([]live.DueRoll, error) {
	conditions, err := conditionsOfEntry(ctx, st.turnEffects, entry)
	if err != nil {
		return nil, err
	}
	return live.DueRolls(entry, conditions), nil
}

// conditionsOfEntry são as condições de uma linha da fila, de onde elas moram: as do PC na ficha, as do NPC na
// linha. Sem porta para a ficha, ficam as da linha.
func conditionsOfEntry(ctx context.Context, effects live.SheetTurnEffects, entry live.InitiativeEntry) ([]string, error) {
	if entry.CharacterID == nil || effects == nil {
		return entry.Conditions, nil
	}
	conditions, err := effects.ConditionsOf(ctx, *entry.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("ler as condições de %s: %w", entry.Label, err)
	}
	return conditions, nil
}

// RollDue rola o n-ésimo dado da vez da linha, ou recebe o total que a mesa rolou na mão para cada parcela, e o
// põe na faixa da magia. Só na vez dela: é o momento que o livro diz, e rolar o muco do ogro na vez do anão seria
// a mesa contando um turno que não aconteceu.
func (st *Store) RollDue(
	ctx context.Context, sessionID int64, entryID string, n int, fromTheTable []*int, byUserID int64,
	rollDie func(faces int) (int, error),
) (*live.SessionRuntimeState, error) {
	state, err := st.State(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if state.TurnIndex < 0 || state.TurnIndex >= len(state.Initiative) || state.Initiative[state.TurnIndex].ID != entryID {
		return nil, fmt.Errorf("o dado da marca se rola na vez de quem está marcado, e a vez não é da linha %q", entryID)
	}
	entry := state.Initiative[state.TurnIndex]
	due, err := st.DueRollsOf(ctx, entry)
	if err != nil {
		return nil, err
	}
	if n < 0 || n >= len(due) {
		return nil, fmt.Errorf("%s tem %d dado(s) na vez, e o pedido foi o %d", entry.Label, len(due), n+1)
	}
	asked := due[n]
	roll := live.SpellCastRoll{
		Kind: live.RollOfTheMark, Who: asked.Who, Spell: asked.Spell, Moment: asked.At, Target: asked.Target,
		Save: asked.Save, DC: asked.DC, ByUserID: byUserID, Bonuses: asked.Bonuses,
	}
	if asked.Condition != "" {
		roll.Kind, roll.Condition = live.RollOfTheCondition, asked.Condition
	}
	if entry.CharacterID != nil {
		roll.CharacterID = *entry.CharacterID
	}
	for i, part := range asked.Parts {
		dice, err := engine.ParseDiceSum(part.Dice)
		if err != nil {
			return nil, err
		}
		var told *int
		if i < len(fromTheTable) {
			told = fromTheTable[i]
		}
		rolls, total, err := dice.RollOrTake(told, rollDie)
		if err != nil {
			return nil, err
		}
		// O DADO DO CONFUSO diz o que o resultado manda, pela tabela da condição (p394).
		note := part.Note
		if outcome := live.ConditionRollNote(asked.Condition, total); outcome != "" {
			note = outcome
		}
		roll.Parts = append(roll.Parts, live.SpellCastPart{
			Dice: part.Dice, Kind: part.Kind, Note: note, Rolls: rolls, Total: total, ByHand: told != nil,
		})
	}
	return st.RecordRoll(ctx, sessionID, roll)
}
