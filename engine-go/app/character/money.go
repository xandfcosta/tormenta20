package character

import (
	"context"
	"fmt"

	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
)

// ChangeMoney recebe, gasta ou corrige o dinheiro da ficha.
//
// Ela recebe a LINHA e não o saldo: a conta parte do que está gravado, e quem
// chama passando um número seu decidiria de onde a soma começa. O `row` já foi
// lido pelo funil que confere a posse.
//
// A conta e as recusas são do `domain/sheet` (`BalanceAfterMoneyGesture`), que
// é onde o teto da ficha já morava. Aqui fica o que é desta camada: pedir a
// decisão e gravá-la.
func (p Plays) ChangeMoney(
	ctx context.Context, row sqlcgen.Character, modo string, valor float64,
) error {
	saldo, recusa := sheet.BalanceAfterMoneyGesture(row.Tibar, modo, valor)
	if recusa != "" {
		return fmt.Errorf("%s", recusa)
	}
	if err := p.queries.SetCharacterTibar(ctx, sqlcgen.SetCharacterTibarParams{
		Tibar: saldo, UpdatedAt: dbvalue.NowISO(), ID: row.ID,
	}); err != nil {
		return fmt.Errorf("gravar o dinheiro da ficha %d: %w", row.ID, err)
	}
	return nil
}
