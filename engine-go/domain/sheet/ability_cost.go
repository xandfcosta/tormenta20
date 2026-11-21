package sheet

import "t20engine/domain/engine"

// AbilityCostRules são as regras do custo de habilidade desta ficha: o aumento do Alquebrado (p394), que viaja
// nas condições, e a metade do Artista Completo (p45), que liga com a Inspiração. Lidas do mesmo jeito que as
// flags da ficha: sem regras ou sem personagem montável, não há o que ler, e o custo é o do catálogo.
//
//	cost := sheet.AbilityCostRules(dto).Price(book.ScaledCost(spec, steps), book.ActivationClass(spec))
func AbilityCostRules(dto CharacterDTO) engine.AbilityCostRules {
	if dto.Ruleset == nil {
		return engine.AbilityCostRules{}
	}
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return engine.AbilityCostRules{}
	}
	return dto.Ruleset.EffectsOf(ec, ToStringSet(dto.Conditionals)).AbilityCostRules()
}
