package transcript

type GeneralPower struct {
	ID            string         `json:"id"`
	Kind          string         `json:"kind"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Prerequisites []Prerequisite `json:"prerequisites,omitempty"`
	BookPage      int            `json:"bookPage"`
	UID           string         `json:"uid"`
	Modifiers     []Modifier     `json:"modifiers,omitempty"`
	MinLevel      int            `json:"minLevel,omitempty"`
	Choice        *PowerChoice   `json:"choice,omitempty"`
}
