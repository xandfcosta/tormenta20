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
	// DurationText é o texto do livro onde a `DurationNote`, que o motor lê, diverge dele.
	DurationText   string `json:"durationText,omitempty"`
	ExecutionNote  string `json:"executionNote,omitempty"`
	RangeNote      string `json:"rangeNote,omitempty"`
	ResistanceNote string `json:"resistanceNote,omitempty"`
	// Rolls é o dado que a magia rola ao ser conjurada — ver `book.Spell.Rolls`.
	Rolls []SpellRoll `json:"rolls,omitempty"`
}

// SpellRoll é uma parcela do dado da magia — ver `book.SpellRoll`.
type SpellRoll struct {
	Dice string `json:"dice"`
	Kind string `json:"kind,omitempty"`
	Note string `json:"note,omitempty"`
}

type SpellAugment struct {
	PMCost         int    `json:"pmCost"`
	Kind           string `json:"kind"`
	Description    string `json:"description"`
	Exclusive      bool   `json:"exclusive,omitempty"`
	RequiresCircle int    `json:"requiresCircle,omitempty"`
	ClassOnly      string `json:"classOnly,omitempty"`
	Cantrip        bool   `json:"truque,omitempty"`
	Sustains       *bool  `json:"sustains,omitempty"`
	Execution      string `json:"execution,omitempty"`
	// Adds, Rolls e ChangesTheRoll são o que o aprimoramento faz com o dado da magia — ver `book.SpellAugment`.
	Adds           []SpellRoll `json:"adds,omitempty"`
	Rolls          []SpellRoll `json:"rolls,omitempty"`
	ChangesTheRoll bool        `json:"changesTheRoll,omitempty"`
}

type SpellBuff struct {
	Modifiers    []Modifier `json:"modifiers"`
	Facts        []Fact     `json:"facts,omitempty"`
	DefaultScope string     `json:"defaultScope,omitempty"`
}
