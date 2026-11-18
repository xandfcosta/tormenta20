package character

import (
	"context"
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// RecoverOnTriggers devolve ao pool o PM que os gatilhos da mesa dão a esta ficha: o
// Panache (p48) recupera 1 PM no acerto crítico ou ao reduzir um inimigo a 0 PV. Quem
// chama é a confirmação do mestre, com os mesmos gatilhos do bônus cumulativo.
//
// RECUPERAR NÃO PASSA DO MÁXIMO: é PM que volta, e não PM temporário.
//
//	p.RecoverOnTriggers(ctx, characterID, "criticalOrDrop")
func (p Plays) RecoverOnTriggers(ctx context.Context, characterID int64, triggers ...string) error {
	var owned []book.Activation
	row, err := p.queries.GetCharacter(ctx, characterID)
	if err != nil {
		return fmt.Errorf("carregar o personagem %d para recuperar PM: %w", characterID, err)
	}
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return fmt.Errorf("montar a ficha %d para recuperar PM: %w", characterID, err)
	}
	for _, spec := range book.Activations() {
		if spec.Recovers != nil && slices.Contains(triggers, spec.Recovers.On) && sheet.OwnsActivation(dto, spec) {
			owned = append(owned, spec)
		}
	}
	if len(owned) == 0 {
		return nil
	}
	gain := 0
	for _, spec := range owned {
		gain += spec.Recovers.Amount
	}
	if _, err := sheet.ApplyToPools(ctx, p.queries, p.catalogs, row, func(pools sheet.Pools) (sheet.Pools, error) {
		pools.MpCurrent = min(pools.MpMax, pools.MpCurrent+int64(gain))
		return pools, nil
	}); err != nil {
		return fmt.Errorf("devolver %d PM a %s: %w", gain, row.Name, err)
	}
	return nil
}
