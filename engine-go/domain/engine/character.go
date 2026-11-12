package engine

// PowerUse é o contador de usos de um poder, numa janela de tempo.
type PowerUse struct {
	PowerID string `json:"powerId"`
	Scope   string `json:"scope"`
	Used    int    `json:"used"`
}

// Character is the RAW persisted sheet the collection layer (ActiveItemsFor)
// reads, distinct from the flattened CharacterInput the MVP engine consumes.
// Only the fields the collection layer touches are typed; the many JSON-string
// columns (proficiencies, classPowers, raceAttributeChoices…) stay as strings
// and are parsed on demand.
type Character struct {
	ID        int    `json:"id"`
	Origin    string `json:"origin"`
	GodPowers string `json:"godPowers"`
	// God é a devoção, que uma regra da mesa pode olhar.
	God *string `json:"god"`
	// PowerUses são os usos contados da ficha. O motor só lê os de cena, porque há poder que
	// só vale depois de usado (as Presas Primordiais, p135).
	PowerUses []PowerUse `json:"powerUses"`
	// WildShape é a Forma Selvagem em que o druida está, ou nil (p62-63).
	WildShape    *WildShape `json:"wildShape,omitempty"`
	Level        int        `json:"level"`
	Strength     int        `json:"strength"`
	Dexterity    int        `json:"dexterity"`
	Constitution int        `json:"constitution"`
	Intelligence int        `json:"intelligence"`
	Wisdom       int        `json:"wisdom"`
	Charisma     int        `json:"charisma"`

	// Tibar é o dinheiro do personagem, e ele é CARGA: cada mil moedas ocupam um
	// espaço (p141). Por isso vive no motor e não só na tela.
	Tibar float64 `json:"tibar"`

	// JSON-encoded columns, parsed lazily by the collect_parse.go helpers.
	Proficiencies        string `json:"proficiencies"`
	RaceAbilityChoices   string `json:"raceAbilityChoices"`
	RaceAttributeChoices string `json:"raceAttributeChoices"`
	SecondaryRaceChoices string `json:"secondaryRaceChoices"`
	OriginChoices        string `json:"originChoices"`
	ClassPowers          string `json:"classPowers"`
	ClassChoices         string `json:"classChoices"`
	PowerChoices         string `json:"powerChoices"`
	ActiveConditions     string `json:"activeConditions"` // JSON ConditionId[] — p394 status conditions

	Displacement int `json:"displacement"`

	// IgnoredRules são as regras opcionais que a campanha desligou.
	// Elas não pertencem ao personagem: pertencem à MESA, e chegam aqui porque o
	// motor é uma função pura da entrada dele. Quem resolve a qual campanha a
	// ficha responde é o servidor, no `loadCharacter`. Valor zero = tudo em
	// vigor, que é o padrão do livro — ver `IgnoredRules`.
	IgnoredRules IgnoredRules `json:"ignoredRules"`

	Races         []CharacterRace      `json:"races"`
	Classes       []CharacterClass     `json:"classes"`
	Items         []CharacterItem      `json:"items"`
	ActiveEffects []ActiveEffectRow    `json:"activeEffects"`
	Expertises    []CharacterExpertise `json:"expertises"`
}

// CharacterExpertise is the per-perícia row the expertise breakdown reads
// (name + key attribute + trained flag).
type CharacterExpertise struct {
	Name      string `json:"name"`
	Attribute string `json:"attribute"`
	Trained   bool   `json:"trained"`
}

type CharacterRace struct {
	Race string `json:"race"`
}

type CharacterClass struct {
	ClassName string `json:"className"`
	Level     int    `json:"level"`
}

// CharacterItem is one row of the character's bag. Equipped is a pointer so the
// null (unequipped) state is distinguishable from a wear slot.
type CharacterItem struct {
	// ID é a linha da mochila, e o golpe a leva para saber de que arma a dose sai.
	ID        int64   `json:"id,omitempty"`
	CatalogID *string `json:"catalogId"`
	Name      string  `json:"name"`
	// Quantity e Slots são o que a linha PESA na mochila (p141). Quem os lê não
	// é a coleção de efeitos, é a CARGA.
	Quantity     int     `json:"quantity"`
	Slots        float64 `json:"slots"`
	Equipped     *string `json:"equipped"`
	Improvements string  `json:"improvements"`
	// Enchants são os encantos da arma, por id (p335). Eles são SOBREPOSIÇÃO
	// como a melhoria e o material — o `ownItemMods` colhe os modificadores dos
	// três pelo mesmo vetor —, e ficam num campo próprio porque o livro os conta
	// e os limita em separado: "quatro melhorias e três encantos" (p334).
	Enchants []string `json:"enchants,omitempty"`
	// EnchantChoices é o que o mestre escolheu ao encantar, por encanto: o tipo de criatura da
	// Anticriatura (p335). Só o encanto que pede escolha tem linha aqui.
	EnchantChoices map[string]string `json:"enchantChoices,omitempty"`
	// Choice é a escolha que o próprio item pede a quem o tem: a escola do tomo hermético (p160). Ela
	// preenche o modificador do verbete com `FromChoice: school` (`ownItemMods`).
	Choice   string  `json:"choice,omitempty"`
	Material *string `json:"material"`
	// Oversized é a arma feita para uma categoria de tamanho maior (p106). O catálogo
	// não dá tamanho por item, então é a mochila que guarda a marca.
	Oversized bool `json:"oversized,omitempty"`
	// Doses são o que a Injeção alquímica guarda, por id, da mais antiga à mais nova
	// (p165).
	Doses []string `json:"doses,omitempty"`
}

// ActiveEffectRow is a consumed scene/day buff carrying a JSON-encoded
// Modifier[] copied from the catalog at consume time.
type ActiveEffectRow struct {
	CatalogID string `json:"catalogId"`
	Scope     string `json:"scope"` // 'scene' | 'day'
	Modifiers string `json:"modifiers"`
}

// attributeValue reads a raw base attribute by AttributeKey.
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
