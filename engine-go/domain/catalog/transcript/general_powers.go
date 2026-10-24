package transcript

// GeneralPowers são os poderes gerais do Capítulo 2. A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var GeneralPowers = concat(
	generalPowersPart01,
	generalPowersPart02,
)

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
}
