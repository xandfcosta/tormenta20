package transcript

// Parte 13 de 13 de `ClassPowers` (class_powers.go).
var classPowersPart13 = []ClassPower{
	{
		ID:          "poder.valentao",
		BookPage:    66,
		Description: "+2 em testes de ataque e rolagens de dano contra oponentes caídos, desprevenidos, flanqueados ou indefesos.",
		Name:        "Valentão",
		UID:         "pwr_hpf2masdfi",
		Modeled:     []string{"o +2 no ataque e no dano, que o jogador liga"},
		Modifiers: []Modifier{
			{
				Target:    ModifierTarget{Kind: "attack", Scope: "all"},
				Amount:    2,
				BonusType: "untyped",
				Condition: &ModifierCondition{
					Kind:  "flagOn",
					Flag:  "valentao",
					Label: "contra caído, desprevenido, flanqueado ou indefeso",
				},
			},
			{
				Target:    ModifierTarget{Kind: "damage", Scope: "all"},
				Amount:    2,
				BonusType: "untyped",
				Condition: &ModifierCondition{
					Kind:  "flagOn",
					Flag:  "valentao",
					Label: "contra caído, desprevenido, flanqueado ou indefeso",
				},
			},
		},
	},
}
