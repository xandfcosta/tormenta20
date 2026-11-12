package transcript

type ModifierTarget struct {
	Kind       string `json:"k"`
	Scope      string `json:"scope,omitempty"`
	Name       string `json:"name,omitempty"`
	DamageType string `json:"damageType,omitempty"`
	Attribute  string `json:"attribute,omitempty"`
	School     string `json:"school,omitempty"`
	// Tradition, Range e Spell dizem que magias um modificador de magia (CD, limite, custo) alcança, e
	// vazio é "todas". São campos próprios porque o `Scope` queria dizer tradição num lugar e alcance no
	// outro, e o −1 do cajado do poder caiu entre os dois.
	Tradition string `json:"tradition,omitempty"`
	Range     string `json:"range,omitempty"`
	Spell     string `json:"spell,omitempty"`
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
	// Ends diz quando o interruptor de um `flagOn` desliga sozinho na sessão ao vivo: "turn" quando a vez
	// passa, "own-turn" quando ela volta para o dono. Vazio fica ligado até alguém desligar.
	Ends string `json:"ends,omitempty"`
	// Requires e Unless são as flags que o termo pede, ou que o calam, em qualquer condição: o "se também
	// possuir" do livro num `flagOn`, e o "um bardo que empunhe" num `wielded`.
	Requires []string `json:"requires,omitempty"`
	Unless   []string `json:"unless,omitempty"`
	Trait    string   `json:"trait,omitempty"`
	Type     string   `json:"type,omitempty"`
}

type PMScaling struct {
	BasePM          int    `json:"basePm"`
	StepPM          int    `json:"stepPm"`
	StepLabel       string `json:"stepLabel"`
	FirstStepLevel  int    `json:"firstStepLevel"`
	StepEveryLevels int    `json:"stepEveryLevels"`
	StepAmount      int    `json:"stepAmount,omitempty"`
	// StepDice é o dado de cada degrau quando o degrau TROCA o dado por outro
	// da tabela da classe, em vez de somar: a Marca da Presa vai de 1d4 a 1d8,
	// 1d12, 2d8 e 2d10 (p50).
	StepDice []string `json:"stepDice,omitempty"`
	// CapAttribute e CapPm: o teto do custo variável que o verbete escreve,
	// abaixo do teto de nível da p224 (ALE-506).
	CapAttribute string `json:"capAttribute,omitempty"`
	CapPm        int    `json:"capPm,omitempty"`
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
	// Min é o piso do `per: attribute` ("Sabedoria (mínimo +1)", p50), e
	// RepeatBonus o que cada escolha REPETIDA soma ("+2", p50).
	Min         int `json:"min,omitempty"`
	RepeatBonus int `json:"repeatBonus,omitempty"`
}

type Fact struct {
	Category string `json:"category"`
	Text     string `json:"text"`
}
