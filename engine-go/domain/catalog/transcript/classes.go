package transcript

// Classes são as 14 classes, com proficiências e a regra de conjuração.
var Classes = []Class{
	{
		ID:            "arcanista",
		Name:          "Arcanista",
		BookPage:      36,
		Proficiencies: []string{},
		Spellcasting: &Spellcasting{
			List:                "arcana",
			Attribute:           "intelligence",
			SchoolRestriction:   nil,
			UnlockLevel:         OrderedMap[*int]{{"1", ptr(1)}, {"2", ptr(5)}, {"3", ptr(9)}, {"4", ptr(13)}, {"5", ptr(17)}},
			MaxCircle:           5,
			StartingSpellsKnown: 3,
			LearnCadence:        "every-level",
			Paths: OrderedMap[SpellcastingPath]{
				{"bruxo", SpellcastingPath{Attribute: "intelligence", LearnCadence: "every-level"}},
				{"feiticeiro", SpellcastingPath{Attribute: "charisma", LearnCadence: "every-odd-level"}},
				{"mago", SpellcastingPath{Attribute: "intelligence", LearnCadence: "every-level", StartingSpellsKnown: 4}},
			},
		},
		UID: "cls_uov4q5z5jl",
	},
	{
		ID:            "barbaro",
		Name:          "Bárbaro",
		BookPage:      40,
		Proficiencies: []string{"armas-marciais", "escudos"},
		UID:           "cls_lxyxwsuxiz",
	},
	{
		ID:            "bardo",
		Name:          "Bardo",
		BookPage:      43,
		Proficiencies: []string{"armas-marciais"},
		Spellcasting: &Spellcasting{
			List:                "arcana",
			Attribute:           "charisma",
			SchoolRestriction:   ptr(3),
			UnlockLevel:         OrderedMap[*int]{{"1", ptr(1)}, {"2", ptr(6)}, {"3", ptr(10)}, {"4", ptr(14)}, {"5", nil}},
			MaxCircle:           4,
			StartingSpellsKnown: 2,
			LearnCadence:        "every-even-level",
		},
		UID: "cls_g45ckxsvny",
	},
	{
		ID:            "bucaneiro",
		Name:          "Bucaneiro",
		BookPage:      46,
		Proficiencies: []string{"armas-marciais"},
		UID:           "cls_ko2luzybmq",
	},
	{
		ID:            "cacador",
		Name:          "Caçador",
		BookPage:      49,
		Proficiencies: []string{"armas-marciais", "escudos"},
		UID:           "cls_6k4pge7wfw",
	},
	{
		ID:            "cavaleiro",
		Name:          "Cavaleiro",
		BookPage:      52,
		Proficiencies: []string{"armas-marciais", "armaduras-pesadas", "escudos"},
		UID:           "cls_ws3bguokut",
	},
	{
		ID:            "clerigo",
		Name:          "Clérigo",
		BookPage:      56,
		Proficiencies: []string{"armaduras-pesadas", "escudos"},
		Spellcasting: &Spellcasting{
			List:                "divina",
			Attribute:           "wisdom",
			SchoolRestriction:   nil,
			UnlockLevel:         OrderedMap[*int]{{"1", ptr(1)}, {"2", ptr(5)}, {"3", ptr(9)}, {"4", ptr(13)}, {"5", ptr(17)}},
			MaxCircle:           5,
			StartingSpellsKnown: 3,
			LearnCadence:        "every-level",
		},
		UID: "cls_jzd2m4lqui",
	},
	{
		ID:            "druida",
		Name:          "Druida",
		BookPage:      60,
		Proficiencies: []string{"escudos"},
		Spellcasting: &Spellcasting{
			List:                "divina",
			Attribute:           "wisdom",
			SchoolRestriction:   ptr(3),
			UnlockLevel:         OrderedMap[*int]{{"1", ptr(1)}, {"2", ptr(6)}, {"3", ptr(10)}, {"4", ptr(14)}, {"5", nil}},
			MaxCircle:           4,
			StartingSpellsKnown: 2,
			LearnCadence:        "every-even-level",
		},
		UID: "cls_iyamiwy2cy",
	},
	{
		ID:            "guerreiro",
		Name:          "Guerreiro",
		BookPage:      64,
		Proficiencies: []string{"armas-marciais", "armaduras-pesadas", "escudos"},
		UID:           "cls_3uvp3l3ni6",
	},
	{ID: "inventor", Name: "Inventor", BookPage: 67, Proficiencies: []string{}, UID: "cls_qpfgh7unpc"},
	{ID: "ladino", Name: "Ladino", BookPage: 72, Proficiencies: []string{}, UID: "cls_lk46dp3cfl"},
	{ID: "lutador", Name: "Lutador", BookPage: 75, Proficiencies: []string{}, UID: "cls_7hdzltdxmu"},
	{
		ID:            "nobre",
		Name:          "Nobre",
		BookPage:      78,
		Proficiencies: []string{"armas-marciais", "armaduras-pesadas", "escudos"},
		UID:           "cls_drudyajimm",
	},
	{
		ID:            "paladino",
		Name:          "Paladino",
		BookPage:      81,
		Proficiencies: []string{"armas-marciais", "armaduras-pesadas", "escudos"},
		Spellcasting: &Spellcasting{
			List:                "divina",
			Attribute:           "wisdom",
			SchoolRestriction:   nil,
			UnlockLevel:         OrderedMap[*int]{{"1", ptr(2)}, {"2", nil}, {"3", nil}, {"4", nil}, {"5", nil}},
			MaxCircle:           1,
			StartingSpellsKnown: 0,
			LearnCadence:        "via-power",
		},
		UID: "cls_cd5bgihzz7",
	},
}

type Class struct {
	ID            string        `json:"id"`
	Name          string        `json:"name"`
	BookPage      int           `json:"bookPage"`
	Proficiencies []string      `json:"proficiencies"`
	Spellcasting  *Spellcasting `json:"spellcasting,omitempty"`
	UID           string        `json:"uid"`
}

type Spellcasting struct {
	List                string                       `json:"list"`
	Attribute           string                       `json:"attribute"`
	SchoolRestriction   *int                         `json:"schoolRestriction"`
	UnlockLevel         OrderedMap[*int]             `json:"unlockLevel"`
	MaxCircle           int                          `json:"maxCircle"`
	StartingSpellsKnown int                          `json:"startingSpellsKnown"`
	LearnCadence        string                       `json:"learnCadence"`
	Paths               OrderedMap[SpellcastingPath] `json:"paths,omitempty"`
}

type SpellcastingPath struct {
	Attribute           string `json:"attribute"`
	LearnCadence        string `json:"learnCadence"`
	StartingSpellsKnown int    `json:"startingSpellsKnown,omitempty"`
}
