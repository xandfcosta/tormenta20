package transcript

type Race struct {
	ID             string               `json:"id"`
	Name           string               `json:"name"`
	Tier           string               `json:"tier"`
	AttributeRule  RaceAttributeRule    `json:"atributoMod"`
	Size           string               `json:"tamanho"`
	Displacement   int                  `json:"deslocamento"`
	Darkvision     bool                 `json:"visaoNoEscuro"`
	LowLightVision bool                 `json:"visaoNaPenumbra"`
	Abilities      []RaceAbilitySummary `json:"abilities"`
	BookPage       int                  `json:"bookPage"`
	UID            string               `json:"uid"`
	Ancestries     []string             `json:"ascendencias,omitempty"`
}

type RaceAttributeRule struct {
	Kind     string                      `json:"kind"`
	Count    int                         `json:"count,omitempty"`
	Value    int                         `json:"value,omitempty"`
	Mods     OrderedMap[int]             `json:"mods,omitempty"`
	Exclude  string                      `json:"exclude,omitempty"`
	Penalty  *AttributePenalty           `json:"penalty,omitempty"`
	Variants OrderedMap[OrderedMap[int]] `json:"variants,omitempty"`
}

type RaceAbilitySummary struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

type AttributePenalty struct {
	Attribute string `json:"attribute"`
	Value     int    `json:"value"`
}
