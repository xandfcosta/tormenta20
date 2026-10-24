package transcript

// Activations são o que cada poder CUSTA para ser usado — ação, PM, usos, postura. A lista não cabe num arquivo de 500 linhas,
// então mora em partes, na ordem em que o livro transcrito sempre a guardou.
var Activations = concat(
	activationsPart01,
	activationsPart02,
	activationsPart03,
	activationsPart04,
	activationsPart05,
	activationsPart06,
	activationsPart07,
	activationsPart08,
	activationsPart09,
	activationsPart10,
)

type Activation struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Kind         string           `json:"kind"`
	Action       string           `json:"action"`
	PMCost       PMCost           `json:"pmCost"`
	Uses         *string          `json:"uses"`
	RequiresFlag string           `json:"requiresFlag,omitempty"`
	BookPage     int              `json:"bookPage"`
	Grant        *ActivationGrant `json:"grant,omitempty"`
	Cumulative   *CumulativeGrant `json:"cumulative,omitempty"`
	Flag         string           `json:"flag,omitempty"`
	StanceGroup  string           `json:"stanceGroup,omitempty"`
	Scaling      *PMScaling       `json:"scaling,omitempty"`
}

type ActivationGrant struct {
	Kind      string     `json:"kind"`
	Scope     string     `json:"scope"`
	Attribute string     `json:"attribute,omitempty"`
	Modifiers []Modifier `json:"modifiers,omitempty"`
}

type CumulativeGrant struct {
	On      string           `json:"on"`
	Amount  int              `json:"amount"`
	CapPer  string           `json:"capPer"`
	Targets []ModifierTarget `json:"targets"`
}
