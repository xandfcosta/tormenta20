package transcript

type Item struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	Category     string           `json:"category"`
	Price        float64          `json:"price,omitempty"`
	Slots        float64          `json:"slots,omitempty"`
	Equip        string           `json:"equip,omitempty"`
	Hands        int              `json:"hands,omitempty"`
	Weapon       *WeaponStats     `json:"weapon,omitempty"`
	Modifiers    []Modifier       `json:"modifiers"`
	BookPage     int              `json:"bookPage,omitempty"`
	UID          string           `json:"uid"`
	Armor        *ArmorStats      `json:"armor,omitempty"`
	Shield       *ArmorStats      `json:"shield,omitempty"`
	Consumable   *Consumable      `json:"consumable,omitempty"`
	AppliesTo    []string         `json:"appliesTo,omitempty"`
	PriceByItem  *PriceByItem     `json:"priceByItem,omitempty"`
	Description  string           `json:"description,omitempty"`
	Modeled      []string         `json:"modeled,omitempty"`
	Unmodeled    *ItemUnmodeled   `json:"unmodeled,omitempty"`
	RollMin      int              `json:"rollMin,omitempty"`
	RollMax      int              `json:"rollMax,omitempty"`
	CountsAs     int              `json:"countsAs,omitempty"`
	Requires     string           `json:"requires,omitempty"`
	ComposedOf   *ItemComposition `json:"composedOf,omitempty"`
	Artifact     bool             `json:"artifact,omitempty"`
	SkillTies    []SkillTie       `json:"skillTies,omitempty"`
	PowerChanges []PowerChange    `json:"powerChanges,omitempty"`
	Boost        *ItemBoost       `json:"boost,omitempty"`
}

// ItemBoost é o gasto de PM no golpe que o ENCANTO oferece: "Uma vez por rodada, quando ataca, você pode
// gastar 2 PM. Se fizer isso…" (p335-336). Os Modifiers só valem no golpe pago — o dado a mais, a condição
// ou o efeito que a mesa aplica no acerto (`announcedEffect`, com a fórmula escrita). HitPointCost é o
// dado de PV que o próprio portador perde ao gastar (a Tumular).
type ItemBoost struct {
	PM           int        `json:"pm"`
	Modifiers    []Modifier `json:"modifiers"`
	HitPointCost string     `json:"hitPointCost,omitempty"`
	// AgainstOneOf são as opções da escolha que o mestre faz ao encantar: o gasto só vale contra o tipo de
	// criatura escolhido (a Anticriatura, p335). `raca-humanoide` aceita o id de qualquer raça.
	AgainstOneOf []string `json:"againstOneOf,omitempty"`
}

// PowerChange é o que o item EMPUNHADO muda em poderes de quem o empunha (p158). Powers são ids de
// ativação; Action é a ação que eles passam a custar, e Note o aviso que a linha deles ganha quando a
// mudança não é número que o motor guarde (o alcance).
type PowerChange struct {
	Powers []string `json:"powers"`
	Action string   `json:"action,omitempty"`
	Note   string   `json:"note,omitempty"`
}

// SkillTie é a perícia a que uma ferramenta ou um vestuário está ligado sem somar número: ele tira uma
// penalidade (p158-159). When é a situação em que a penalidade vale, quando o livro a limita.
type SkillTie struct {
	Skill string `json:"skill"`
	When  string `json:"when,omitempty"`
}

type WeaponStats struct {
	Damage    string `json:"damage"`
	CritRange int    `json:"critRange"`
	CritMult  int    `json:"critMult"`
	Type      string `json:"type"`
	Hand      string `json:"hand"`
	Purpose   string `json:"purpose"`
	Range     string `json:"range,omitempty"`
	// MeleeWithPenalty é a arma de arremesso que também bate, com −5: a azagaia (p146).
	MeleeWithPenalty bool     `json:"meleeWithPenalty,omitempty"`
	Traits           []string `json:"traits"`
	Finesse          bool     `json:"finesse,omitempty"`
}

type ArmorStats struct {
	Defense int  `json:"defense"`
	Penalty int  `json:"penalty"`
	Heavy   bool `json:"heavy"`
}

type Consumable struct {
	Scope      string           `json:"scope"`
	Instant    *InstantRecovery `json:"instant,omitempty"`
	Modifiers  []Modifier       `json:"modifiers,omitzero"`
	OncePerDay bool             `json:"oncePerDay,omitempty"`
	Spell      string           `json:"spell,omitempty"`
	Kind       string           `json:"kind,omitempty"`
	Augment    string           `json:"augment,omitempty"`
	Dose       *DoseEffect      `json:"dose,omitempty"`
}

// DoseEffect é o que o item faz quando a Injeção alquímica o libera (p165), e a
// presença dele é o que diz que o item carrega. O preparado que não fere (o bálsamo,
// o cosmético) não tem: decisão do dono.
type DoseEffect struct {
	Effect string `json:"effect"`
	// Save é o teste que o item pede, SaveAttribute o atributo da CD (p227) e OnSave o
	// que passar nele faz.
	Save          string `json:"save"`
	SaveAttribute string `json:"saveAttribute"`
	OnSave        string `json:"onSave"`
}

type InstantRecovery struct {
	HP *RecoveryDice `json:"hp,omitempty"`
	MP *RecoveryDice `json:"mp,omitempty"`
}

type RecoveryDice struct {
	Dice  string `json:"dice"`
	Bonus int    `json:"bonus,omitempty"`
}

type PriceByItem struct {
	Weapon     int `json:"weapon"`
	ArmorLight int `json:"armorLight,omitempty"`
	ArmorHeavy int `json:"armorHeavy,omitempty"`
	Shield     int `json:"shield"`
	Esoteric   int `json:"esoteric"`
}

type ItemComposition struct {
	Base         string   `json:"base"`
	Material     string   `json:"material,omitempty"`
	Enchants     []string `json:"enchants,omitempty"`
	Improvements []string `json:"improvements,omitempty"`
}
