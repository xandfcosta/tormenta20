package transcript

type ModifierTarget struct {
	Kind       string `json:"k"`
	Scope      string `json:"scope,omitempty"`
	Name       string `json:"name,omitempty"`
	DamageType string `json:"damageType,omitempty"`
	Attribute  string `json:"attribute,omitempty"`
	School     string `json:"school,omitempty"`
}

type Modifier struct {
	Target            ModifierTarget     `json:"target"`
	Amount            float64            `json:"amount,omitempty"`
	BonusType         string             `json:"bonusType"`
	Note              string             `json:"note,omitempty"`
	Condition         *ModifierCondition `json:"condition,omitempty"`
	Dice              string             `json:"dice,omitempty"`
	Factor            *ModifierFactor    `json:"factor,omitempty"`
	Scale             *ModifierScale     `json:"scale,omitempty"`
	FromChoice        string             `json:"fromChoice,omitempty"`
	AppliesToCategory []string           `json:"appliesToCategory,omitempty"`
}

type ModifierCondition struct {
	Kind string `json:"c"`
	Note string `json:"note,omitempty"`
	Flag string `json:"flag,omitempty"`
	// Flags são as travas A MAIS de um `flagOff`/`flagRequired`: o `flagOff`
	// cala se qualquer uma estiver posta, o `flagRequired` pede todas.
	Flags []string `json:"flags,omitempty"`
	Label string   `json:"label,omitempty"`
	Trait string   `json:"trait,omitempty"`
	Type  string   `json:"type,omitempty"`
}

type PMScaling struct {
	BasePM          int    `json:"basePm"`
	StepPM          int    `json:"stepPm"`
	StepLabel       string `json:"stepLabel"`
	FirstStepLevel  int    `json:"firstStepLevel"`
	StepEveryLevels int    `json:"stepEveryLevels"`
	StepAmount      int    `json:"stepAmount,omitempty"`
}

type ModifierFactor struct {
	Num int `json:"num"`
	Den int `json:"den"`
}

type Prerequisite struct {
	Kind        string   `json:"kind"`
	ID          string   `json:"id,omitempty"`
	Description string   `json:"description,omitempty"`
	IDs         []string `json:"ids,omitempty"`
	Expertise   string   `json:"expertise,omitempty"`
	Attr        string   `json:"attr,omitempty"`
	Min         int      `json:"min,omitempty"`
	Class       string   `json:"class,omitempty"`
	Field       string   `json:"field,omitempty"`
	Label       string   `json:"label,omitempty"`
	Allowed     []string `json:"allowed,omitempty"`
	Forbidden   []string `json:"forbidden,omitempty"`
}

type ModifierScale struct {
	Per       string `json:"per"`
	Attribute string `json:"attribute,omitempty"`
	Step      int    `json:"step,omitempty"`
	Round     string `json:"round,omitempty"`
	// Cap é o teto de um `per: attribute`: `classLevel` é o nível na classe
	// que concede o poder (p226). Vazio é sem teto.
	Cap string `json:"cap,omitempty"`
}

type Fact struct {
	Category string `json:"category"`
	Text     string `json:"text"`
}
