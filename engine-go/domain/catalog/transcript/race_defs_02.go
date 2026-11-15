package transcript

// Parte 2 de 3 de `RaceDefinitions` (race_defs.go).
var raceDefinitionsPart02 = []RaceDefinition{
	{
		ID:               "qareen",
		Name:             "Qareen",
		Type:             "humanoide",
		AttributeBonuses: OrderedMap[int]{{"charisma", 2}, {"intelligence", 1}, {"wisdom", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "qareen-desejos",
				RaceID:      "qareen",
				Name:        "Desejos",
				Description: "Se lançar uma magia que alguém tenha pedido desde seu último turno, o custo da magia diminui em –1 PM. Fazer um desejo ao qareen é uma ação livre.",
				UID:         "rac_5tafsebqv6",
				Unmodeled:   "é regra de magia, e o motor de magia é do capítulo 4 (ALE-465)",
			},
			{
				ID:          "qareen-resistencia-elemental",
				RaceID:      "qareen",
				Name:        "Resistência Elemental",
				Description: "Conforme sua ascendência, recebe redução 10 a um tipo de dano. Escolha uma das variantes abaixo.",
				UID:         "rac_fgbigor4yt",
				Variants: []RaceAbilityVariant{
					{
						ID:          "qareen-res-frio",
						Name:        "Frio (qareen da água)",
						Description: "Redução 10 a frio.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageReduction", DamageType: "frio"},
								Amount:    10,
								BonusType: "untyped",
								Note:      "redução de frio 10",
							},
						},
					},
					{
						ID:          "qareen-res-eletricidade",
						Name:        "Eletricidade (do ar)",
						Description: "Redução 10 a eletricidade.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageReduction", DamageType: "eletricidade"},
								Amount:    10,
								BonusType: "untyped",
								Note:      "redução de eletricidade 10",
							},
						},
					},
					{
						ID:          "qareen-res-fogo",
						Name:        "Fogo (do fogo)",
						Description: "Redução 10 a fogo.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageReduction", DamageType: "fogo"},
								Amount:    10,
								BonusType: "untyped",
								Note:      "redução de fogo 10",
							},
						},
					},
					{
						ID:          "qareen-res-acido",
						Name:        "Ácido (da terra)",
						Description: "Redução 10 a ácido.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageReduction", DamageType: "acido"},
								Amount:    10,
								BonusType: "untyped",
								Note:      "redução de acido 10",
							},
						},
					},
					{
						ID:          "qareen-res-luz",
						Name:        "Luz (da luz)",
						Description: "Redução 10 a luz.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageReduction", DamageType: "luz"},
								Amount:    10,
								BonusType: "untyped",
								Note:      "redução de luz 10",
							},
						},
					},
					{
						ID:          "qareen-res-trevas",
						Name:        "Trevas (qareen das trevas)",
						Description: "Redução 10 a trevas.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageReduction", DamageType: "trevas"},
								Amount:    10,
								BonusType: "untyped",
								Note:      "redução de trevas 10",
							},
						},
					},
				},
			},
			{
				ID:          "qareen-tatuagem-mistica",
				RaceID:      "qareen",
				Name:        "Tatuagem Mística",
				Description: "Você pode lançar uma magia de 1º círculo a sua escolha (atributo-chave Carisma). Caso aprenda novamente essa magia, seu custo diminui em –1 PM.",
				UID:         "rac_gonx6fnbue",
				Unmodeled:   "é regra de magia, e o motor de magia é do capítulo 4 (ALE-465)",
			},
		},
		UID: "rac_7zdkeazq4b",
	},
	{
		ID:               "suraggel",
		Name:             "Suraggel",
		Type:             "espirito",
		AttributeBonuses: OrderedMap[int]{},
		Abilities: []RaceAbility{
			{
				ID:          "suraggel-heranca",
				Modeled:     []string{"a visão no escuro aparece como fato; o tipo espírito é o `tipo` da raça"},
				RaceID:      "suraggel",
				Name:        "Herança Divina",
				Description: "Você é uma criatura do tipo espírito e recebe visão no escuro.",
				UID:         "rac_w6lwa7527h",
				Facts:       []Fact{{Category: "sense", Text: "Visão no escuro"}},
			},
			{
				ID:          "suraggel-luz-sagrada",
				Modeled:     []string{"o +2 em Diplomacia e Intuição"},
				RaceID:      "suraggel",
				Name:        "Luz Sagrada",
				Description: "Aggelus. Você recebe +2 em Diplomacia e Intuição. Além disso, pode lançar Luz (como uma magia divina; atributo-chave Carisma). Caso aprenda novamente essa magia, seu custo diminui em –1 PM.",
				UID:         "rac_6kb2436pvi",
				Unmodeled:   "lançar Luz como magia divina: o motor de magia (ALE-465)",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "expertise", Name: "Diplomacia"}, Amount: 2, BonusType: "untyped"},
					{Target: ModifierTarget{Kind: "expertise", Name: "Intuição"}, Amount: 2, BonusType: "untyped"},
				},
				Ancestry: "aggelus",
			},
			{
				ID:          "suraggel-sombras-profanas",
				Modeled:     []string{"o +2 em Enganação e Furtividade"},
				RaceID:      "suraggel",
				Name:        "Sombras Profanas",
				Description: "Sulfure. Você recebe +2 em Enganação e Furtividade. Além disso, pode lançar Escuridão (como uma magia divina; atributo-chave Inteligência). Caso aprenda novamente essa magia, seu custo diminui em –1 PM.",
				UID:         "rac_mdpv3zhlwj",
				Unmodeled:   "lançar Escuridão como magia divina: o motor de magia (ALE-465)",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "expertise", Name: "Enganação"}, Amount: 2, BonusType: "untyped"},
					{Target: ModifierTarget{Kind: "expertise", Name: "Furtividade"}, Amount: 2, BonusType: "untyped"},
				},
				Ancestry: "sulfure",
			},
		},
		UID: "rac_jfdp3txvv4",
	},
	{
		ID:               "silfide",
		Name:             "Sílfide",
		Type:             "espirito",
		AttributeBonuses: OrderedMap[int]{{"charisma", 2}, {"dexterity", 1}, {"strength", -2}},
		Abilities: []RaceAbility{
			{
				ID: "silfide-asas-de-borboleta",
				Modeled: []string{
					"o tamanho Minúsculo soma em Furtividade, manobras e no passo da arma reduzida pela Tab. 1-21 (`raceSizeItem`)",
				},
				RaceID:      "silfide",
				Name:        "Asas de Borboleta",
				Description: "Seu tamanho é Minúsculo. Pode pairar a 1,5m do chão com deslocamento 9m, ignorando terreno difícil e imune a dano por queda. Gastando 1 PM por rodada, voa com deslocamento 12m.",
				UID:         "rac_ljzrjw3hlq",
				Unmodeled:   "o pairar e o voo de 12m: é uma forma de ataque ou movimento que o motor ainda não tem (ALE-654)",
			},
			{
				ID:          "silfide-espirito-da-natureza",
				RaceID:      "silfide",
				Name:        "Espírito da Natureza",
				Description: "Você é uma criatura do tipo espírito, recebe visão na penumbra e pode falar com animais livremente.",
				UID:         "rac_yspdaw2jpl",
				Unmodeled:   "espírito: o tipo está no catálogo da raça, mas não chega ao alvo da mesa nem às imunidades do tipo (p228) (ALE-650); a visão na penumbra e a fala com animais são prosa",
			},
			{
				ID:          "silfide-magia-das-fadas",
				RaceID:      "silfide",
				Name:        "Magia das Fadas",
				Description: "Pode lançar duas das magias a seguir (atributo-chave Carisma): Criar Ilusão, Enfeitiçar, Luz (como magia arcana) e Sono. Caso aprenda novamente uma dessas magias, seu custo diminui em –1 PM.",
				UID:         "rac_gihxnrd5o7",
				Unmodeled:   "é regra de magia, e o motor de magia é do capítulo 4 (ALE-465)",
			},
		},
		UID: "rac_m5u3mczntv",
	},
	{
		ID:               "sereia-tritao",
		Name:             "Sereia/Tritão",
		Type:             "humanoide",
		AttributeBonuses: OrderedMap[int]{},
		Abilities: []RaceAbility{
			{
				ID:          "sereia-tres-atributos",
				Modeled:     []string{"os três +1 entram pelo `raceAttributeChoices`"},
				RaceID:      "sereia-tritao",
				Name:        "+1 em Três Atributos Diferentes",
				Description: "Distribua +1 em três atributos diferentes (já aplicado na ficha).",
				UID:         "rac_dg7bnsv4h7",
			},
			{
				ID:          "sereia-cancao-dos-mares",
				RaceID:      "sereia-tritao",
				Name:        "Canção dos Mares",
				Description: "Pode lançar duas das magias a seguir: Amedrontar, Comando, Despedaçar, Enfeitiçar, Hipnotismo ou Sono (atributo-chave Carisma). Caso aprenda novamente uma dessas magias, seu custo diminui em –1 PM.",
				UID:         "rac_cvz63biaim",
				Unmodeled:   "é regra de magia, e o motor de magia é do capítulo 4 (ALE-465)",
			},
			{
				ID:          "sereia-mestre-do-tridente",
				Modeled:     []string{"o +2 no dano com azagaias, lanças e tridentes, e os itens específicos feitos deles"},
				RaceID:      "sereia-tritao",
				Name:        "Mestre do Tridente",
				Description: "Para você, o tridente é uma arma simples. Você recebe +2 em rolagens de dano com azagaias, lanças e tridentes.",
				UID:         "rac_rojbbpkrph",
				Unmodeled:   "o tridente como arma simples: a habilidade não concede proficiência (ALE-649)",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "damage", Scope: "weapon:azagaia"}, Amount: 2, BonusType: "untyped"},
					{Target: ModifierTarget{Kind: "damage", Scope: "weapon:lanca"}, Amount: 2, BonusType: "untyped"},
					{
						Target:    ModifierTarget{Kind: "damage", Scope: "weapon:lanca-montada"},
						Amount:    2,
						BonusType: "untyped",
					},
					{Target: ModifierTarget{Kind: "damage", Scope: "weapon:tridente"}, Amount: 2, BonusType: "untyped"},
				},
			},
			{
				ID:          "sereia-transformacao-anfibia",
				RaceID:      "sereia-tritao",
				Name:        "Transformação Anfíbia",
				Description: "Pode respirar debaixo d’água e possui cauda com deslocamento de natação 12m. Fora d’água, a cauda dá lugar a pernas (deslocamento 9m). Mais de um dia sem contato com água impede recuperação de PM por descanso.",
				UID:         "rac_axjfztusy3",
				Unmodeled:   "natação de 12m: é uma forma de movimento que o motor ainda não tem (ALE-654); e o PM sem água é do mestre, que sabe se houve água (decisão do dono)",
			},
		},
		UID: "rac_kcyddre5st",
	},
}
