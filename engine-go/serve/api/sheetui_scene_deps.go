package api

import (
	"context"

	"t20engine/domain/engine"
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

// ActionFitsOnTurn e SpendActionOnTurn levam o gesto da ficha ao turno da mesa
// em que este personagem está.
//
// Achar a mesa é do hospedeiro porque só ele tem o `session.Store`: a ficha
// pergunta por personagem, e a resposta atravessa a porta já decidida.
func (h sheetHost) ActionFitsOnTurn(characterID int64, custo engine.ActionCost) error {
	return h.rules.sessions.CharacterActionFits(characterID, custo)
}

func (h sheetHost) SpendActionOnTurn(characterID int64, custo engine.ActionCost) error {
	return h.rules.sessions.SpendCharacterAction(characterID, custo)
}
