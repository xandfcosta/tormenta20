package transcript

// DungeonDesign são as tabelas de desenho de masmorra do Capítulo 6.
var DungeonDesign = DungeonDesignTable{
	Sizes: []string{"pequena", "media", "grande"},
	SizeTable: []DungeonSize{
		{
			Size:                   "pequena",
			Label:                  "Pequena",
			MinRooms:               3,
			MaxRooms:               6,
			Pacing:                 "parte-de-sessao",
			MaxSecondaryObjectives: 0,
			OptionalObjectives:     1,
		},
		{
			Size:                   "media",
			Label:                  "Média",
			MinRooms:               7,
			MaxRooms:               20,
			Pacing:                 "sessao-inteira",
			MaxSecondaryObjectives: 1,
			OptionalObjectives:     2,
		},
		{
			Size:                   "grande",
			Label:                  "Grande",
			MinRooms:               21,
			MaxRooms:               50,
			Pacing:                 "aventura-inteira",
			MaxSecondaryObjectives: 3,
			OptionalObjectives:     3,
		},
	},
	RoomsPerThreat: 3,
	Ideas: []DungeonIdea{
		{Roll: 1, Label: "Complexo de cavernas subterrâneas"},
		{Roll: 2, Label: "Mina abandonada"},
		{Roll: 3, Label: "Templo de um deus maligno"},
		{Roll: 4, Label: "Esgotos da cidade"},
		{Roll: 5, Label: "Castelo de um déspota"},
		{Roll: 6, Label: "Torre de um mago louco"},
		{Roll: 7, Label: "Moinho da vila"},
		{Roll: 8, Label: "Armazém no porto"},
		{Roll: 9, Label: "Ruínas de uma civilização perdida"},
		{Roll: 10, Label: "Fortaleza anã abandonada"},
		{Roll: 11, Label: "Mansão assombrada"},
		{Roll: 12, Label: "Prisão da cidade"},
		{Roll: 13, Label: "Caverna submersa"},
		{Roll: 14, Label: "Gruta usada como covil por um monstro"},
		{Roll: 15, Label: "Biblioteca mágica"},
		{Roll: 16, Label: "Galeão encalhado"},
		{Roll: 17, Label: "Labirinto feito para proteger um tesouro"},
		{Roll: 18, Label: "Manicômio repleto de vilões insanos"},
		{Roll: 19, Label: "Vulcão inativo"},
		{Roll: 20, Label: "Castelo nas nuvens"},
	},
}

type DungeonDesignTable struct {
	Sizes          []string      `json:"sizes"`
	SizeTable      []DungeonSize `json:"sizeTable"`
	RoomsPerThreat int           `json:"roomsPerThreat"`
	Ideas          []DungeonIdea `json:"ideas"`
}

type DungeonSize struct {
	Size                   string `json:"size"`
	Label                  string `json:"label"`
	MinRooms               int    `json:"minRooms"`
	MaxRooms               int    `json:"maxRooms"`
	Pacing                 string `json:"pacing"`
	MaxSecondaryObjectives int    `json:"maxSecondaryObjectives"`
	OptionalObjectives     int    `json:"optionalObjectives"`
}

type DungeonIdea struct {
	Roll  int    `json:"roll"`
	Label string `json:"label"`
}
