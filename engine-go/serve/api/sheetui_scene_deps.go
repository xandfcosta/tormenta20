package api

import (
	"context"

	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// A CENA DA FICHA, com adaptador próprio: o núcleo mais um `sheetRules`, que é
// onde as regras moram.
type sheetHost struct {
	sceneCore
	rules sheetRules
}

func (s *Server) sheetHost() sheetHost {
	return sheetHost{sceneCore: s.sceneCore(), rules: s.sheetRules()}
}

// O adaptador cumprindo a porta da FICHA (`sheetui.Deps`).
//
// O sinal de que a fronteira está no lugar é nenhum destes métodos desenhar
// nada — e nenhum handler da cena tocar banco fora do `Queries`.

// LoadCharacter e ComputeSheet atravessam pelo adaptador, e não pelo núcleo:
// só a ficha e a Mesa as pedem, e o núcleo é o que quase toda cena pede.
func (h sheetHost) LoadCharacter(ctx context.Context, c sqlcgen.Character) (sheet.CharacterDTO, error) {
	return h.rules.LoadCharacter(ctx, c)
}

// CharacterChanged avisa a MESA que esta ficha mexeu.
func (h sheetHost) CharacterChanged(characterID int64) { h.rules.characterChanged(characterID) }

// SaveProficiencies grava as categorias, devolvendo o blob e a lista limpa.
func (h sheetHost) SaveProficiencies(
	ctx context.Context, id int64, categorias []string,
) (string, []string, error) {
	return h.rules.saveProficiencies(ctx, id, categorias)
}

// SaveNewCraft acrescenta a perícia que o livro não tem.
func (h sheetHost) SaveNewCraft(ctx context.Context, id int64, nome string) error {
	return h.rules.saveNewCraft(ctx, id, nome)
}
