package transcript

type ClassPower struct {
	ID              string         `json:"id"`
	Modeled         []string       `json:"modeled,omitempty"`
	ClassName       string         `json:"className,omitempty"`
	Name            string         `json:"name,omitempty"`
	Description     string         `json:"description,omitempty"`
	GrantedAtLevel  int            `json:"grantedAtLevel,omitempty"`
	BookPage        int            `json:"bookPage,omitempty"`
	UID             string         `json:"uid"`
	PowerUID        string         `json:"powerUid,omitempty"`
	Modifiers       []Modifier     `json:"modifiers,omitzero"`
	Unmodeled       string         `json:"unmodeled,omitempty"`
	Prerequisites   []Prerequisite `json:"prerequisites,omitempty"`
	Choice          *PowerChoice   `json:"choice,omitempty"`
	MinLevel        int            `json:"minLevel,omitempty"`
	GrantedByChoice *ChoiceGrant   `json:"grantedByChoice,omitempty"`
	Degrees         *PMScaling     `json:"degrees,omitempty"`
}

type PowerChoice struct {
	Kind                 string              `json:"kind"`
	Label                string              `json:"label"`
	Repeatable           bool                `json:"repeatable,omitempty"`
	Options              []PowerChoiceOption `json:"options,omitempty"`
	GrantsSpellAttribute string              `json:"grantsSpellAttribute,omitempty"`
}

type PowerChoiceOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
}

type ChoiceGrant struct {
	Field string `json:"field"`
	Value string `json:"value"`
}
