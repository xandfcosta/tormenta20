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
	stats, found := engine.WildShapeStatsOf(shape.Form, shape.Tier)
	if !found {
		return fmt.Errorf("a forma %q no nível %q não está no quadro da p63", shape.Form, shape.Tier)
	}
	if powerID := engine.WildTierPowerID(shape.Tier); !sheet.OwnsClassPower(dto, powerID) {
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
	cost := engine.WildTierPM(shape.Tier)
	return p.inTx(ctx, "assumir a forma selvagem", func(q *sqlcgen.Queries) error {
		if err := chargeMp(ctx, q, p.catalogs, row, cost); err != nil {
			return err
		}
		return q.SaveCharacterWildShape(ctx, sqlcgen.SaveCharacterWildShapeParams{
			Characterid: row.ID, Form: string(shape.Form), Tier: string(shape.Tier),
			Damagetype: shape.DamageType, Movement: string(shape.Movement), Pmpaid: int64(cost),
		})
	})
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
