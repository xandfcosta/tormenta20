package transcript

type Spell struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Circle       int            `json:"circle"`
	School       string         `json:"school"`
	Execution    string         `json:"execution"`
	Range        string         `json:"range"`
	Targeting    string         `json:"targeting"`
	Duration     string         `json:"duration"`
	SaveType     string         `json:"saveType"`
	Resistance   *string        `json:"resistance"`
	Components   []string       `json:"components"`
	Classes      []string       `json:"classes"`
	BaseEffect   string         `json:"baseEffect"`
	Augments     []SpellAugment `json:"augments"`
	BookPage     int            `json:"bookPage"`
	UID          string         `json:"uid"`
	Buff         *SpellBuff     `json:"buff,omitempty"`
	DurationNote string         `json:"durationNote,omitempty"`
	// As notas são o texto do livro onde o campo não tem como dizer: o `Execution` é o custo de
	// ação que o motor cobra, e "1 hora" não é um deles.
	ExecutionNote  string `json:"executionNote,omitempty"`
	RangeNote      string `json:"rangeNote,omitempty"`
	ResistanceNote string `json:"resistanceNote,omitempty"`
}

type SpellAugment struct {
	PMCost         int    `json:"pmCost"`
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Exclusive      bool   `json:"exclusive,omitempty"`
	RequiresCircle int    `json:"requiresCircle,omitempty"`
	ClassOnly      string `json:"classOnly,omitempty"`
	Cantrip        bool   `json:"truque,omitempty"`
}

type SpellBuff struct {
	Modifiers    []Modifier `json:"modifiers"`
	Facts        []Fact     `json:"facts,omitempty"`
	DefaultScope string     `json:"defaultScope,omitempty"`
}
