package character

import (
	"context"
	"encoding/json"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// ManaTheftPowerID é o Roubo de Mana do ladino (p74).
const ManaTheftPowerID = "class.ladino.roubo-de-mana"

// HasManaTheft diz se a ficha tem o Roubo de Mana, pela lista de poderes dela.
func (p Plays) HasManaTheft(ctx context.Context, characterID int64) (bool, error) {
	row, err := p.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return false, fmt.Errorf("ler o personagem %d: %w", characterID, err)
	}
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return false, fmt.Errorf("montar a ficha de %s: %w", row.Name, err)
	}
	return sheet.OwnsPower(dto, ManaTheftPowerID), nil
}

// StealMana é o Roubo de Mana (p74): o ladino recebe 1 PM temporário por dado do furtivo, e a
// criatura atrás do alvo perde 1 PM por dado, se tiver. `victim` é zero quando o alvo não é uma
// ficha (o NPC da fila não tem PM rastreado).
//
// OS ROUBOS SOMAM (decisão do dono): a reserva do Roubo de Mana é uma só na cena, e cada roubo
// soma o dele ao que já está lá — contra criaturas diferentes, porque contra a mesma a cena já
// recusou (`live.MarkManaStolen`).
//
//	p.StealMana(ctx, 48, 2, 4) // a Ladra de Mana rouba 4 PM da Guerreira
func (p Plays) StealMana(ctx context.Context, thief, victim int64, dice int) error {
	if dice <= 0 {
		return nil
	}
	already, err := p.manaTheftPool(ctx, thief)
	if err != nil {
		return err
	}
	if _, err := p.ApplyTempPool(ctx, thief, "power", ManaTheftPowerID,
		book.PowerGrant{Kind: "temp-mp", Scope: "scene"}, already+dice); err != nil {
		return err
	}
	if victim == 0 {
		return nil
	}
	row, err := p.queries.GetCharacter(ctx, victim)
	if err != nil {
		return fmt.Errorf("ler o alvo %d do Roubo de Mana: %w", victim, err)
	}
	if _, err := sheet.ApplyToPools(ctx, p.queries, p.catalogs, row, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = max(0, pools.MpCurrent-int64(dice))
		return pools, nil
	}); err != nil {
		return fmt.Errorf("tirar %d PM de %s: %w", dice, row.Name, err)
	}
	return nil
}

// manaTheftPool é quanto a reserva do Roubo de Mana do ladino tem agora, zero sem ela.
func (p Plays) manaTheftPool(ctx context.Context, thief int64) (int, error) {
	effects, err := p.queries.ListActiveEffectsByCharacter(ctx, thief)
	if err != nil {
		return 0, fmt.Errorf("ler as reservas do personagem %d: %w", thief, err)
	}
	for _, e := range effects {
		if e.Catalogid != ManaTheftPowerID || e.Scope != "scene" {
			continue
		}
		var mods []map[string]any
		if json.Unmarshal([]byte(e.Modifiers), &mods) != nil {
			continue
		}
		for _, m := range mods {
			if sheet.IsTempModifier(sheet.TempMpTarget, m) {
				return sheet.ToInt(m["amount"]), nil
			}
		}
	}
	return 0, nil
}
