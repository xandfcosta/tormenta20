package transcript

import "sync"

// Resources são os nomes dos recursos, na ordem do índice do catálogo.
var Resources = []string{"activations", "bestiary", "class-expertises", "class-powers", "classes", "conditions", "devotee-terms", "divine-powers", "dungeon-design", "effect-types", "expertises", "general-powers", "gm-tables", "gods", "granted-powers", "items", "options", "origins-source", "origins", "race-defs", "races", "spell-schools", "spells", "tormenta-powers"}

// ResourceJSON é o recurso serializado na forma em que os leitores sempre o
// receberam, ou (nil, false) para um nome que não existe. Serializado uma vez
// por processo: o valor não muda enquanto o binário for o mesmo.
//
//	raw, ok := transcript.ResourceJSON("items")
func ResourceJSON(name string) ([]byte, bool) {
	resourcesOnce.Do(marshalResources)
	raw, ok := resourceBytes[name]
	return raw, ok
}

var (
	resourcesOnce sync.Once
	resourceBytes map[string][]byte
)

func marshalResources() {
	resourceBytes = make(map[string][]byte, len(Resources))
	put := func(name string, marshal func() ([]byte, error)) {
		raw, err := marshal()
		if err != nil {
			// Um verbete que não se serializa é binário quebrado: não há ficha certa a servir.
			panic("transcript: " + name + ": " + err.Error())
		}
		resourceBytes[name] = raw
	}
	put("activations", func() ([]byte, error) { return marshalPlain(Activations) })
	put("bestiary", func() ([]byte, error) { return marshalPlain(Bestiary) })
	put("class-expertises", ClassExpertises.MarshalJSON)
	put("class-powers", func() ([]byte, error) { return marshalPlain(ClassPowers) })
	put("classes", func() ([]byte, error) { return marshalPlain(Classes) })
	put("conditions", func() ([]byte, error) { return keyedByID(Conditions, func(v Condition) string { return v.ID }) })
	put("devotee-terms", func() ([]byte, error) { return marshalPlain(DevoteeTerms) })
	put("divine-powers", func() ([]byte, error) { return marshalPlain(DivinePowers) })
	put("dungeon-design", func() ([]byte, error) { return marshalPlain(DungeonDesign) })
	put("effect-types", func() ([]byte, error) { return marshalPlain(EffectTypes) })
	put("expertises", func() ([]byte, error) { return marshalPlain(Expertises) })
	put("general-powers", func() ([]byte, error) { return marshalPlain(GeneralPowers) })
	put("gm-tables", func() ([]byte, error) { return marshalPlain(GMTables) })
	put("gods", func() ([]byte, error) { return marshalPlain(Gods) })
	put("granted-powers", func() ([]byte, error) { return marshalPlain(GrantedPowers) })
	put("items", func() ([]byte, error) { return marshalPlain(Items) })
	put("options", func() ([]byte, error) { return marshalPlain(CreationOptions) })
	put("origins-source", func() ([]byte, error) { return keyedByID(OriginSources, func(v OriginSource) string { return v.ID }) })
	put("origins", func() ([]byte, error) { return marshalPlain(Origins) })
	put("race-defs", func() ([]byte, error) { return marshalPlain(RaceDefinitions) })
	put("races", func() ([]byte, error) { return keyedByID(Races, func(v Race) string { return v.ID }) })
	put("spell-schools", func() ([]byte, error) { return marshalPlain(SpellSchools) })
	put("spells", func() ([]byte, error) { return keyedByID(Spells, func(v Spell) string { return v.ID }) })
	put("tormenta-powers", func() ([]byte, error) { return keyedByID(TormentaPowers, func(v TormentaPower) string { return v.ID }) })
}
