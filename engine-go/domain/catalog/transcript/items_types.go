package transcript

type Item struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Category    string           `json:"category"`
	Price       float64          `json:"price,omitempty"`
	Slots       float64          `json:"slots,omitempty"`
	Equip       string           `json:"equip,omitempty"`
	Hands       int              `json:"hands,omitempty"`
	Weapon      *WeaponStats     `json:"weapon,omitempty"`
	Modifiers   []Modifier       `json:"modifiers"`
	BookPage    int              `json:"bookPage,omitempty"`
	UID         string           `json:"uid"`
	Armor       *ArmorStats      `json:"armor,omitempty"`
	Shield      *ArmorStats      `json:"shield,omitempty"`
	Consumable  *Consumable      `json:"consumable,omitempty"`
	AppliesTo   []string         `json:"appliesTo,omitempty"`
	PriceByItem *PriceByItem     `json:"priceByItem,omitempty"`
	Description string           `json:"description,omitempty"`
	Modeled     []string         `json:"modeled,omitempty"`
	Unmodeled   *ItemUnmodeled   `json:"unmodeled,omitempty"`
	RollMin     int              `json:"rollMin,omitempty"`
	RollMax     int              `json:"rollMax,omitempty"`
	CountsAs    int              `json:"countsAs,omitempty"`
	Requires    string           `json:"requires,omitempty"`
	ComposedOf  *ItemComposition `json:"composedOf,omitempty"`
	Artifact    bool             `json:"artifact,omitempty"`
}

type WeaponStats struct {
	Damage    string   `json:"damage"`
	CritRange int      `json:"critRange"`
	CritMult  int      `json:"critMult"`
	Type      string   `json:"type"`
	Hand      string   `json:"hand"`
	Purpose   string   `json:"purpose"`
	Range     string   `json:"range,omitempty"`
	Traits    []string `json:"traits"`
	Finesse   bool     `json:"finesse,omitempty"`
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
