package transcript

// Parte 3 de 3 de `RaceDefinitions` (race_defs.go).
var raceDefinitionsPart03 = []RaceDefinition{
	{
		ID:               "golem",
		Name:             "Golem",
		AttributeBonuses: OrderedMap[int]{{"strength", 2}, {"constitution", 1}, {"charisma", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "golem-chassi",
				Modeled:     []string{"+2 na Defesa, –2 de penalidade de armadura, deslocamento sem redução"},
				RaceID:      "golem",
				Name:        "Chassi",
				Description: "Seu corpo artificial é resistente, mas rígido. Deslocamento 6m, não reduzido por armadura ou excesso de carga. Recebe +2 na Defesa, mas possui penalidade de armadura –2. Sua armadura precisa de um dia para vestir/remover e não conta no limite de itens em uso.",
				UID:         "rac_533hrox6zi",
				Unmodeled:   "a armadura leva um dia para vestir e não conta no limite de itens: é prosa do mestre: o livro descreve o que acontece na mesa, e não um número",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "defense"}, Amount: 2, BonusType: "untyped", Note: "Chassi"},
					{Target: ModifierTarget{Kind: "armorPenalty"}, Amount: -2, BonusType: "untyped", Note: "Chassi"},
					{
						Target:    ModifierTarget{Kind: "flag", Name: "displacement-ignores-armor"},
						Amount:    1,
						BonusType: "untyped",
					},
					{
						Target:    ModifierTarget{Kind: "flag", Name: "displacement-ignores-load"},
						Amount:    1,
						BonusType: "untyped",
					},
				},
			},
			{
				ID: "golem-criatura-artificial",
				Modeled: []string{
					"a visão no escuro e as imunidades aparecem como fato",
					"descansa sempre em condição normal",
				},
				RaceID:      "golem",
				Name:        "Criatura Artificial",
				Description: "Você é uma criatura do tipo construto. Visão no escuro e imunidade a cansaço, metabólicos e veneno. Não respira, alimenta-se ou dorme, mas não se beneficia de cura mundana ou itens de alimentação. Precisa ficar inerte 8h por dia para recarregar (descansa em condições normais sempre).",
				UID:         "rac_enocpcdpss",
				Unmodeled:   "construto e sem cura mundana: é o TIPO de criatura da raça, que a ficha não declara (ALE-469)",
				Facts: []Fact{
					{Category: "sense", Text: "Visão no escuro"},
					{Category: "immunity", Text: "Imune a cansaço, metabólicos e veneno"},
				},
				Modifiers: []Modifier{
					{
						Target:    ModifierTarget{Kind: "flag", Name: "rest-always-normal"},
						Amount:    1,
						BonusType: "untyped",
						Note:      "descansa sempre em condição normal",
					},
				},
			},
			{
				ID:          "golem-fonte-elemental",
				RaceID:      "golem",
				Name:        "Fonte Elemental",
				Description: "Você possui um espírito elemental preso em seu corpo. Imune ao tipo de dano associado; se sofreria dano mágico desse tipo, em vez disso cura PV em metade do dano. Escolha sua origem elemental:",
				UID:         "rac_iuopumis6i",
				Unmodeled:   "a imunidade ao tipo de dano escolhido, e a cura pelo dano mágico dele: é uma forma de ataque ou movimento que o motor ainda não tem, sem fatia",
				Variants: []RaceAbilityVariant{
					{ID: "golem-fonte-agua", Name: "Água (frio)", Description: "Imune a frio."},
					{ID: "golem-fonte-ar", Name: "Ar (eletricidade)", Description: "Imune a eletricidade."},
					{ID: "golem-fonte-fogo", Name: "Fogo (fogo)", Description: "Imune a fogo."},
					{ID: "golem-fonte-terra", Name: "Terra (ácido)", Description: "Imune a ácido."},
				},
			},
			{
				ID:          "golem-proposito-de-criacao",
				RaceID:      "golem",
				Name:        "Propósito de Criação",
				Description: "Você foi construído pronto para um propósito específico e não teve infância. Não tem direito a escolher uma origem, mas recebe um poder geral a sua escolha.",
				UID:         "rac_itj2en6aii",
				Unmodeled:   "sem origem e com um poder geral, que se escolhe à mão: é prosa do mestre: o livro descreve o que acontece na mesa, e não um número",
			},
		},
		UID: "rac_slnpwagamn",
	},
	{
		ID:               "trog",
		Name:             "Trog",
		AttributeBonuses: OrderedMap[int]{{"constitution", 2}, {"strength", 1}, {"intelligence", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "trog-mau-cheiro",
				RaceID:      "trog",
				Name:        "Mau Cheiro",
				Description: "Pode gastar uma ação padrão e 2 PM para expelir um gás fétido. Criaturas (exceto trogs) em alcance curto devem passar em Fortitude contra veneno (CD Con) ou ficam enjoadas por 1d6 rodadas. Quem passar fica imune por um dia.",
				UID:         "rac_5szvbiermh",
				Unmodeled:   "2 PM, Fortitude contra enjoado: é gastar PM num gesto que a aba Ações não oferece (ALE-492)",
			},
			{
				ID:          "trog-mordida",
				RaceID:      "trog",
				Name:        "Mordida",
				Description: "Arma natural de mordida (dano 1d6, crítico x2, perfuração). Uma vez por rodada, ao usar agredir com outra arma, pode gastar 1 PM para fazer ataque corpo a corpo extra com a mordida.",
				UID:         "rac_xx6tdynfxt",
				Unmodeled:   "a arma natural e o ataque extra por 1 PM: é uma forma de ataque ou movimento que o motor ainda não tem, sem fatia",
			},
			{
				ID:          "trog-reptiliano",
				Modeled:     []string{"+1 na Defesa", "o +5 em Furtividade, que o jogador liga"},
				RaceID:      "trog",
				Name:        "Reptiliano",
				Description: "Criatura do tipo monstro, visão no escuro, +1 na Defesa e, se estiver sem armadura ou roupas pesadas, +5 em Furtividade.",
				UID:         "rac_7seadwuucv",
				Unmodeled:   "monstro e visão no escuro: é o TIPO de criatura da raça, que a ficha não declara (ALE-469)",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "defense"}, Amount: 1, BonusType: "untyped", Note: "Reptiliano"},
					{
						Target:    ModifierTarget{Kind: "expertise", Name: "Furtividade"},
						Amount:    5,
						BonusType: "untyped",
						Condition: &ModifierCondition{Kind: "flagOn", Flag: "reptiliano", Label: "sem armadura nem roupas pesadas"},
					},
				},
			},
			{
				ID:          "trog-sangue-frio",
				RaceID:      "trog",
				Name:        "Sangue Frio",
				Description: "Você sofre 1 ponto de dano adicional por dado de dano de frio.",
				UID:         "rac_d7ur63vhtr",
				Unmodeled:   "+1 de dano por dado de frio sofrido: é uma forma de ataque ou movimento que o motor ainda não tem, sem fatia",
			},
		},
		UID: "rac_avgggbkcvx",
	},
}
