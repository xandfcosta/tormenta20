package transcript

// Parte 4 de 4 de `RaceDefinitions` (race_defs.go).
var raceDefinitionsPart04 = []RaceDefinition{
	{
		ID:               "hynne",
		Name:             "Hynne",
		Type:             "humanoide",
		AttributeBonuses: OrderedMap[int]{{"dexterity", 2}, {"charisma", 1}, {"strength", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "hynne-arremessador",
				RaceID:      "hynne",
				Name:        "Arremessador",
				Description: "Quando faz um ataque à distância com uma funda ou uma arma de arremesso, seu dano aumenta em um passo.",
				BookPage:    28,
				UID:         "rac_rdh4sutt2c",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "damageStep", Scope: "thrown"}, Amount: 1, BonusType: "untyped"},
					{Target: ModifierTarget{Kind: "damageStep", Scope: "weapon:funda"}, Amount: 1, BonusType: "untyped"},
				},
			},
			{
				ID: "hynne-pequeno-e-rechonchudo",
				Modeled: []string{
					"o +2 em Enganação",
					"os 6m saem do `deslocamento` do races.json",
					"o tamanho Pequeno soma em Furtividade, manobras pela Tab. 1-21 (`raceSizeItem`)",
					"a Destreza no Atletismo, quando ela é maior que a Força (decisão do dono)",
				},
				RaceID:      "hynne",
				Name:        "Pequeno e Rechonchudo",
				Description: "Seu tamanho é Pequeno e seu deslocamento é 6m. Recebe +2 em Enganação e pode usar Destreza como atributo-chave de Atletismo (em vez de Força).",
				BookPage:    28,
				UID:         "rac_eief7ymkwz",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "expertise", Name: "Enganação"}, Amount: 2, BonusType: "untyped"},
					{
						Target:    ModifierTarget{Kind: "expertiseAttribute", Name: "Atletismo", Attribute: "dexterity"},
						BonusType: "untyped",
						Note:      "Destreza no lugar da Força em Atletismo",
					},
				},
			},
			{
				ID:          "hynne-sorte-salvadora",
				Modeled:     []string{"o rolar de novo do teste de resistência por 1 PM, na faixa do teste"},
				RaceID:      "hynne",
				Name:        "Sorte Salvadora",
				Description: "Quando faz um teste de resistência, você pode gastar 1 PM para rolar este teste novamente.",
				BookPage:    28,
				UID:         "rac_fljkqob7lk",
			},
		},
		UID: "rac_76vs5luayw",
	},
	{
		ID:               "golem",
		Name:             "Golem",
		Type:             "construto",
		AttributeBonuses: OrderedMap[int]{{"strength", 2}, {"constitution", 1}, {"charisma", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "golem-chassi",
				Modeled:     []string{"+2 na Defesa, –2 de penalidade de armadura, deslocamento sem redução"},
				RaceID:      "golem",
				Name:        "Chassi",
				Description: "Seu corpo artificial é resistente, mas rígido. Deslocamento 6m, não reduzido por armadura ou excesso de carga. Recebe +2 na Defesa, mas possui penalidade de armadura –2. Sua armadura precisa de um dia para vestir/remover e não conta no limite de itens em uso.",
				BookPage:    27,
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
					"o tipo construto na mesa: o alvo da Anticriatura e as imunidades do tipo (p228)",
				},
				RaceID:      "golem",
				Name:        "Criatura Artificial",
				Description: "Você é uma criatura do tipo construto. Visão no escuro e imunidade a cansaço, metabólicos e veneno. Não respira, alimenta-se ou dorme, mas não se beneficia de cura mundana ou itens de alimentação. Precisa ficar inerte 8h por dia para recarregar (descansa em condições normais sempre).",
				BookPage:    27,
				UID:         "rac_enocpcdpss",
				Unmodeled:   "a cura mundana que não vale é do mestre",
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
				BookPage:    27,
				UID:         "rac_iuopumis6i",
				Variants: []RaceAbilityVariant{
					{
						ID:          "golem-fonte-agua",
						Name:        "Água (frio)",
						Description: "Imune a frio.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageImmunity", DamageType: "frio"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
							{
								Target:    ModifierTarget{Kind: "magicDamageHeals", DamageType: "frio"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
						},
					},
					{
						ID:          "golem-fonte-ar",
						Name:        "Ar (eletricidade)",
						Description: "Imune a eletricidade.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageImmunity", DamageType: "eletricidade"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
							{
								Target:    ModifierTarget{Kind: "magicDamageHeals", DamageType: "eletricidade"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
						},
					},
					{
						ID:          "golem-fonte-fogo",
						Name:        "Fogo (fogo)",
						Description: "Imune a fogo.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageImmunity", DamageType: "fogo"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
							{
								Target:    ModifierTarget{Kind: "magicDamageHeals", DamageType: "fogo"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
						},
					},
					{
						ID:          "golem-fonte-terra",
						Name:        "Terra (ácido)",
						Description: "Imune a ácido.",
						Modifiers: []Modifier{
							{
								Target:    ModifierTarget{Kind: "damageImmunity", DamageType: "ácido"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
							{
								Target:    ModifierTarget{Kind: "magicDamageHeals", DamageType: "ácido"},
								Amount:    1,
								BonusType: "untyped",
								Note:      "Fonte Elemental",
							},
						},
					},
				},
			},
			{
				ID:          "golem-proposito-de-criacao",
				RaceID:      "golem",
				Name:        "Propósito de Criação",
				Description: "Você foi construído pronto para um propósito específico e não teve infância. Não tem direito a escolher uma origem, mas recebe um poder geral a sua escolha.",
				BookPage:    27,
				UID:         "rac_itj2en6aii",
				Unmodeled:   "sem origem e com um poder geral, que se escolhe à mão: é prosa do mestre: o livro descreve o que acontece na mesa, e não um número",
			},
		},
		UID: "rac_slnpwagamn",
	},
	{
		ID:               "trog",
		Name:             "Trog",
		Type:             "monstro",
		AttributeBonuses: OrderedMap[int]{{"constitution", 2}, {"strength", 1}, {"intelligence", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "trog-mau-cheiro",
				RaceID:      "trog",
				Name:        "Mau Cheiro",
				Description: "Pode gastar uma ação padrão e 2 PM para expelir um gás fétido. Criaturas (exceto trogs) em alcance curto devem passar em Fortitude contra veneno (CD Con) ou ficam enjoadas por 1d6 rodadas. Quem passar fica imune por um dia.",
				BookPage:    31,
				UID:         "rac_5szvbiermh",
			},
			{
				ID:            "trog-mordida",
				Modeled:       []string{"a linha de ataque da mordida e o ataque extra por 1 PM"},
				RaceID:        "trog",
				Name:          "Mordida",
				Description:   "Arma natural de mordida (dano 1d6, crítico x2, perfuração). Uma vez por rodada, ao usar agredir com outra arma, pode gastar 1 PM para fazer ataque corpo a corpo extra com a mordida.",
				BookPage:      31,
				UID:           "rac_xx6tdynfxt",
				NaturalWeapon: &NaturalWeapon{Name: "Mordida", Damage: "1d6", CritMult: 2, DamageType: "perfuracao", ExtraPM: 1},
			},
			{
				ID: "trog-reptiliano",
				Modeled: []string{
					"+1 na Defesa",
					"o +5 em Furtividade, que o jogador liga",
					"o tipo monstro na mesa: o alvo da Anticriatura e as imunidades do tipo (p228)",
				},
				RaceID:      "trog",
				Name:        "Reptiliano",
				Description: "Criatura do tipo monstro, visão no escuro, +1 na Defesa e, se estiver sem armadura ou roupas pesadas, +5 em Furtividade.",
				BookPage:    31,
				UID:         "rac_7seadwuucv",
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
				BookPage:    31,
				UID:         "rac_d7ur63vhtr",
				Modifiers: []Modifier{
					{
						Target:    ModifierTarget{Kind: "damageTakenPerDie", DamageType: "frio"},
						Amount:    1,
						BonusType: "untyped",
						Note:      "Sangue Frio",
					},
				},
			},
		},
		UID: "rac_avgggbkcvx",
	},
}
