package character

import (
	"context"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// O QUE COMEÇA COM A CENA.
//
// A Armadura da Honra (p53) é *"no início de cada cena, você recebe uma
// quantidade de pontos de vida temporários igual a seu nível + seu Carisma"*.
// Ela não tem gesto — quem a tem, recebe —, e por isso mora no INÍCIO DA CENA e
// não num botão: oferecer um clique para o que o livro dá de graça seria cobrar
// atrito, e ela não tem decisão nenhuma dentro.

// GrantSceneStartPools entrega as reservas que o início da cena concede e diz
// quantas saíram.
//
// A CONTAGEM é o que torna o best-effort honesto lá em cima: quem percorre o
// grupo precisa saber quantas fichas receberam, senão uma falha some.
//
// @example GrantSceneStartPools(ctx, row) // 1, nil — o cavaleiro com a Armadura da Honra
func (p Plays) GrantSceneStartPools(ctx context.Context, row sqlcgen.Character) (int, error) {
	dto, err := sheet.Load(ctx, p.queries, p.catalogs, row)
	if err != nil {
		return 0, fmt.Errorf("montar a ficha %d para o início da cena: %w", row.ID, err)
	}
	given := 0
	for _, spec := range book.SceneStartGrants() {
		if !sheet.OwnsActivation(dto, spec) {
			continue
		}
		howMuch, ok := p.TempHpAmount(ctx, row, spec.Grant.Attribute)
		if !ok {
			continue
		}
		if _, err := p.ApplyTempHpPool(
			ctx, row.ID, "power", spec.ID, spec.Grant.Scope, howMuch, "PV temporários"); err != nil {
			return given, err
		}
		given++
	}
	return given, nil
}
