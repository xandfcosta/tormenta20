package transcript

// Parte 2 de 2 de `GrantedPowers` (granted_powers.go).
var grantedPowersPart02 = []GrantedPower{
	{
		ID:       "talento-artistico",
		Name:     "Talento Artístico",
		Gods:     []string{"Marah"},
		Effect:   "Você recebe +2 em Acrobacia, Atuação e Diplomacia.",
		Kind:     "social",
		BookPage: 135,
		Modifiers: []Modifier{
			{Target: ModifierTarget{Kind: "expertise", Name: "Acrobacia"}, Amount: 2, BonusType: "untyped"},
			{Target: ModifierTarget{Kind: "expertise", Name: "Atuação"}, Amount: 2, BonusType: "untyped"},
			{Target: ModifierTarget{Kind: "expertise", Name: "Diplomacia"}, Amount: 2, BonusType: "untyped"},
		},
		UID: "pwr_mxuz65xum4",
	},
	{
		ID:       "armas-da-ambicao",
		Name:     "Armas da Ambição",
		Gods:     []string{"Valkaria"},
		Effect:   "Você recebe +1 em testes de ataque e na margem de ameaça com armas nas quais é proficiente.",
		Kind:     "utilidade",
		BookPage: 132,
		Modifiers: []Modifier{
			{
				Target:    ModifierTarget{Kind: "attack", Scope: "all"},
				Amount:    1,
				BonusType: "untyped",
				Condition: &ModifierCondition{
					Kind:  "flagOn",
					Flag:  "armas-da-ambicao",
					Label: "com armas nas quais é proficiente",
				},
			},
			{
				Target:    ModifierTarget{Kind: "critRange"},
				Amount:    1,
				BonusType: "untyped",
				Condition: &ModifierCondition{
					Kind:  "flagOn",
					Flag:  "armas-da-ambicao",
					Label: "com armas nas quais é proficiente",
				},
			},
		},
		UID: "pwr_xfd2x2ta5a",
	},
	{
		ID:       "arsenal-das-profundezas",
		Name:     "Arsenal das Profundezas",
		Gods:     []string{"Oceano"},
		Effect:   "Você recebe +2 nas rolagens de dano com azagaias, lanças e tridentes e seu multiplicador de crítico com essas armas aumenta em +1.",
		Kind:     "utilidade",
		BookPage: 132,
		Modifiers: []Modifier{
			{
				Target:    ModifierTarget{Kind: "damage", Scope: "all"},
				Amount:    2,
				BonusType: "untyped",
				Condition: &ModifierCondition{
					Kind:  "flagOn",
					Flag:  "arsenal-das-profundezas",
					Label: "com azagaias, lanças e tridentes",
				},
			},
			{
				Target:    ModifierTarget{Kind: "critMult"},
				Amount:    1,
				BonusType: "untyped",
				Condition: &ModifierCondition{
					Kind:  "flagOn",
					Flag:  "arsenal-das-profundezas",
					Label: "com azagaias, lanças e tridentes",
				},
			},
		},
		UID: "pwr_aivhbbbd5a",
	},
	{
		ID:       "sangue-ofidico",
		Name:     "Sangue Ofídico",
		Gods:     []string{"Sszzaas"},
		Effect:   "Você recebe resistência a veneno +5 e a CD para resistir aos seus venenos aumenta em +2.",
		Kind:     "utilidade",
		BookPage: 135,
		Modifiers: []Modifier{
			{
				Target:    ModifierTarget{Kind: "resistance"},
				Amount:    5,
				BonusType: "untyped",
				Condition: &ModifierCondition{Kind: "against", Trait: "veneno"},
			},
		},
		UID: "pwr_rrvxgfr3tt",
	},
}
