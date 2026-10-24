package transcript

// RaceDefinitions são as habilidades de raça, com os modificadores que somam na ficha. A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var RaceDefinitions = concat(
	raceDefinitionsPart01,
	raceDefinitionsPart02,
	raceDefinitionsPart03,
)

type RaceDefinition struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	AttributeBonuses     OrderedMap[int] `json:"attributeBonuses"`
	Abilities            []RaceAbility   `json:"abilities"`
	UID                  string          `json:"uid"`
	GrantsTormentaPowers bool            `json:"grantsTormentaPowers,omitempty"`
	HasDeformity         bool            `json:"hasDeformidade,omitempty"`
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
}

type RaceAbilityVariant struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Modifiers   []Modifier `json:"modifiers,omitempty"`
}
