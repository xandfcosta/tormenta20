package transcript

// Parte 2 de 2 de `Races` (races.go).
var racesPart02 = []Race{
	{
		ID:   "trog",
		Name: "Trog",
		Tier: "extra",
		AttributeRule: RaceAttributeRule{
			Kind: "fixed",
			Mods: OrderedMap[int]{{"constitution", 2}, {"strength", 1}, {"intelligence", -1}},
		},
		Size:           "Médio",
		Displacement:   9,
		Darkvision:     true,
		LowLightVision: false,
		Abilities: []RaceAbilitySummary{
			{
				Name:    "Mau Cheiro",
				Summary: "Ação padrão + 2 PM: outras criaturas em curto fazem Fortitude vs veneno (CD Con); falha = enjoadas 1d6 rodadas.",
			},
			{
				Name:    "Mordida",
				Summary: "Arma natural 1d6/x2/perfuração; 1×/rodada na ação agredir, 1 PM dá ataque corpo a corpo extra.",
			},
			{
				Name:    "Reptiliano",
				Summary: "Monstro; visão no escuro; +1 Defesa; +5 em Furtividade sem armadura/roupas pesadas.",
			},
			{Name: "Sangue Frio", Summary: "+1 ponto de dano adicional por dado de dano de frio sofrido."},
		},
		BookPage: 31,
		UID:      "rac_luj33il7si",
	},
}
