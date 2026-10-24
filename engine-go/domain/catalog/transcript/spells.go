package transcript

// Spells são as magias do Capítulo 4. A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var Spells = concat(
	spellsPart01,
	spellsPart02,
	spellsPart03,
	spellsPart04,
	spellsPart05,
	spellsPart06,
	spellsPart07,
	spellsPart08,
	spellsPart09,
	spellsPart10,
	spellsPart11,
	spellsPart12,
	spellsPart13,
	spellsPart14,
)

type Spell struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Circle       int            `json:"circle"`
	School       string         `json:"school"`
	Execution    string         `json:"execution"`
	Range        string         `json:"range"`
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
