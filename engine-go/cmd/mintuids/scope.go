package main

import "t20engine/domain/catalog/transcript"

// mintable é um verbete visto pela cunhagem: a identidade PÚBLICA — que diz
// quais linhas são o MESMO conceito e portanto dividem um uid — e o uid dele.
type mintable struct {
	identity string
	uid      *string
}

// scopeEntry diz a espécie de um recurso e os CAMINHOS até os verbetes dele: a
// coleção raiz e, quando há, a coleção aninhada que o banco referencia por conta
// própria.
type scopeEntry struct {
	file   string
	prefix string
	paths  func() [][]mintable
}

// collect vê cada elemento de uma lista como verbete, pelo endereço — é o uid
// do próprio literal que a cunhagem preenche.
func collect[T any](list []T, view func(*T) mintable) []mintable {
	out := make([]mintable, len(list))
	for i := range list {
		out[i] = view(&list[i])
	}
	return out
}

// scope é ONDE moram os verbetes de cada recurso, declarado UM A UM.
//
// Duas tentativas anteriores usaram um predicado esperto — "objeto com nome e
// descrição" — e as duas mediram o conjunto errado: uma deixou de fora os itens
// com `modifiers` vazio, a outra varreria as ativações, cujo id é DERIVADO do
// poder e que portanto não têm identidade própria. A declaração explícita é mais
// longa e não erra; recurso fora desta lista e de `outOfScope` sai NOMEADO.
var scope = []scopeEntry{
	{"class-powers.json", "pwr", func() [][]mintable {
		return [][]mintable{collect(transcript.ClassPowers, func(v *transcript.ClassPower) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"general-powers.json", "pwr", func() [][]mintable {
		return [][]mintable{collect(transcript.GeneralPowers, func(v *transcript.GeneralPower) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"tormenta-powers.json", "pwr", func() [][]mintable {
		return [][]mintable{collect(transcript.TormentaPowers, func(v *transcript.TormentaPower) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"granted-powers.json", "pwr", func() [][]mintable {
		return [][]mintable{collect(transcript.GrantedPowers, func(v *transcript.GrantedPower) mintable { return mintable{v.ID, &v.UID} })}
	}},
	// O poder divino não tem `id`: a identidade é o NOME, que é o que junta as
	// linhas de um poder concedido por mais de um deus.
	{"divine-powers.json", "pwr", func() [][]mintable {
		return [][]mintable{collect(transcript.DivinePowers, func(v *transcript.DivinePower) mintable { return mintable{v.Name, &v.UID} })}
	}},
	{"spells.json", "spl", func() [][]mintable {
		return [][]mintable{collect(transcript.Spells, func(v *transcript.Spell) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"items.json", "itm", func() [][]mintable {
		return [][]mintable{collect(transcript.Items, func(v *transcript.Item) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"conditions.json", "cnd", func() [][]mintable {
		return [][]mintable{collect(transcript.Conditions, func(v *transcript.Condition) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"effect-types.json", "eft", func() [][]mintable {
		return [][]mintable{collect(transcript.EffectTypes, func(v *transcript.EffectType) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"spell-schools.json", "sch", func() [][]mintable {
		return [][]mintable{collect(transcript.SpellSchools, func(v *transcript.SpellSchool) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"expertises.json", "exp", func() [][]mintable {
		return [][]mintable{collect(transcript.Expertises, func(v *transcript.Expertise) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"classes.json", "cls", func() [][]mintable {
		return [][]mintable{collect(transcript.Classes, func(v *transcript.Class) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"gods.json", "god", func() [][]mintable {
		return [][]mintable{collect(transcript.Gods, func(v *transcript.God) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"races.json", "rac", func() [][]mintable {
		return [][]mintable{collect(transcript.Races, func(v *transcript.Race) mintable { return mintable{v.ID, &v.UID} })}
	}},
	{"bestiary.json", "cre", func() [][]mintable {
		return [][]mintable{collect(transcript.Bestiary, func(v *transcript.Creature) mintable { return mintable{v.ID, &v.UID} })}
	}},
	// A escolha de benefício de origem e a de habilidade de raça são gravadas
	// na ficha, então o verbete ANINHADO também precisa de identidade.
	{"origins.json", "org", func() [][]mintable {
		var benefits []mintable
		for i := range transcript.Origins {
			benefits = append(benefits, collect(transcript.Origins[i].Benefits, func(v *transcript.OriginBenefit) mintable { return mintable{v.ID, &v.UID} })...)
		}
		return [][]mintable{collect(transcript.Origins, func(v *transcript.Origin) mintable { return mintable{v.ID, &v.UID} }), benefits}
	}},
	// O origins-source é a transcrição do LIVRO das MESMAS origens; ele não
	// ganha uid próprio, ele HERDA o da origem (`mirrorOrigins`).
	{"origins-source.json", "org", func() [][]mintable { return nil }},
	{"race-defs.json", "rac", func() [][]mintable {
		var abilities []mintable
		for i := range transcript.RaceDefinitions {
			abilities = append(abilities, collect(transcript.RaceDefinitions[i].Abilities, func(v *transcript.RaceAbility) mintable { return mintable{v.ID, &v.UID} })...)
		}
		return [][]mintable{collect(transcript.RaceDefinitions, func(v *transcript.RaceDefinition) mintable { return mintable{v.ID, &v.UID} }), abilities}
	}},
}

// outOfScope existe para a ausência ser DITA:
//
//	activations       o id é o do poder que a ativação descreve; ela é satélite
//	options           listas de strings para a tela, não verbetes
//	gm-tables, dungeon-design, devotee-terms   tabelas de mestre
//	class-expertises  é APONTAMENTO — classe para nomes de perícia, e as
//	                  perícias já têm identidade em expertises
var outOfScope = map[string]bool{
	"activations.json": true, "options.json": true,
	"gm-tables.json": true, "dungeon-design.json": true, "devotee-terms.json": true,
	"class-expertises.json": true,
}

func scopeOf(file string) (scopeEntry, bool) {
	for _, s := range scope {
		if s.file == file {
			return s, true
		}
	}
	return scopeEntry{}, false
}
