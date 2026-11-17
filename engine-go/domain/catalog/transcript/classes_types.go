package transcript

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
	SchoolRestriction   *int                         `json:"schoolRestriction"`
	UnlockLevel         OrderedMap[*int]             `json:"unlockLevel"`
	MaxCircle           int                          `json:"maxCircle"`
	StartingSpellsKnown int                          `json:"startingSpellsKnown"`
	LearnCadence        string                       `json:"learnCadence"`
	Paths               OrderedMap[SpellcastingPath] `json:"paths,omitempty"`
}

type SpellcastingPath struct {
	LearnCadence        string `json:"learnCadence"`
	StartingSpellsKnown int    `json:"startingSpellsKnown,omitempty"`
}
