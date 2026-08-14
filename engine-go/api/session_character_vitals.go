package api

// O PV do rastreador É o PV da ficha (ALE-122).
//
// Antes desta fatia havia dois PV para o mesmo personagem: o socket escrevia num
// blob (`sessions.runtimeState`) e o HTTP escrevia na linha do personagem. A
// mesma tela mostrava 52/95 na iniciativa e 57/95 no card do grupo, e a ficha do
// jogador — ao lado do rastreador dele — continuava no valor antigo. O espelho
// existia, mas atrás de uma variável de ambiente que não estava em `.env`
// nenhum, nem em produção.
//
// E o caminho do socket ignorava PV TEMPORÁRIOS: bater 5 num personagem com
// Armadura Arcana cobrava dos PV reais, enquanto o mesmo 5 pela ficha drenava o
// pool primeiro. Duas regras para a mesma pancada.
//
// Agora a linha do personagem é a fonte: o dano percorre a MESMA regra do
// `POST /characters/{id}/damage` e a entrada da iniciativa espelha o que foi
// gravado. NPC continua vivendo só no rastreador — não há ficha atrás dele.

import (
	"context"

	"t20engine/db/sqlcgen"
)

// applyDamagePlan runs the book's damage order — temporary pools first, biggest
// first — persisting the drained pools and the new PV. Returns the plan so the
// HTTP handler can report what was absorbed.
//
// Shared by POST /characters/{id}/damage and the live tracker: uma pancada
// digitada na sessão e a mesma pancada digitada na ficha não podem discordar.
func applyDamagePlan(
	ctx context.Context, q *sqlcgen.Queries, row sqlcgen.Character, amount int,
) (damagePlan, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return damagePlan{}, err
	}
	plan := planDamage(parseTempHpPools(effects), int(row.Hpcurrent), amount)
	for _, u := range plan.updates {
		if err := q.UpdateEffectModifiers(ctx, sqlcgen.UpdateEffectModifiersParams{
			Modifiers: u.modifiers, ID: u.effectID,
		}); err != nil {
			return damagePlan{}, err
		}
	}
	for _, delID := range plan.deleteIDs {
		if err := q.DeleteEffectByID(ctx, delID); err != nil {
			return damagePlan{}, err
		}
	}
	if plan.hpCurrent != int(row.Hpcurrent) {
		if err := q.SetHpCurrent(ctx, sqlcgen.SetHpCurrentParams{
			HpCurrent: int64(plan.hpCurrent), UpdatedAt: nowISO(), ID: row.ID,
		}); err != nil {
			return damagePlan{}, err
		}
	}
	return plan, nil
}

// applyCharacterDelta moves a character's PV/PM by a delta and persists it,
// returning the values the tracker entry must mirror. Damage (negative PV) goes
// through applyDamagePlan; healing and PM are clamped to the character's maxes.
func (st *sessionStore) applyCharacterDelta(
	ctx context.Context, charID int64, hpDelta, mpDelta *int64,
) (*int64, *int64, error) {
	row, err := st.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	hp, healed := row.Hpcurrent, false
	if hpDelta != nil && *hpDelta < 0 {
		plan, err := applyDamagePlan(ctx, st.q, row, int(-*hpDelta))
		if err != nil {
			return nil, nil, err
		}
		hp = int64(plan.hpCurrent) // já persistido pelo plano
	} else if hpDelta != nil {
		hp, healed = clampVital(row.Hpcurrent+*hpDelta, &row.Hpmax), true
	}
	mp := row.Mpcurrent
	if mpDelta != nil {
		mp = clampVital(row.Mpcurrent+*mpDelta, &row.Mpmax)
	}
	return st.persistVitals(ctx, charID, hp, healed, mp, mpDelta != nil)
}

// applyCharacterVitals sets absolute PV/PM on the character (the tracker's
// "vitals-patch"). An absolute value is a statement about the total, not a hit,
// so it does NOT drain temporary pools — that rule belongs to damage.
func (st *sessionStore) applyCharacterVitals(
	ctx context.Context, charID int64, hpCurrent, mpCurrent *int64,
) (*int64, *int64, error) {
	row, err := st.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	hp := row.Hpcurrent
	if hpCurrent != nil {
		hp = clampVital(*hpCurrent, &row.Hpmax)
	}
	mp := row.Mpcurrent
	if mpCurrent != nil {
		mp = clampVital(*mpCurrent, &row.Mpmax)
	}
	return st.persistVitals(ctx, charID, hp, hpCurrent != nil, mp, mpCurrent != nil)
}

// persistVitals writes only what changed and hands back BOTH values for the
// entry to mirror — inclusive o que não foi escrito, senão o rastreador voltaria
// a mostrar um número que a ficha não tem. `writeHp` é falso quando o plano de
// dano já gravou o PV.
func (st *sessionStore) persistVitals(
	ctx context.Context, charID, hp int64, writeHp bool, mp int64, writeMp bool,
) (*int64, *int64, error) {
	if writeHp || writeMp {
		params := sqlcgen.UpdateVitalsParams{UpdatedAt: nowISO(), ID: charID}
		if writeHp {
			params.HpCurrent = nullInt(&hp)
		}
		if writeMp {
			params.MpCurrent = nullInt(&mp)
		}
		if _, err := st.q.UpdateVitals(ctx, params); err != nil {
			return nil, nil, err
		}
	}
	return &hp, &mp, nil
}

// characterIDOf reports which character backs an entry, or nil for an NPC —
// which is what decides whether the sheet or the tracker is the record.
func (st *sessionStore) characterIDOf(sessionID int64, entryID string) *int64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	state := st.states[sessionID]
	if state == nil {
		return nil
	}
	idx := findEntryIndex(state, entryID)
	if idx < 0 {
		return nil
	}
	return state.Initiative[idx].CharacterID
}
