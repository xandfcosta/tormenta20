package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// A FORMA SELVAGEM (p62-63): transformar paga o PM do nível e grava a forma; voltar ao normal apaga a
// forma e não devolve nada. As duas coisas de transformar — o PM e a forma — vão numa transação, pela
// razão das posturas: PM cobrado sem forma, ou forma sem PM, seria a metade de um gesto.

// TakeWildShape transforma o druida na forma escolhida. Quem já está numa forma troca de forma, pagando
// de novo: é uma transformação nova.
func (p Plays) TakeWildShape(ctx context.Context, row sqlcgen.Character, dto sheet.CharacterDTO, shape engine.WildShape) error {
	if _, found := engine.WildShapeStatsOf(shape.Form, shape.Tier); !found {
		return fmt.Errorf("a forma %q no nível %q não está no quadro da p63", shape.Form, shape.Tier)
	}
	if err := primalFormFits(dto, shape); err != nil {
		return err
	}
	stats, _ := engine.WildShapeStatsOfShape(shape)
	if powerID := engine.WildTierPowerID(shape.Tier); !sheet.OwnsPower(dto, powerID) {
		return fmt.Errorf("a forma %s pede o poder %s, e o personagem não o tem (p62)", shape.Tier, powerName(powerID))
	}
	if !slices.Contains(engine.WildShapeDamageTypes, shape.DamageType) {
		return fmt.Errorf("o dano da arma da forma é corte, impacto ou perfuração (p63), e veio %q", shape.DamageType)
	}
	if _, chosen := stats.Moves[shape.Movement]; stats.Moves != nil && !chosen {
		return fmt.Errorf("a %s pede o deslocamento escolhido (p63), e veio %q", engine.WildFormName(shape.Form), shape.Movement)
	}
	if stats.Moves == nil {
		shape.Movement = ""
	}
	cost := engine.AbilityPmCost(engine.WildTierPM(shape.Tier), sheet.AbilityCostIncrease(dto))
	return p.inTx(ctx, "assumir a forma selvagem", func(q *sqlcgen.Queries) error {
		if err := chargeMp(ctx, q, p.catalogs, row, cost); err != nil {
			return err
		}
		if err := q.SaveCharacterWildShape(ctx, sqlcgen.SaveCharacterWildShapeParams{
			Characterid: row.ID, Form: string(shape.Form), Tier: string(shape.Tier),
			Damagetype: shape.DamageType, Movement: string(shape.Movement), Pmpaid: int64(cost),
		}); err != nil {
			return err
		}
		// A SEGUNDA FORMA é linha à parte, e a transformação nova a regrava ou a apaga: trocar de fera
		// primal para uma forma só não pode deixar a segunda de antes valendo.
		if shape.Second == "" {
			return q.ClearCharacterWildShapePrimal(ctx, row.ID)
		}
		return q.SaveCharacterWildShapePrimal(ctx, sqlcgen.SaveCharacterWildShapePrimalParams{
			Characterid: row.ID, Secondform: string(shape.Second),
			Weaponfrom: string(shape.WeaponFrom), Sizefrom: string(shape.SizeFrom),
		})
	})
}

// primalFormFits confere a segunda forma da Forma Primal (p62): ela pede o poder, é OUTRO animal ("dois
// tipos"), e a arma e o tamanho vêm de uma das duas.
func primalFormFits(dto sheet.CharacterDTO, shape engine.WildShape) error {
	if shape.Second == "" {
		return nil
	}
	if !sheet.OwnsPower(dto, engine.PrimalFormPowerID) {
		return fmt.Errorf("a segunda forma pede o poder Forma Primal (p62), e o personagem não o tem")
	}
	if shape.Second == shape.Form {
		return fmt.Errorf("a Forma Primal são dois tipos de animais (p62), e veio %s duas vezes", engine.WildFormName(shape.Form))
	}
	if _, found := engine.WildShapeStatsOf(shape.Second, shape.Tier); !found {
		return fmt.Errorf("a segunda forma %q no nível %q não está no quadro da p63", shape.Second, shape.Tier)
	}
	for _, from := range []engine.WildForm{shape.WeaponFrom, shape.SizeFrom} {
		if from != "" && from != shape.Form && from != shape.Second {
			return fmt.Errorf("a arma e o tamanho vêm de uma das duas formas (%s ou %s), e veio %q",
				engine.WildFormName(shape.Form), engine.WildFormName(shape.Second), from)
		}
	}
	return nil
}

// LeaveWildShape volta o druida ao normal: "cada transformação dura pelo tempo que você quiser" (p63).
func (p Plays) LeaveWildShape(ctx context.Context, row sqlcgen.Character) error {
	if err := p.queries.ClearCharacterWildShape(ctx, row.ID); err != nil {
		return fmt.Errorf("voltar %d ao normal: %w", row.ID, err)
	}
	return nil
}

// powerName é o nome do poder para a recusa, ou o id quando o catálogo não o conhece.
func powerName(id string) string {
	if power, found := book.ClassPowers()[id]; found {
		return power.Name
	}
	return id
}
