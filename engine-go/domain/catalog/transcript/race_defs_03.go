package transcript

// Parte 3 de 3 de `RaceDefinitions` (race_defs.go).
var raceDefinitionsPart03 = []RaceDefinition{
	{
		ID:               "osteon",
		Name:             "Osteon",
		Type:             "morto-vivo",
		AttributeBonuses: OrderedMap[int]{{"constitution", -1}},
		Abilities: []RaceAbility{
			{
				ID:          "osteon-tres-atributos",
				Modeled:     []string{"os três +1 e o –1 em Constituição entram pelo `raceAttributeChoices`"},
				RaceID:      "osteon",
				Name:        "+1 em Três Atributos Diferentes (exceto Constituição)",
				Description: "Distribua +1 em três atributos diferentes, exceto Constituição (já aplicado na ficha).",
				UID:         "rac_xr2sef46ac",
			},
			{
				ID:          "osteon-armadura-ossea",
				RaceID:      "osteon",
				Name:        "Armadura Óssea",
				Description: "Você recebe redução de corte, frio e perfuração 5.",
				UID:         "rac_4vdeiyhe3c",
				Facts:       []Fact{{Category: "dr", Text: "RD 5 a corte, frio e perfuração"}},
				Modifiers: []Modifier{
					{
						Target:    ModifierTarget{Kind: "damageReduction", DamageType: "corte"},
						Amount:    5,
						BonusType: "untyped",
						Note:      "redução de corte 5",
					},
					{
						Target:    ModifierTarget{Kind: "damageReduction", DamageType: "frio"},
						Amount:    5,
						BonusType: "untyped",
						Note:      "redução de frio 5",
					},
					{
						Target:    ModifierTarget{Kind: "damageReduction", DamageType: "perfuracao"},
						Amount:    5,
						BonusType: "untyped",
						Note:      "redução de perfuracao 5",
					},
				},
			},
			{
				ID: "osteon-memoria-postuma",
				Modeled: []string{
					"a perícia escolhida se marca treinada na coluna da ficha, que é o lugar da escolha livre",
					"a habilidade emprestada de outra raça humanoide, e o tamanho dela",
				},
				RaceID:      "osteon",
				Name:        "Memória Póstuma",
				Description: "Você se torna treinado em uma perícia (não precisa ser da sua classe) ou recebe um poder geral a sua escolha. Alternativamente, escolha uma habilidade de outra raça humanoide (que não humano); herda também o tamanho dessa raça se diferente de Médio.",
				UID:         "rac_ewrcutxhsw",
				Choice:      &PowerChoice{Kind: "trained-or-power", Label: "Perícia ou poder geral", Count: 1},
			},
			{
				ID: "osteon-natureza-esqueletica",
				Modeled: []string{
					"a visão no escuro e as imunidades aparecem como fato",
					"o tipo morto-vivo na mesa: o alvo da Anticriatura e as imunidades do tipo (p228)",
				},
				RaceID:      "osteon",
				Name:        "Natureza Esquelética",
				Description: "Criatura do tipo morto-vivo. Visão no escuro e imunidade a cansaço, metabólicos, trevas e veneno. Não precisa respirar, comer ou dormir. Efeitos mágicos de cura de luz causam dano; dano de trevas recupera PV.",
				UID:         "rac_f64irbsgi3",
				Unmodeled:   "a cura de luz que fere e as trevas que curam são do mestre",
				Facts: []Fact{
					{Category: "sense", Text: "Visão no escuro"},
					{Category: "immunity", Text: "Imune a cansaço, metabólicos, trevas e veneno"},
				},
			},
			{
				ID: "osteon-preco-da-nao-vida",
				Modeled: []string{
					"descansa sempre em condição normal; as oito horas sob as estrelas são supostas cumpridas, e a fome é do mestre (decisão do dono, ALE-488)",
				},
				RaceID:      "osteon",
				Name:        "Preço da Não Vida",
				Description: "Precisa passar oito horas sob luz de estrelas ou no subterrâneo. Se fizer isso, recupera PV e PM por descanso normalmente. Caso contrário, sofre efeitos de fome.",
				UID:         "rac_ffhjotbnr5",
				Modifiers: []Modifier{
					{
						Target:    ModifierTarget{Kind: "flag", Name: "rest-always-normal"},
						Amount:    1,
						BonusType: "untyped",
						Note:      "descansa sempre em condição normal",
					},
				},
			},
		},
		UID: "rac_qqg3zuo5ec",
	},
	{
		ID:               "medusa",
		Name:             "Medusa",
		Type:             "monstro",
		AttributeBonuses: OrderedMap[int]{{"dexterity", 2}, {"charisma", 1}},
		Abilities: []RaceAbility{
			{
				ID: "medusa-cria-de-megalokk",
				Modeled: []string{
					"a visão no escuro aparece como fato",
					"o tipo monstro na mesa: o alvo da Anticriatura e as imunidades do tipo (p228)",
				},
				RaceID:      "medusa",
				Name:        "Cria de Megalokk",
				Description: "Você é uma criatura do tipo monstro e recebe visão no escuro.",
				UID:         "rac_ma2nx5ov6h",
				Facts:       []Fact{{Category: "sense", Text: "Visão no escuro"}},
			},
			{
				ID:          "medusa-natureza-venenosa",
				Modeled:     []string{"a resistência a veneno +5"},
				RaceID:      "medusa",
				Name:        "Natureza Venenosa",
				Description: "Recebe resistência a veneno +5 e pode gastar uma ação de movimento e 1 PM para envenenar uma arma. A arma causa perda de 1d12 PV adicional. Veneno dura até acertar um ataque ou fim da cena.",
				UID:         "rac_3dyes2cszk",
				Unmodeled:   "o veneno na arma (perda de 1d12 PV) é do mestre: o gesto cobra o PM",
				Modifiers: []Modifier{
					{
						Target:    ModifierTarget{Kind: "resistance"},
						Amount:    5,
						BonusType: "untyped",
						Condition: &ModifierCondition{Kind: "against", Trait: "veneno"},
					},
				},
			},
			{
				ID:          "medusa-olhar-atordoante",
				RaceID:      "medusa",
				Name:        "Olhar Atordoante",
				Description: "Pode gastar uma ação de movimento e 1 PM para forçar uma criatura em alcance curto a fazer Fortitude (CD Car). Se falhar, fica atordoada por uma rodada (apenas uma vez por cena).",
				UID:         "rac_m45ak3bgq6",
			},
		},
		UID: "rac_7ynliohbbn",
	},
	{
		ID:               "kliren",
		Name:             "Kliren",
		Type:             "humanoide",
		AttributeBonuses: OrderedMap[int]{{"intelligence", 2}, {"charisma", 1}, {"strength", -1}},
		Abilities: []RaceAbility{
			{
				ID: "kliren-hibrido",
				Modeled: []string{
					"a perícia escolhida se marca treinada na coluna da ficha, que é o lugar da escolha livre",
				},
				RaceID:      "kliren",
				Name:        "Híbrido",
				Description: "Sua natureza multifacetada o fez aprender conhecimentos variados. Você se torna treinado em uma perícia a sua escolha (não precisa ser da sua classe).",
				UID:         "rac_e7iz2ymt7d",
			},
			{
				ID: "kliren-engenhosidade",
				Modeled: []string{
					"o gasto no diálogo do teste: 2 PM somam a Inteligência, e 1 PM para quem a recebe de novo",
				},
				RaceID:      "kliren",
				Name:        "Engenhosidade",
				Description: "Quando faz um teste de perícia, pode gastar 2 PM para somar sua Inteligência no teste. Não pode usar em testes de ataque. Caso receba esta habilidade novamente, seu custo é reduzido em –1 PM.",
				UID:         "rac_xse6geohcx",
			},
			{
				ID:          "kliren-ossos-frageis",
				RaceID:      "kliren",
				Name:        "Ossos Frágeis",
				Description: "Você sofre 1 ponto de dano adicional por dado de dano de impacto.",
				UID:         "rac_bvxnjgjfnv",
				Modifiers: []Modifier{
					{
						Target:    ModifierTarget{Kind: "damageTakenPerDie", DamageType: "impacto"},
						Amount:    1,
						BonusType: "untyped",
						Note:      "Ossos Frágeis",
					},
				},
			},
			{
				ID:          "kliren-vanguardista",
				RaceID:      "kliren",
				Name:        "Vanguardista",
				Description: "Recebe proficiência em armas de fogo e +2 em Ofício (um qualquer, a sua escolha).",
				UID:         "rac_nswmnwe2uc",
				Modifiers: []Modifier{
					{
						Target:     ModifierTarget{Kind: "expertise"},
						Amount:     2,
						BonusType:  "untyped",
						Note:       "Vanguardista",
						FromChoice: "name",
					},
				},
				Choice:            &PowerChoice{Kind: "craft", Label: "Ofício", Count: 1},
				GrantsProficiency: []string{"armas-de-fogo"},
			},
		},
		UID: "rac_fmeindoywi",
	},
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
				},
				RaceID:      "hynne",
				Name:        "Pequeno e Rechonchudo",
				Description: "Seu tamanho é Pequeno e seu deslocamento é 6m. Recebe +2 em Enganação e pode usar Destreza como atributo-chave de Atletismo (em vez de Força).",
				UID:         "rac_eief7ymkwz",
				Unmodeled:   "Destreza como atributo de Atletismo: é uma forma que o motor ainda não tem (ALE-655)",
				Modifiers: []Modifier{
					{Target: ModifierTarget{Kind: "expertise", Name: "Enganação"}, Amount: 2, BonusType: "untyped"},
				},
			},
			{
				ID:          "hynne-sorte-salvadora",
				Modeled:     []string{"o rolar de novo do teste de resistência por 1 PM, na faixa do teste"},
				RaceID:      "hynne",
				Name:        "Sorte Salvadora",
				Description: "Quando faz um teste de resistência, você pode gastar 1 PM para rolar este teste novamente.",
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
				UID:         "rac_5szvbiermh",
			},
			{
				ID:            "trog-mordida",
				Modeled:       []string{"a linha de ataque da mordida e o ataque extra por 1 PM"},
				RaceID:        "trog",
				Name:          "Mordida",
				Description:   "Arma natural de mordida (dano 1d6, crítico x2, perfuração). Uma vez por rodada, ao usar agredir com outra arma, pode gastar 1 PM para fazer ataque corpo a corpo extra com a mordida.",
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
