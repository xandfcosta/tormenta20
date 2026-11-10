package sheet

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"t20engine/domain/engine"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// O RECIBO DE UMA PERDA DE PM OU DE PV: de onde cada ponto saiu, para a perda desfeita voltar igual.
//
// Por que não devolver a soma ao poço: a poça temporária paga primeiro (p106), e o poço já
// podia estar cheio. A parte da poça se perderia no teto, ou viraria ponto que não expira
// (decisão do dono: o personagem não perde PM num golpe que não aconteceu, nem o PV do gasto
// da Tumular).

// PoolReceipt é uma perda de um vital, separada entre o poço e cada poça temporária.
type PoolReceipt struct {
	FromPool int           `json:"fromPool,omitempty"`
	Temp     []DrainedPool `json:"temp,omitempty"`
}

// DrainedPool é o que saiu de uma poça. Modifiers é a linha como ela estava, porque a
// poça que esvazia é apagada, e ela volta com os campos dela.
type DrainedPool struct {
	Source    string `json:"source"`
	CatalogID string `json:"catalogId"`
	Scope     string `json:"scope"`
	Amount    int    `json:"amount"`
	Modifiers string `json:"modifiers"`
}

// receiptOf separa a perda do vital `target`: `fromPool` é o que o poço pagou, e o resto sai das
// poças que o plano drenou.
func receiptOf(target string, effects []sqlcgen.ListActiveEffectsByCharacterRow, drains []DamageDrain, fromPool int) PoolReceipt {
	receipt := PoolReceipt{FromPool: fromPool}
	for _, drained := range drains {
		at := slices.IndexFunc(effects, func(e sqlcgen.ListActiveEffectsByCharacterRow) bool { return e.ID == drained.EffectID })
		if at < 0 {
			continue
		}
		row := effects[at]
		before, _, _, _ := readTempPool(target, row.Modifiers)
		receipt.Temp = append(receipt.Temp, DrainedPool{
			Source: row.Source, CatalogID: row.Catalogid, Scope: row.Scope,
			Amount: before - drained.NewAmount, Modifiers: row.Modifiers,
		})
	}
	return receipt
}

// GiveManaBack desfaz o gasto do recibo: o poço recebe o que pagou, e cada poça o dela.
//
//	pools, err := sheet.GiveManaBack(ctx, q, catalogs, row, receipt)
func GiveManaBack(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character, receipt PoolReceipt,
) (Pools, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return Pools{}, fmt.Errorf("ler as poças de PM da ficha %d: %w", row.ID, err)
	}
	for _, drained := range receipt.Temp {
		if err := restoreTempPool(ctx, q, TempMpTarget, row.ID, effects, drained); err != nil {
			return Pools{}, err
		}
	}
	pools, err := ApplyToPools(ctx, q, cat, row, func(pools Pools) (Pools, error) {
		pools.MpCurrent += int64(receipt.FromPool)
		return pools, nil
	})
	if err != nil {
		return pools, fmt.Errorf("devolver %d PM à ficha %d: %w", receipt.FromPool, row.ID, err)
	}
	return pools, nil
}

// restoreTempPool soma de volta à poça do vital `target` que ainda existe, e recria a que foi
// apagada. A linha que perdeu o modificador recebe o dele de volta, sem perder os outros.
func restoreTempPool(
	ctx context.Context, q *sqlcgen.Queries, target string, characterID int64,
	effects []sqlcgen.ListActiveEffectsByCharacterRow, drained DrainedPool,
) error {
	at := slices.IndexFunc(effects, func(e sqlcgen.ListActiveEffectsByCharacterRow) bool {
		return e.Catalogid == drained.CatalogID && e.Scope == drained.Scope
	})
	var mods []map[string]any
	if at >= 0 {
		if err := json.Unmarshal([]byte(effects[at].Modifiers), &mods); err != nil {
			return fmt.Errorf("ler a poça de %s: %w", drained.CatalogID, err)
		}
	}
	current, found := 0, false
	for _, m := range mods {
		if IsTempModifier(target, m) {
			current, found = ToInt(m["amount"]), true
		}
	}
	if !found {
		var original []map[string]any
		if err := json.Unmarshal([]byte(drained.Modifiers), &original); err != nil {
			return fmt.Errorf("ler a poça guardada de %s: %w", drained.CatalogID, err)
		}
		for _, m := range original {
			if at < 0 || IsTempModifier(target, m) {
				mods = append(mods, m)
			}
		}
	}
	modifiers := withTempAmount(target, mods, current+drained.Amount)
	if at >= 0 {
		if err := q.UpdateEffectModifiers(ctx, sqlcgen.UpdateEffectModifiersParams{
			Modifiers: modifiers, ID: effects[at].ID,
		}); err != nil {
			return fmt.Errorf("devolver a poça de %s: %w", drained.CatalogID, err)
		}
		return nil
	}
	if _, err := q.UpsertActiveEffect(ctx, sqlcgen.UpsertActiveEffectParams{
		Characterid: characterID, Source: drained.Source, Catalogid: drained.CatalogID, Scope: drained.Scope,
		Modifiers: modifiers, Createdat: dbvalue.NowISO(),
	}); err != nil {
		return fmt.Errorf("recriar a poça de %s: %w", drained.CatalogID, err)
	}
	return nil
}

// LoseHitPointsKeepingReceipt tira `amount` de PV pela ordem do livro — os temporários primeiro, o maior
// antes (p106) — e devolve o recibo de onde cada ponto saiu, para quem pode ter de desfazer a perda (o PV
// do gasto da Tumular, que o cancelamento do golpe devolve). Quem chega a 0 cai pelo funil (p236).
//
//	pools, receipt, err := sheet.LoseHitPointsKeepingReceipt(ctx, q, catalogs, row, 5)
func LoseHitPointsKeepingReceipt(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character, amount int,
) (Pools, PoolReceipt, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return Pools{}, PoolReceipt{}, fmt.Errorf("ler as poças de PV da ficha %d: %w", row.ID, err)
	}
	var plan DamagePlan
	var receipt PoolReceipt
	pools, err := ApplyToPools(ctx, q, cat, row, func(pools Pools) (Pools, error) {
		if amount <= 0 {
			return pools, nil
		}
		plan = PlanDamage(ParseTempHpPools(effects), int(pools.HpCurrent), amount)
		receipt = receiptOf(TempHpTarget, effects, plan.Drained, int(pools.HpCurrent)-plan.HpCurrent)
		pools.HpCurrent = int64(plan.HpCurrent)
		return pools, nil
	})
	if err != nil {
		return pools, receipt, fmt.Errorf("tirar %d PV da ficha %d: %w", amount, row.ID, err)
	}
	return pools, receipt, WriteDrain(ctx, q, plan.Updates, plan.DeleteIDs)
}

// GiveHitPointsBack desfaz a perda do recibo: o PV recebe o que pagou, e cada poça temporária o dela.
func GiveHitPointsBack(
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character, receipt PoolReceipt,
) (Pools, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return Pools{}, fmt.Errorf("ler as poças de PV da ficha %d: %w", row.ID, err)
	}
	for _, drained := range receipt.Temp {
		if err := restoreTempPool(ctx, q, TempHpTarget, row.ID, effects, drained); err != nil {
			return Pools{}, err
		}
	}
	pools, err := ApplyToPools(ctx, q, cat, row, func(pools Pools) (Pools, error) {
		pools.HpCurrent += int64(receipt.FromPool)
		return pools, nil
	})
	if err != nil {
		return pools, fmt.Errorf("devolver %d PV à ficha %d: %w", receipt.FromPool, row.ID, err)
	}
	return pools, nil
}
