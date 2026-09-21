// Package engine is the Tormenta 20 character-sheet compute engine, and the
// single authority on the book's rules: attributes, vitals (PV/PM), defense,
// saves, skills, attacks, conditions and buffs.
//
// It imports NOTHING from this project — only the standard library. That purity
// is the property worth defending: the rules answer with no database, no HTTP
// and no state, which is why 4,000 lines of rule tests run with no test double.
//
// KEY INSIGHT: only maxPv/maxPm catalog modifiers reach the sheet directly.
// Every other computed field derives from attributes + inline equipment +
// active effects.
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
func IsAttributeKey(key string) bool {
	for _, k := range AttributeKeys {
		if k == key {
			return true
		}
	}
	return false
}

// ─── Input ────────────────────────────────────────────────────────────

// CharacterInput. Unused catalog fields are
// still parsed so the HTTP contract accepts the full body.
type CharacterInput struct {
	Level              int                    `json:"level"`
	ClassName          string                 `json:"className"`
	RaceID             string                 `json:"raceId"`
	RaceFloatingPicks  []string               `json:"raceFloatingPicks"`
	RaceAncestry       string                 `json:"raceAscendencia"`
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
	Deformity          *Deformidade           `json:"deformidade"`
	ActiveConditions   []string               `json:"activeConditions"`
}

type AdditionalRace struct {
	RaceID        string   `json:"raceId"`
	FloatingPicks []string `json:"floatingPicks"`
	Ancestry      string   `json:"ascendencia"`
}

type ClassChoice struct {
	Devotee string `json:"devoto"`
	Path    string `json:"caminho"`
}

type ClassEntry struct {
	ClassName string `json:"className"`
	Level     int    `json:"level"`
}

type Deformidade struct {
	Expertises    []string `json:"pericias"`
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
