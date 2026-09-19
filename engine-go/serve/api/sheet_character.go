package api

import (
	"context"

	"t20engine/domain/engine"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// Os três invólucros abaixo existem para os chamadores de dentro do `api` não
// mudarem junto com a extração (ALE-278): a lógica mora no `sheet`, e aqui só
// se passa o que o `Server` tem na mão. Eles somem quando cada cena receber as
// dependências dela por construtor.

func (sr sheetRules) LoadCharacter(ctx context.Context, c sqlcgen.Character) (sheet.CharacterDTO, error) {
	return sheet.Load(ctx, sr.queries, sr.catalogs, c)
}

func (sr sheetRules) ComputeSheet(ctx context.Context, row sqlcgen.Character) (engine.ComputedSheet, error) {
	return sheet.LoadAndCompute(ctx, sr.queries, sr.catalogs, row)
}
