package engine

// Character is the Go mirror of the frontend `Character` (shared/api/api.ts) —
// the RAW persisted sheet the collection layer (ActiveItemsFor) reads, distinct
// from the flattened CharacterInput the MVP engine consumes. Only the fields the
// collection layer touches are typed; the many JSON-string columns
// (proficiencies, classPowers, raceAttributeChoices…) stay as strings and are
// parsed on demand, exactly as derived.ts does.
type Character struct {
	ID           int    `json:"id"`
	Origin       string `json:"origin"`
	Level        int    `json:"level"`
	Strength     int    `json:"strength"`
	Dexterity    int    `json:"dexterity"`
	Constitution int    `json:"constitution"`
	Intelligence int    `json:"intelligence"`
	Wisdom       int    `json:"wisdom"`
	Charisma     int    `json:"charisma"`

	// JSON-encoded columns (parsed lazily, mirroring derived.ts parse helpers).
	Proficiencies        string `json:"proficiencies"`
	RaceAbilityChoices   string `json:"raceAbilityChoices"`
	RaceAttributeChoices string `json:"raceAttributeChoices"`
	SecondaryRaceChoices string `json:"secondaryRaceChoices"`
	OriginChoices        string `json:"originChoices"`
	ClassPowers          string `json:"classPowers"`
	ClassChoices         string `json:"classChoices"`
	PowerChoices         string `json:"powerChoices"`

	Races         []CharacterRace   `json:"races"`
	Classes       []CharacterClass  `json:"classes"`
	Items         []CharacterItem   `json:"items"`
	ActiveEffects []ActiveEffectRow `json:"activeEffects"`
}

type CharacterRace struct {
	Race string `json:"race"`
}

type CharacterClass struct {
	ClassName string `json:"className"`
	Level     int    `json:"level"`
}

// CharacterItem mirrors the api.ts CharacterItem. Equipped is a pointer so the
// null (unequipped) state is distinguishable from a wear slot.
type CharacterItem struct {
	CatalogID    *string `json:"catalogId"`
	Name         string  `json:"name"`
	Equipped     *string `json:"equipped"`
	Improvements string  `json:"improvements"`
	Material     *string `json:"material"`
}

// ActiveEffectRow mirrors the api.ts ActiveEffect: a consumed scene/day buff
// carrying a JSON-encoded Modifier[] copied from the catalog at consume time.
type ActiveEffectRow struct {
	CatalogID string `json:"catalogId"`
	Scope     string `json:"scope"` // 'scene' | 'day'
	Modifiers string `json:"modifiers"`
}

// attributeValue reads a raw base attribute by AttributeKey. Mirrors the
// `character[attr]` index derived.ts uses.
func (c Character) attributeValue(attr string) int {
	switch attr {
	case "strength":
		return c.Strength
	case "dexterity":
		return c.Dexterity
	case "constitution":
		return c.Constitution
	case "intelligence":
		return c.Intelligence
	case "wisdom":
		return c.Wisdom
	case "charisma":
		return c.Charisma
	}
	return 0
}
