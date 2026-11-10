package character

import (
	"context"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Blast é o que um encanto dispara em vez do golpe, rolado: a bola de fogo da Flamejante (p335), com a CD de
// quem dispara e os dados. A ação, a vez da rodada e o PM são cobrados na mesa, numa transação só.
type Blast struct {
	engine.EnchantBlast
	DC     int
	Rolls  []int
	Total  int
	ByHand bool
}

// RollsTheBlast acha o disparo do encanto na arma em punho e o rola, ou confere o total que a mesa rolou na
// mão (decisão do dono: o dado da mesa é de primeira classe). Ela recusa a arma sem disparo e o total que os
// dados não tiram, antes de qualquer cobrança.
//
//	plays.RollsTheBlast(ctx, row, 7, nil) // a bola de fogo da espada (item 7), com os 6d6 rolados no app
func (p Plays) RollsTheBlast(ctx context.Context, row sqlcgen.Character, itemID int64, fromTheTable *int) (Blast, error) {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return Blast{}, fmt.Errorf("carregar a ficha de %s: %w", row.Name, err)
	}
	computed, err := sheet.Compute(dto)
	if err != nil {
		return Blast{}, fmt.Errorf("computar a ficha de %s: %w", row.Name, err)
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return Blast{}, err
	}
	for _, found := range dto.Ruleset.EnchantBlasts(ec) {
		if found.ItemID != itemID {
			continue
		}
		blast := Blast{EnchantBlast: found, DC: sheet.BlastSaveDC(dto.Level, computed.Attributes, found.SaveAttributes)}
		if fromTheTable != nil {
			lowest, highest, err := engine.DiceRange(found.Damage)
			if err != nil {
				return Blast{}, err
			}
			if *fromTheTable < lowest || *fromTheTable > highest {
				return Blast{}, fmt.Errorf("%s tira de %d a %d, e a mesa disse %d", found.Damage, lowest, highest, *fromTheTable)
			}
			blast.Total, blast.ByHand = *fromTheTable, true
			return blast, nil
		}
		if blast.Rolls, err = engine.RollEachDieWith(found.Damage, p.rollDie); err != nil {
			return Blast{}, err
		}
		for _, die := range blast.Rolls {
			blast.Total += die
		}
		return blast, nil
	}
	return Blast{}, fmt.Errorf("o item %d de %s não dispara nada em vez do golpe: empunhe a arma do encanto", itemID, row.Name)
}
