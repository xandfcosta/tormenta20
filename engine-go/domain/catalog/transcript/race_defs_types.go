package transcript

type RaceDefinition struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	AttributeBonuses     OrderedMap[int] `json:"attributeBonuses"`
	Abilities            []RaceAbility   `json:"abilities"`
	UID                  string          `json:"uid"`
	GrantsTormentaPowers bool            `json:"grantsTormentaPowers,omitempty"`
}

type RaceAbility struct {
	ID          string               `json:"id"`
	Modeled     []string             `json:"modeled,omitempty"`
	RaceID      string               `json:"raceId"`
	Name        string               `json:"name"`
	Description string               `json:"description"`
	UID         string               `json:"uid"`
	Unmodeled   string               `json:"unmodeled,omitempty"`
	Facts       []Fact               `json:"facts,omitempty"`
	Modifiers   []Modifier           `json:"modifiers,omitempty"`
	Variants    []RaceAbilityVariant `json:"variants,omitempty"`
	Choice      *PowerChoice         `json:"choice,omitempty"`
	// GrantsProficiency são as proficiências que a habilidade dá (o
	// Vanguardista, armas de fogo).
	GrantsProficiency []string `json:"grantsProficiency,omitempty"`
}

type RaceAbilityVariant struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Modifiers   []Modifier `json:"modifiers,omitempty"`
}
