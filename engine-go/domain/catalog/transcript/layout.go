package transcript

import "sync"

// layouts é o REGISTRO dos recursos do livro transcrito: o nome com que o
// catálogo os serve, a variável, o arquivo, a frase do comentário e as partes.
// É a única lista deles — o `ResourceJSON` e o emissor leem daqui.
var layouts = []resourceLayout{
	listLayout("activations", "Activations", "activations", "o que cada poder CUSTA para ser usado — ação, PM, usos, postura", &Activations, []*[]Activation{&activationsPart01, &activationsPart02, &activationsPart03, &activationsPart04, &activationsPart05, &activationsPart06, &activationsPart07, &activationsPart08, &activationsPart09, &activationsPart10}, nil),
	listLayout("bestiary", "Bestiary", "bestiary", "as criaturas do Capítulo 7 (Tabela 7-1)", &Bestiary, []*[]Creature{&bestiaryPart01, &bestiaryPart02, &bestiaryPart03, &bestiaryPart04, &bestiaryPart05, &bestiaryPart06, &bestiaryPart07}, nil),
	valueLayout("class-expertises", "ClassExpertises", "class_expertises", "as perícias de cada classe: as fixas e as que se escolhem", &ClassExpertises, ClassExpertises.MarshalJSON),
	listLayout("class-powers", "ClassPowers", "class_powers", "os poderes e habilidades de classe do Capítulo 1", &ClassPowers, []*[]ClassPower{&classPowersPart01, &classPowersPart02, &classPowersPart03, &classPowersPart04, &classPowersPart05, &classPowersPart06, &classPowersPart07, &classPowersPart08, &classPowersPart09, &classPowersPart10, &classPowersPart11, &classPowersPart12, &classPowersPart13}, nil),
	listLayout("classes", "Classes", "classes", "as 14 classes, com proficiências e a regra de conjuração", &Classes, nil, nil),
	listLayout("conditions", "Conditions", "conditions", "as condições do apêndice", &Conditions, nil, func(v Condition) string { return v.ID }),
	valueLayout("devotee-terms", "DevoteeTerms", "devotee_terms", "os termos com que os deuses nomeiam quem pode ser devoto", &DevoteeTerms, func() ([]byte, error) { return marshalPlain(DevoteeTerms) }),
	listLayout("divine-powers", "DivinePowers", "divine_powers", "os poderes concedidos, um por par (deus, poder)", &DivinePowers, []*[]DivinePower{&divinePowersPart01, &divinePowersPart02}, nil),
	valueLayout("dungeon-design", "DungeonDesign", "dungeon_design", "as tabelas de desenho de masmorra do Capítulo 6", &DungeonDesign, func() ([]byte, error) { return marshalPlain(DungeonDesign) }),
	listLayout("effect-types", "EffectTypes", "effect_types", "os tipos de efeito (p228)", &EffectTypes, nil, nil),
	listLayout("expertises", "Expertises", "expertises", "as perícias da Tabela 2-1", &Expertises, nil, nil),
	listLayout("general-powers", "GeneralPowers", "general_powers", "os poderes gerais do Capítulo 2", &GeneralPowers, []*[]GeneralPower{&generalPowersPart01, &generalPowersPart02}, nil),
	valueLayout("gm-tables", "GMTables", "gm_tables", "as tabelas de improviso do mestre (Capítulo 6)", &GMTables, func() ([]byte, error) { return marshalPlain(GMTables) }),
	listLayout("gods", "Gods", "gods", "o Panteão: os 20 deuses", &Gods, nil, nil),
	listLayout("granted-powers", "GrantedPowers", "granted_powers", "os poderes concedidos que o motor sabe somar", &GrantedPowers, []*[]GrantedPower{&grantedPowersPart01, &grantedPowersPart02}, nil),
	listLayout("items", "Items", "items", "o equipamento do Capítulo 3 e os itens mágicos do Capítulo 8", &Items, []*[]Item{&itemsPart01, &itemsPart02, &itemsPart03, &itemsPart04, &itemsPart05, &itemsPart06, &itemsPart07, &itemsPart08, &itemsPart09, &itemsPart10, &itemsPart11, &itemsPart12, &itemsPart13, &itemsPart14, &itemsPart15, &itemsPart16, &itemsPart17, &itemsPart18}, nil),
	valueLayout("options", "CreationOptions", "options", "as listas de criação de personagem", &CreationOptions, func() ([]byte, error) { return marshalPlain(CreationOptions) }),
	listLayout("origins-source", "OriginSources", "origins_source", "as origens como o livro as imprime: perícias, poderes e itens", &OriginSources, nil, func(v OriginSource) string { return v.ID }),
	listLayout("origins", "Origins", "origins", "as origens achatadas em benefícios, que a ficha aplica", &Origins, []*[]Origin{&originsPart01, &originsPart02, &originsPart03, &originsPart04, &originsPart05}, nil),
	listLayout("race-defs", "RaceDefinitions", "race_defs", "as habilidades de raça, com os modificadores que somam na ficha", &RaceDefinitions, []*[]RaceDefinition{&raceDefinitionsPart01, &raceDefinitionsPart02, &raceDefinitionsPart03}, nil),
	listLayout("races", "Races", "races", "as raças: atributos, tamanho, deslocamento e visão", &Races, []*[]Race{&racesPart01, &racesPart02}, func(v Race) string { return v.ID }),
	listLayout("spell-schools", "SpellSchools", "spell_schools", "as oito escolas de magia (p172)", &SpellSchools, nil, nil),
	listLayout("spells", "Spells", "spells", "as magias do Capítulo 4", &Spells, []*[]Spell{&spellsPart01, &spellsPart02, &spellsPart03, &spellsPart04, &spellsPart05, &spellsPart06, &spellsPart07, &spellsPart08, &spellsPart09, &spellsPart10, &spellsPart11, &spellsPart12, &spellsPart13, &spellsPart14}, func(v Spell) string { return v.ID }),
	listLayout("tormenta-powers", "TormentaPowers", "tormenta_powers", "os poderes da Tormenta", &TormentaPowers, nil, func(v TormentaPower) string { return v.ID }),
}

// Resources são os nomes dos recursos, na ordem do registro.
var Resources = func() []string {
	out := make([]string, len(layouts))
	for i, r := range layouts {
		out[i] = r.name
	}
	return out
}()

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
	resourceBytes = make(map[string][]byte, len(layouts))
	for _, r := range layouts {
		raw, err := r.marshal()
		if err != nil {
			// Um verbete que não se serializa é binário quebrado: não há ficha
			// certa a servir.
			panic("transcript: " + r.name + ": " + err.Error())
		}
		resourceBytes[r.name] = raw
	}
}
