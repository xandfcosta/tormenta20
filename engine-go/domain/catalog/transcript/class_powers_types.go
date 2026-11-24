package transcript

type ClassPower struct {
	ID             string     `json:"id"`
	Modeled        []string   `json:"modeled,omitempty"`
	ClassName      string     `json:"className,omitempty"`
	Name           string     `json:"name,omitempty"`
	Description    string     `json:"description,omitempty"`
	GrantedAtLevel int        `json:"grantedAtLevel,omitempty"`
	BookPage       int        `json:"bookPage,omitempty"`
	UID            string     `json:"uid"`
	PowerUID       string     `json:"powerUid,omitempty"`
	Modifiers      []Modifier `json:"modifiers,omitzero"`
	Unmodeled      string     `json:"unmodeled,omitempty"`
	// ProficientWith é a proficiência que o poder dá e que o motor calcula — ver `engine.ProficiencyOf`.
	ProficientWith  []ProficiencyReach `json:"proficientWith,omitempty"`
	Prerequisites   []Prerequisite     `json:"prerequisites,omitempty"`
	Choice          *PowerChoice       `json:"choice,omitempty"`
	MinLevel        int                `json:"minLevel,omitempty"`
	GrantedByChoice *ChoiceGrant       `json:"grantedByChoice,omitempty"`
	Degrees         *PMScaling         `json:"degrees,omitempty"`
	Boost           *ClassPowerBoost   `json:"boost,omitempty"`
	// ExtraStrike é o ataque pago que não gasta a vez — ver `engine.PaidExtraStrike`.
	ExtraStrike *ClassPowerExtraStrike `json:"extraStrike,omitempty"`
	// BoostAddOn é o acréscimo pago num impulso de outro poder — ver `engine.ClassPowerBoostAddOn`.
	BoostAddOn *ClassPowerBoostAddOn `json:"boostAddOn,omitempty"`
	// FavoredWeaponDie é o dado do impulso com a arma preferida — ver `engine.ClassPowerFavoredWeaponDie`.
	FavoredWeaponDie *ClassPowerFavoredWeaponDie `json:"favoredWeaponDie,omitempty"`
	// FollowUp é a manobra livre depois de outro gesto declarado — ver `engine.FollowUpManeuver`.
	FollowUp *ClassPowerFollowUp `json:"followUp,omitempty"`
	// UnarmedDamageVariant é a linha do dano desarmado numa situação — ver `engine.DamageVariant`.
	UnarmedDamageVariant *ClassPowerUnarmedDamageVariant `json:"unarmedDamageVariant,omitempty"`
	// SpellAugment é o preço e o efeito do poder que se paga ao conjurar, como os gerais da p131 — ver
	// `book.PowerAugments`.
	SpellAugment *SpellAugment `json:"spellAugment,omitempty"`
}

// ClassPowerUnarmedDamageVariant é a linha que o poder dá à carta desarmada: a situação e quantos passos o dado
// sobe nela (a Arma Improvisada e a Chave, p76).
type ClassPowerUnarmedDamageVariant struct {
	Label string `json:"label"`
	Steps int    `json:"steps"`
}

// ClassPowerFollowUp é a manobra em seguida que o poder dá: a derrubada depois do Sucesso no agarrar (Lutador de
// Chão, p77), o agarrar depois do acerto com o Ataque Furtivo ligado (Mão na Boca, p74).
type ClassPowerFollowUp struct {
	Maneuver  string `json:"maneuver"`
	After     string `json:"after,omitempty"`
	On        string `json:"on"`
	PM        int    `json:"pm,omitempty"`
	NeedsFlag string `json:"needsFlag,omitempty"`
}

// ClassPowerExtraStrike é o ataque adicional que o poder vende ao agredir, por PM: o Ataque Extra (p66) com
// qualquer arma, o Golpe Relâmpago (p76) só desarmado.
type ClassPowerExtraStrike struct {
	PM      int  `json:"pm"`
	Unarmed bool `json:"unarmed,omitempty"`
}

// NaturalWeapon é a arma natural que uma habilidade ou poder dá (quadro de Ataques Desarmados
// e Armas Naturais). ExtraPM é o que custa o ataque extra ao agredir com outra arma, e
// ExtraAttacks diz quantos esse pagamento compra, quando é mais de um (Membros Extras, p137).
// AwakenedByUse diz que ela só existe depois de usar o poder, até o fim da cena, e
// StepsOnSameWeapon, que quem já tem arma natural desse nome soma os passos nela, em vez de
// ganhar outra (p135).
type NaturalWeapon struct {
	Name              string `json:"name"`
	Damage            string `json:"damage"`
	CritMult          int    `json:"critMult"`
	DamageType        string `json:"damageType"`
	ExtraPM           int    `json:"extraPm"`
	ExtraAttacks      int    `json:"extraAttacks,omitempty"`
	AwakenedByUse     bool   `json:"awakenedByUse,omitempty"`
	StepsOnSameWeapon int    `json:"stepsOnSameWeapon,omitempty"`
}

type PowerChoice struct {
	Kind                 string              `json:"kind"`
	Label                string              `json:"label"`
	Repeatable           bool                `json:"repeatable,omitempty"`
	Options              []PowerChoiceOption `json:"options,omitempty"`
	GrantsSpellAttribute string              `json:"grantsSpellAttribute,omitempty"`
	CountAttribute       string              `json:"countAttribute,omitempty"`
	Count                int                 `json:"count,omitempty"`
	// CountLevels são os níveis da classe em que se ganha uma escolha — o
	// Explorador, no 3º e a cada quatro (p50).
	CountLevels []int    `json:"countLevels,omitempty"`
	SwapFor     string   `json:"swapFor,omitempty"`
	Exclude     []string `json:"exclude,omitempty"`
	// RequiresProficiency diz que a arma escolhida pede proficiência — o Foco
	// em Arma (p128), e não a Especialização em Arma (p65).
	RequiresProficiency bool `json:"requiresProficiency,omitempty"`
	// AmongPicksOf é o poder cujas escolhas limitam esta: o Mestre em Escola escolhe uma escola que o
	// Especialista em Escola já escolheu (p38).
	AmongPicksOf string `json:"amongPicksOf,omitempty"`
}

type PowerChoiceOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Note string `json:"note,omitempty"`
	// MinLevel é o nível da classe a partir do qual a opção vale — a área de
	// Tormenta do Explorador, no 11º (p50).
	MinLevel int `json:"minLevel,omitempty"`
}

type ChoiceGrant struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

// ClassPowerBoost é o que o poder vende NO ATAQUE, por PM (ALE-487): o preço e
// o teto em `degrees`, o que cada degrau soma, e as travas do livro.
type ClassPowerBoost struct {
	Degrees   PMScaling  `json:"degrees"`
	Modifiers []Modifier `json:"modifiers"`
	MeleeOnly bool       `json:"meleeOnly,omitempty"`
	Split     bool       `json:"split,omitempty"`
}

// ClassPowerBoostAddOn é o acréscimo pago num impulso de outro poder: o Romper Resistências põe +1 PM no Ataque
// Especial para ignorar 10 de RD (p66).
type ClassPowerBoostAddOn struct {
	Boost     string     `json:"boost"`
	PM        int        `json:"pm"`
	Modifiers []Modifier `json:"modifiers"`
}

// ClassPowerFavoredWeaponDie é o dado que um poder põe no impulso de outro quando o golpe é com a arma preferida
// da divindade: a Arma Sagrada leva o Golpe Divino a d12 (p82).
type ClassPowerFavoredWeaponDie struct {
	Boost string `json:"boost"`
	Faces int    `json:"faces"`
}

// HasEngineEffect diz se o motor aplica alguma coisa deste poder: um modificador, ou um campo que o motor lê sem
// modificador — o ataque pago, o aprimoramento, o que se soma a ele, o dado da arma favorita, a escala de PM, a
// proficiência, a manobra em seguida, a linha do dano desarmado e o que se paga ao conjurar. É o que o auditor de cobertura conta como efeito, e um campo novo do tipo
// tem de decidir de que lado fica (`TestEveryClassPowerFieldSaysWhetherItIsAnEngineEffect`).
//
//	if power.HasEngineEffect() { … }
func (p ClassPower) HasEngineEffect() bool {
	return len(p.Modifiers) > 0 || len(p.ProficientWith) > 0 || p.Degrees != nil || p.Boost != nil ||
		p.ExtraStrike != nil || p.BoostAddOn != nil || p.FavoredWeaponDie != nil || p.FollowUp != nil ||
		p.UnarmedDamageVariant != nil || p.SpellAugment != nil
}
