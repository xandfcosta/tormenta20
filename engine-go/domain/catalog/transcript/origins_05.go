package transcript

// Parte 5 de 5 de `Origins` (origins.go).
var originsPart05 = []Origin{
	{
		ID:   "trabalhador",
		Name: "Trabalhador",
		Benefits: []OriginBenefit{
			{
				ID:          "origin-trabalhador-pericia-atletismo",
				Name:        "Atletismo",
				Kind:        "pericia",
				Description: "Você se torna treinado em Atletismo.",
				Expertise:   "Atletismo",
				UID:         "org_imgpbuc3av",
			},
			{
				ID:          "origin-trabalhador-pericia-fortitude",
				Name:        "Fortitude",
				Kind:        "pericia",
				Description: "Você se torna treinado em Fortitude.",
				Expertise:   "Fortitude",
				UID:         "org_m5lypuz7dx",
			},
			{
				ID:       "poder-atletico",
				Name:     "Atlético",
				Kind:     "poder",
				UID:      "org_nsu6shuexr",
				PowerUID: "pwr_x4zxg223af",
			},
		},
		UniquePower: OriginUniquePower{
			ID:          "origin-trabalhador-unique",
			Name:        "Esforçado",
			Kind:        "poder",
			Description: "Você não teme trabalho duro nem prazos apertados: recebe um bônus de +2 em todos os testes de perícias estendidos (incluindo perigos complexos).",
			Modifiers: []Modifier{
				{
					Target:    ModifierTarget{Kind: "expertiseAll"},
					Amount:    2,
					BonusType: "untyped",
					Condition: &ModifierCondition{Kind: "context", Note: "em teste estendido"},
				},
			},
		},
		UID: "org_n2y5ngk6cd",
	},
}
