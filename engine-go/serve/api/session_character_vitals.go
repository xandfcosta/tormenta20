package api

import (
	"context"
	"database/sql"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/platform"
)

// O PV do rastreador É o PV da ficha, e ele atravessa uma PORTA.
//
// Este arquivo implementa `live.SheetVitals`. Os métodos moram deste lado porque
// usam `applyDamagePlan` e `live.ClampVital`, que são regras da FICHA — e um
// pacote do regime ao vivo não pode conhecê-las. O regime declara o que precisa;
// quem entrega fica aqui.
//
// A LINHA DO PERSONAGEM é a fonte, e não um blob no estado da sessão. Com duas
// fontes a mesma tela mostra 52/95 na iniciativa e 57/95 no card do grupo, e o
// caminho do socket ignora PV TEMPORÁRIOS: bater 5 num personagem com Armadura
// Arcana cobraria dos PV reais, enquanto o mesmo 5 pela ficha drena o pool
// primeiro — duas regras para a mesma pancada. Aqui o dano percorre a MESMA
// regra do `POST /personagens/{id}/damage` e a entrada da iniciativa espelha o
// que foi gravado. NPC continua vivendo só no rastreador: não há ficha atrás
// dele.

// sheetVitals é quem cumpre a porta. Guarda só o que precisa — as queries —
// em vez de um `*Server` inteiro: uma porta que recebesse o servidor não seria
// porta, seria o acoplamento de antes com outro nome.
type sheetVitals struct{ q *sqlcgen.Queries }

// applyDamagePlan runs the book's damage order — temporary pools first, biggest
// first — persisting the drained pools and the new PV. Returns the plan so the
// HTTP handler can report what was absorbed.
//
// Shared by POST /personagens/{id}/damage and the live tracker: uma pancada
// digitada na sessão e a mesma pancada digitada na ficha não podem discordar.
func applyDamagePlan(
	ctx context.Context, q *sqlcgen.Queries, row sqlcgen.Character, amount int,
) (sheet.DamagePlan, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return sheet.DamagePlan{}, err
	}
	plan := sheet.PlanDamage(sheet.ParseTempHpPools(effects), int(row.Hpcurrent), amount)
	for _, u := range plan.Updates {
		if err := q.UpdateEffectModifiers(ctx, sqlcgen.UpdateEffectModifiersParams{
			Modifiers: u.Modifiers, ID: u.EffectID,
		}); err != nil {
			return sheet.DamagePlan{}, err
		}
	}
	for _, delID := range plan.DeleteIDs {
		if err := q.DeleteEffectByID(ctx, delID); err != nil {
			return sheet.DamagePlan{}, err
		}
	}
	if plan.HpCurrent != int(row.Hpcurrent) {
		if err := q.SetHpCurrent(ctx, sqlcgen.SetHpCurrentParams{
			HpCurrent: int64(plan.HpCurrent), UpdatedAt: platform.NowISO(), ID: row.ID,
		}); err != nil {
			return sheet.DamagePlan{}, err
		}
	}
	return plan, nil
}

// ApplyDelta moves a character's PV/PM by a delta and persists it,
// returning the values the tracker entry must mirror. Damage (negative PV) goes
// through applyDamagePlan; healing and PM are clamped to the character's maxes.
func (v sheetVitals) ApplyDelta(
	ctx context.Context, charID int64, hpDelta, mpDelta *int64,
) (*int64, *int64, error) {
	row, err := v.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	hp, healed := row.Hpcurrent, false
	if hpDelta != nil && *hpDelta < 0 {
		plan, err := applyDamagePlan(ctx, v.q, row, int(-*hpDelta))
		if err != nil {
			return nil, nil, err
		}
		hp = int64(plan.HpCurrent) // já persistido pelo plano
	} else if hpDelta != nil {
		hp, healed = live.ClampVital(row.Hpcurrent+*hpDelta, &row.Hpmax), true
	}
	mp := row.Mpcurrent
	if mpDelta != nil {
		mp = live.ClampVital(row.Mpcurrent+*mpDelta, &row.Mpmax)
	}
	return v.persistVitals(ctx, charID, hp, healed, mp, mpDelta != nil)
}

// ApplyAbsolute sets absolute PV/PM on the character (the tracker's
// "vitals-patch"). An absolute value is a statement about the total, not a hit,
// so it does NOT drain temporary pools — that rule belongs to damage.
func (v sheetVitals) ApplyAbsolute(
	ctx context.Context, charID int64, hpCurrent, mpCurrent *int64,
) (*int64, *int64, error) {
	row, err := v.q.GetCharacter(ctx, charID)
	if err != nil {
		return nil, nil, err
	}
	hp := row.Hpcurrent
	if hpCurrent != nil {
		hp = live.ClampVital(*hpCurrent, &row.Hpmax)
	}
	mp := row.Mpcurrent
	if mpCurrent != nil {
		mp = live.ClampVital(*mpCurrent, &row.Mpmax)
	}
	return v.persistVitals(ctx, charID, hp, hpCurrent != nil, mp, mpCurrent != nil)
}

// persistVitals writes only what changed and hands back BOTH values for the
// entry to mirror — inclusive o que não foi escrito, senão o rastreador voltaria
// a mostrar um número que a ficha não tem. `writeHp` é falso quando o plano de
// dano já gravou o PV.
func (v sheetVitals) persistVitals(
	ctx context.Context, charID, hp int64, writeHp bool, mp int64, writeMp bool,
) (*int64, *int64, error) {
	if writeHp || writeMp {
		params := sqlcgen.UpdateVitalsParams{UpdatedAt: platform.NowISO(), ID: charID}
		if writeHp {
			params.HpCurrent = nullInt(&hp)
		}
		if writeMp {
			params.MpCurrent = nullInt(&mp)
		}
		if _, err := v.q.UpdateVitals(ctx, params); err != nil {
			return nil, nil, err
		}
	}
	return &hp, &mp, nil
}

// Ponteiro nulo vira NULL, e não zero — a diferença entre "não mexeu neste
// vital" e "zerou este vital".
func nullInt(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}
