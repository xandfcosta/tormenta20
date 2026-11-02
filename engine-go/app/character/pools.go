package character

import (
	"context"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// SetHpTo crava o PV atual num valor exato.
//
// Tem UM chamador de verdade — o gerador do elenco de teste, que precisa de
// ficha machucada para a tela ter o que mostrar. É o único gesto que declara um
// TOTAL em vez de um passo, e por isso não drena PV temporários: drenar é regra
// de pancada, e isto aqui não é uma pancada.
func (p Plays) SetHpTo(ctx context.Context, c sqlcgen.Character, current int64) error {
	_, err := sheet.ApplyToPools(ctx, p.queries, p.catalogs, c,
		func(pools sheet.Pools) (sheet.Pools, error) {
			pools.HpCurrent = current
			return pools, nil
		})
	return err
}
