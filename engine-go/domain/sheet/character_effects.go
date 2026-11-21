package sheet

import (
	"errors"

	"t20engine/domain/engine"
)

// ErrSheetWithoutRules diz que a ficha chegou sem o mundo dela (`Ruleset`): não há efeito para calcular. Uns
// chamadores leem isso como "nada a ler" e seguem com o vazio; outros, como erro — a decisão é de cada um.
var ErrSheetWithoutRules = errors.New("a ficha chegou sem as regras do mundo dela")

// CharacterEffects são os efeitos desta ficha: as regras dela, o personagem do motor e os interruptores que o
// jogador ligou (`Conditionals`). Devolve também o personagem do motor, que alguns chamadores usam para outra
// coisa (os itens, a Forma Selvagem). Era o passo a passo que onze lugares escreviam à mão.
//
//	effects, ec, err := sheet.CharacterEffects(dto)
func CharacterEffects(dto CharacterDTO) (engine.ItemEffects, engine.Character, error) {
	if dto.Ruleset == nil {
		return engine.ItemEffects{}, engine.Character{}, ErrSheetWithoutRules
	}
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return engine.ItemEffects{}, engine.Character{}, err
	}
	return dto.Ruleset.EffectsOf(ec, ToStringSet(dto.Conditionals)), ec, nil
}
