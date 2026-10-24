package transcript

type Creature struct {
	ID               string           `json:"id"`
	Name             string           `json:"name"`
	ND               float64          `json:"nd"`
	Type             string           `json:"tipo"`
	Size             string           `json:"size"`
	HP               int              `json:"hp"`
	Defense          int              `json:"defesa"`
	Strength         *int             `json:"forca"`
	Dexterity        int              `json:"destreza"`
	Constitution     int              `json:"constituicao"`
	Intelligence     *int             `json:"inteligencia"`
	Wisdom           int              `json:"sabedoria"`
	Charisma         int              `json:"carisma"`
	Fortitude        int              `json:"fortitude"`
	Reflexes         int              `json:"reflexos"`
	Will             int              `json:"vontade"`
	Displacement     string           `json:"deslocamento"`
	Attacks          []CreatureAttack `json:"attacks"`
	SpecialAbilities []string         `json:"specialAbilities"`
	BookPage         int              `json:"bookPage"`
	Skills           []CreatureSkill  `json:"skills"`
	Equipment        string           `json:"equipamento"`
	Initiative       int              `json:"iniciativa"`
	Perception       int              `json:"percepcao"`
	Treasure         string           `json:"tesouro"`
	UID              string           `json:"uid"`
	PM               int              `json:"pm,omitempty"`
}

type CreatureAttack struct {
	Name        string `json:"name"`
	AttackBonus int    `json:"attackBonus"`
	Damage      string `json:"damage"`
	Special     string `json:"special,omitempty"`
}

type CreatureSkill struct {
	Name  string `json:"name"`
	Bonus int    `json:"bonus"`
	Note  string `json:"nota,omitempty"`
}
