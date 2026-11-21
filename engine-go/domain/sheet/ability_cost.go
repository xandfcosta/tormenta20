package sheet

// AbilityCostIncrease é o que sobe o custo de toda habilidade desta ficha: o Alquebrado (p394), que viaja nas
// condições da ficha. É o `increase` da conta única (`engine.AbilityPmCost`), lido do mesmo jeito que as flags
// da ficha: sem regras ou sem personagem montável, não há condição para ler, e o aumento é zero.
//
//	cost := engine.AbilityPmCost(book.ScaledCost(spec, steps), sheet.AbilityCostIncrease(dto))
func AbilityCostIncrease(dto CharacterDTO) int {
	if dto.Ruleset == nil {
		return 0
	}
	ec, err := EngineCharacterFrom(dto)
	if err != nil {
		return 0
	}
	return dto.Ruleset.EffectsOf(ec, ToStringSet(dto.Conditionals)).AbilityCostIncrease()
}
