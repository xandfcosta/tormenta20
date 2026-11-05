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

// O RECIBO DE UM GASTO DE PM: de onde cada ponto saiu, para o gasto desfeito voltar igual.
//
// Por que não devolver a soma ao poço: a poça temporária paga primeiro (p106), e o poço já
// podia estar cheio. A parte da poça se perderia no teto, ou viraria PM que não expira
// (decisão do dono: o personagem não perde PM num golpe que não aconteceu).

// ManaReceipt é um gasto de PM, separado entre o poço e cada poça.
type ManaReceipt struct {
	FromPool int               `json:"fromPool,omitempty"`
	Temp     []DrainedManaPool `json:"temp,omitempty"`
}

// DrainedManaPool é o que saiu de uma poça. Modifiers é a linha como ela estava, porque a
// poça que esvazia é apagada, e ela volta com os campos dela.
type DrainedManaPool struct {
	Source    string `json:"source"`
	CatalogID string `json:"catalogId"`
	Scope     string `json:"scope"`
	Amount    int    `json:"amount"`
	Modifiers string `json:"modifiers"`
}

// receiptOf separa o gasto: `fromPool` é o que o poço pagou, e o resto sai do plano.
func receiptOf(effects []sqlcgen.ListActiveEffectsByCharacterRow, plan SpendPlan, fromPool int) ManaReceipt {
	receipt := ManaReceipt{FromPool: fromPool}
	for _, drained := range plan.Drained {
		at := slices.IndexFunc(effects, func(e sqlcgen.ListActiveEffectsByCharacterRow) bool { return e.ID == drained.EffectID })
		if at < 0 {
			continue
		}
		row := effects[at]
		before, _, _, _ := readTempPool(TempMpTarget, row.Modifiers)
		receipt.Temp = append(receipt.Temp, DrainedManaPool{
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
	ctx context.Context, q *sqlcgen.Queries, cat *engine.Catalogs, row sqlcgen.Character, receipt ManaReceipt,
) (Pools, error) {
	effects, err := q.ListActiveEffectsByCharacter(ctx, row.ID)
	if err != nil {
		return Pools{}, fmt.Errorf("ler as poças de PM da ficha %d: %w", row.ID, err)
	}
	for _, drained := range receipt.Temp {
		if err := restoreManaPool(ctx, q, row.ID, effects, drained); err != nil {
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

// restoreManaPool soma de volta à poça que ainda existe, e recria a que foi apagada. A
// linha que perdeu o modificador recebe o dele de volta, sem perder os outros.
func restoreManaPool(
	ctx context.Context, q *sqlcgen.Queries, characterID int64,
	effects []sqlcgen.ListActiveEffectsByCharacterRow, drained DrainedManaPool,
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
		if IsTempModifier(TempMpTarget, m) {
			current, found = ToInt(m["amount"]), true
		}
	}
	if !found {
		var original []map[string]any
		if err := json.Unmarshal([]byte(drained.Modifiers), &original); err != nil {
			return fmt.Errorf("ler a poça guardada de %s: %w", drained.CatalogID, err)
		}
		for _, m := range original {
			if at < 0 || IsTempModifier(TempMpTarget, m) {
				mods = append(mods, m)
			}
		}
	}
	modifiers := withTempAmount(TempMpTarget, mods, current+drained.Amount)
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
