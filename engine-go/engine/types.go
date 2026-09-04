// Package engine is a Go port of the Tormenta 20 character-sheet compute
// engine (t20-data/src/character-sheet.ts). It reproduces the derived-sheet
// pipeline: attributes, vitals (PV/PM), defense, saves, skills, attacks,
// conditions and buffs. Only maxPv/maxPm catalog modifiers reach the sheet —
// every other computed field derives from attributes + inline equipment +
// active effects (see character-sheet.ts "KEY INSIGHT").
package engine

// AttributeKeys is the canonical attribute order (attributes.ts).
var AttributeKeys = []string{
	"strength", "dexterity", "constitution", "intelligence", "wisdom", "charisma",
}

// IsAttributeKey diz se a chave é um dos seis atributos.
//
// Ela nasceu na ALE-278 apagando uma SEGUNDA transcrição dos mesmos seis nomes
// — o `engine.IsAttributeKey` do `api`, que a ficha em Datastar também lia. A lista e
// a pergunta sobre a lista moram juntas de propósito: é a lição do `Fold`
// copiado para o `book`, onde a cópia compilava, tinha o nome certo e fazia
// outra coisa.
func IsAttributeKey(chave string) bool {
	for _, k := range AttributeKeys {
		if k == chave {
			return true
		}
	}
	return false
}

// ─── Input ────────────────────────────────────────────────────────────

// CharacterInput mirrors t20-data CharacterInput. Unused catalog fields are
// still parsed so the HTTP contract accepts the full body.
type CharacterInput struct {
	Level              int                    `json:"level"`
	ClassName          string                 `json:"className"`
	RaceID             string                 `json:"raceId"`
	RaceFloatingPicks  []string               `json:"raceFloatingPicks"`
	RaceAscendencia    string                 `json:"raceAscendencia"`
	AdditionalRaces    []AdditionalRace       `json:"additionalRaces"`
	BaseAttributes     map[string]int         `json:"baseAttributes"`
	CurrentPv          *int                   `json:"currentPv"`
	CurrentPm          *int                   `json:"currentPm"`
	TrainedSkills      []string               `json:"trainedSkills"`
	ArmorPenalty       *int                   `json:"armorPenalty"`
	Equipment          *CharacterEquipment    `json:"equipment"`
	ActiveEffects      []ActiveEffect         `json:"activeEffects"`
	RaceAbilityChoices []string               `json:"raceAbilityChoices"`
	PowerIDs           []string               `json:"powerIds"`
	ClassChoices       map[string]ClassChoice `json:"classChoices"`
	Classes            []ClassEntry           `json:"classes"`
	GodPower           string                 `json:"godPower"`
	Origin             string                 `json:"origin"`
	OriginChoices      []string               `json:"originChoices"`
	Deformidade        *Deformidade           `json:"deformidade"`
	ActiveConditions   []string               `json:"activeConditions"`
}

type AdditionalRace struct {
	RaceID        string   `json:"raceId"`
	FloatingPicks []string `json:"floatingPicks"`
	Ascendencia   string   `json:"ascendencia"`
}

type ClassChoice struct {
	Devoto  string `json:"devoto"`
	Caminho string `json:"caminho"`
}

type ClassEntry struct {
	ClassName string `json:"className"`
	Level     int    `json:"level"`
}

type Deformidade struct {
	Pericias      []string `json:"pericias"`
	TormentaPower string   `json:"tormentaPower"`
}

type CharacterEquipment struct {
	Armor    *EquippedArmor  `json:"armor"`
	Shield   *EquippedShield `json:"shield"`
	MainHand *EquippedWeapon `json:"mainHand"`
	OffHand  *EquippedWeapon `json:"offHand"`
}

type EquippedArmor struct {
	Name    string `json:"name"`
	Defense int    `json:"defense"`
	Penalty int    `json:"penalty"`
	Heavy   bool   `json:"heavy"`
}

type EquippedShield struct {
	Name    string `json:"name"`
	Defense int    `json:"defense"`
	Penalty int    `json:"penalty"`
	Heavy   bool   `json:"heavy"`
}

type EquippedWeapon struct {
	Name       string `json:"name"`
	Hand       string `json:"hand"`
	Purpose    string `json:"purpose"`
	Damage     string `json:"damage"`
	CritRange  int    `json:"critRange"`
	CritMult   int    `json:"critMult"`
	DamageType string `json:"damageType"`
}

// ActiveEffect + EffectModifier + EffectTarget mirror the buff pipeline.
type ActiveEffect struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Source    string           `json:"source"`
	Modifiers []EffectModifier `json:"modifiers"`
}

type EffectModifier struct {
	Target EffectTarget `json:"target"`
	Amount int          `json:"amount"`
}

type EffectTarget struct {
	K         string `json:"k"`
	Attribute string `json:"attribute"`
	Save      string `json:"save"`
	Skill     string `json:"skill"`
}

// ─── Output ───────────────────────────────────────────────────────────

type AttributeComputed struct {
	Base        int  `json:"base"`
	RaceMod     int  `json:"raceMod"`
	TormentaMod *int `json:"tormentaMod,omitempty"`
	Total       int  `json:"total"`
}

type Vitals struct {
	PvMax     int `json:"pvMax"`
	PmMax     int `json:"pmMax"`
	PvCurrent int `json:"pvCurrent"`
	PmCurrent int `json:"pmCurrent"`
}

type Defense struct {
	Base      int `json:"base"`
	Attribute int `json:"attribute"`
	Armor     int `json:"armor"`
	Shield    int `json:"shield"`
	Total     int `json:"total"`
}

type Saves struct {
	Fortitude int `json:"fortitude"`
	Reflexos  int `json:"reflexos"`
	Vontade   int `json:"vontade"`
}

type SkillComputed struct {
	Total               int    `json:"total"`
	Trained             bool   `json:"trained"`
	KeyAttribute        string `json:"keyAttribute"`
	CannotUse           bool   `json:"cannotUse"`
	ArmorPenaltyApplied int    `json:"armorPenaltyApplied"`
}

type ComputedAttack struct {
	WeaponName           string `json:"weaponName,omitempty"`
	Skill                string `json:"skill"`
	AttackTotal          int    `json:"attackTotal"`
	DamageDice           string `json:"damageDice"`
	DamageAttributeBonus int    `json:"damageAttributeBonus"`
	DamageType           string `json:"damageType"`
	CritRange            int    `json:"critRange"`
	CritMult             int    `json:"critMult"`
	Hand                 string `json:"hand"`
	Purpose              string `json:"purpose"`
}

type Attacks struct {
	MainHand *ComputedAttack `json:"mainHand"`
	OffHand  *ComputedAttack `json:"offHand"`
}

type ConditionSummary struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type BuffContribution struct {
	EffectID   string `json:"effectId"`
	EffectName string `json:"effectName"`
	Source     string `json:"source"`
	TargetKey  string `json:"targetKey"`
	Amount     int    `json:"amount"`
	Applied    bool   `json:"applied"`
}

type BuffsSummary struct {
	Totals        map[string]int     `json:"totals"`
	Contributions []BuffContribution `json:"contributions"`
}

type ComputedSheet struct {
	Level        int                          `json:"level"`
	ClassName    string                       `json:"className"`
	Attributes   map[string]AttributeComputed `json:"attributes"`
	Vitals       Vitals                       `json:"vitals"`
	Defense      Defense                      `json:"defense"`
	Saves        Saves                        `json:"saves"`
	Skills       map[string]SkillComputed     `json:"skills"`
	Attacks      Attacks                      `json:"attacks"`
	Conditions   []ConditionSummary           `json:"conditions"`
	Buffs        BuffsSummary                 `json:"buffs"`
	Deslocamento int                          `json:"deslocamento"`
	Tamanho      string                       `json:"tamanho"`
	Warnings     []string                     `json:"warnings"`
}
