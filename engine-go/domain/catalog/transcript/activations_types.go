package transcript

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
	SkillSpend   *SkillSpend      `json:"skillSpend,omitempty"`
}

type SkillSpend struct {
	Kind      string `json:"kind"`
	Attribute string `json:"attribute,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	SkillsOf  string `json:"skillsOf,omitempty"`
	Group     string `json:"group,omitempty"`
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
