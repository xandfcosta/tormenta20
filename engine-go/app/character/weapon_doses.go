package character

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// LoadDose carrega um item da mochila na Injeção alquímica de uma arma da mesma ficha
// (p165): "carregá-la exige uma ação completa e o gasto dos itens". A ação é da mesa,
// como a de usar um item; o gasto é daqui, na mesma transação que grava a dose.
//
//	err := plays.LoadDose(ctx, 13, 40, 41)
func (p Plays) LoadDose(ctx context.Context, characterID, doseItemID, weaponItemID int64) error {
	return p.inTx(ctx, "carregar a dose", func(q *sqlcgen.Queries) error {
		dose, err := itemOfCharacter(ctx, q, characterID, doseItemID)
		if err != nil {
			return err
		}
		weapon, err := itemOfCharacter(ctx, q, characterID, weaponItemID)
		if err != nil {
			return err
		}
		if !dose.Catalogid.Valid || sheet.DoseEffectOf(dose.Catalogid.String) == nil {
			return fmt.Errorf("%s não carrega na %s", dose.Name, sheet.AlchemicalInjectionName)
		}
		catalog := book.ItemByID(weapon.Catalogid.String)
		if catalog == nil || catalog.Weapon == nil || !sheet.HoldsDoses(weapon.Improvements) {
			return fmt.Errorf("%s não tem a melhoria %s", weapon.Name, sheet.AlchemicalInjectionName)
		}
		held, err := q.CountItemDoses(ctx, weapon.ID)
		if err != nil {
			return fmt.Errorf("contar as doses de %s: %w", weapon.Name, err)
		}
		if held >= sheet.MaxDoses {
			return fmt.Errorf("a %s guarda %d doses, e %s já tem %d",
				sheet.AlchemicalInjectionName, sheet.MaxDoses, weapon.Name, held)
		}
		err = q.AddItemDose(ctx, sqlcgen.AddItemDoseParams{Itemid: weapon.ID, Catalogid: dose.Catalogid.String})
		if err != nil {
			return fmt.Errorf("carregar %s em %s: %w", dose.Name, weapon.Name, err)
		}
		if dose.Quantity > 1 {
			return q.SetItemQuantity(ctx, sqlcgen.SetItemQuantityParams{Quantity: dose.Quantity - 1, ID: dose.ID})
		}
		return q.DeleteItem(ctx, dose.ID)
	})
}

func itemOfCharacter(ctx context.Context, q *sqlcgen.Queries, characterID, itemID int64) (sqlcgen.GetItemRow, error) {
	item, err := q.GetItem(ctx, itemID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && item.Characterid != characterID) {
		return sqlcgen.GetItemRow{}, fmt.Errorf("o item %d não está nesta ficha", itemID)
	}
	if err != nil {
		return sqlcgen.GetItemRow{}, fmt.Errorf("ler o item %d: %w", itemID, err)
	}
	return item, nil
}
