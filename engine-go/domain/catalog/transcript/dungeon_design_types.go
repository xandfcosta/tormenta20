package transcript

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
