package engine

// A DERIVAÇÃO: o número final de cada pergunta da ficha, a partir do personagem e dos efeitos já resolvidos.
//
// # Por que é uma FUNÇÃO, e não sistemas
//
// Aqui há uma entidade só — a ficha —, e cada pergunta é uma chamada. Em ECS ela era sete sistemas de uma
// linha cada, e a única dependência (a carga alimenta o deslocamento e as perícias) virava uma ordem numa
// lista: um sistema fora de ordem deixava o componente ausente, e quem o lia recebia ZERO sem erro. Em linha
// reta, o deslocamento e as perícias não compilam sem a carga na mão. O critério é o do `engine.md`: ECS
// onde a consulta atravessa entidades, função onde há uma só.

// deriveSheet monta a ficha decomposta. `raceSquares` é o deslocamento da raça, em quadrados.
//
//	deriveSheet(ch, r.EffectsOf(ch, nil), r.raceDisplacement(ch))
func deriveSheet(ch Character, effects ItemEffects, raceSquares int) ComputedSheet {
	load := loadBreakdownOf(ch, inventorySlotsTotal(ch, effects))

	attributes := make(map[string]AttributeBreakdown, len(AttributeKeys))
	for _, a := range AttributeKeys {
		attributes[a] = attributeBreakdown(ch, a, effects)
	}
	expertises := make([]ExpertiseBreakdown, 0, len(ch.Expertises))
	for _, ex := range ch.Expertises {
		expertises = append(expertises, expertiseBreakdown(ch, ex, effects, load))
	}
	bestAttr, bestBaseCd := bestSpellcasting(ch, effects)

	return ComputedSheet{
		Defense:                 defenseBreakdown(ch, effects),
		Displacement:            displacementBreakdown(raceSquares, effects, load),
		FlySpeed:                flySpeedTotal(effects),
		IgnoresDifficultTerrain: effects.Flags[MovementIgnoresDifficult],
		Load:                    load,
		Attributes:              attributes,
		PmLimit:                 pmLimitBreakdown(ch, effects, SpellAt{}),
		BestBaseSpellCd:         bestBaseCd,
		BestSpellAttribute:      bestAttr,
		SpellCdByAttribute:      spellCdByAttribute(ch, effects),
		SpellDCBonus:            spellDCBonus(effects),
		PmCostMod:               pmCostMod(effects),
		AttackAll:               totalContribsFor(effects, ModifierTarget{K: "attack", Scope: "all"}),
		DamageAll:               totalContribsFor(effects, ModifierTarget{K: "damage", Scope: "all"}),
		DamageReduction:         characterDamageReduction(effects),
		CritExtraIgnoreChance:   totalContribsFor(effects, ModifierTarget{K: "critExtraIgnoreChance"}),
		TempHpFury:              tempHpFromPowers(ch, effects, true),
		Expertises:              expertises,
		AutoFailExpertises:      autoFailExpertises(effects),
		char:                    ch,
		effects:                 effects,
	}
}
