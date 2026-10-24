package book

import (
	"sync"

	"t20engine/domain/catalog"
	"t20engine/domain/engine"
)

// EngineCatalogs é o livro na forma que o motor consulta, primado uma vez a
// partir dos arquivos embutidos.
//
// Mora no `book` porque é o pacote que já conhece os dois lados: o `catalog`
// guarda o dado e o `engine` o indexa, e nenhum dos dois importa o outro. O
// valor é só de leitura depois de primado, e por isso servidor, semente,
// oráculo e testes podem dividir o mesmo.
//
//	catalogs, err := book.EngineCatalogs()
//	cards := engine.BookRuleset(catalogs).ComputeWeaponCards(char)
func EngineCatalogs() (*engine.Catalogs, error) {
	engineCatalogsOnce.Do(func() {
		engineCatalogsValue, engineCatalogsErr = engine.PrimeEngineCatalogs(catalog.EnginePayload())
	})
	return engineCatalogsValue, engineCatalogsErr
}

var (
	engineCatalogsOnce  sync.Once
	engineCatalogsValue *engine.Catalogs
	engineCatalogsErr   error
)
