package book

import (
	"slices"
	"strings"
	"sync"
)

// O DONO DE UMA ATIVAÇÃO — qual poder a concede — se diz AQUI, e só aqui: a ida (`ActivationOf`), a volta da ficha
// (`OwnerIndex`) e a volta da ativação para o verbete dela no catálogo (`entryOf`), de onde o preço do uso lê a
// classe e o registro herda a página. Respostas escritas à mão para cada uma já discordaram.
//
// PELO ID, ou PELO NOME. O nome é a ponte entre catálogos que grafam o mesmo poder de jeitos diferentes: a
// Armadura de Allihanna é `dahllan-armadura-de-allihanna` no de raças e `race.dahllan.armadura-de-allihanna` no
// de ativações; a Coragem é `class.paladino.julgamento-divino-coragem` no de poderes e
// `class.paladino.julgamento-coragem` no de ativações.
//
// O NOME PREFERE A PROCEDÊNCIA. Classes diferentes dão o mesmo nome a habilidades diferentes — o Esgrimista do
// guerreiro e o do bucaneiro, o Ímpeto de bárbaro, caçador e guerreiro —, e o nome solto fazia de cada um dono do
// outro. Mas há a MESMA habilidade em duas classes com uma ativação só no catálogo (a Empatia Selvagem do druida
// usa a do caçador; o Conhecimento Mágico do arcanista, a do clérigo): por isso o nome só atravessa a procedência
// quando a do poder não tem ativação nenhuma com aquele nome. A procedência é a CLASSE (`class.<classe>.`); raça,
// origem, deus, item e poder geral dividem uma só, a de fora das classes.

// PowerRef é um poder como a ficha o tem: o id do catálogo dele e o nome.
type PowerRef struct{ ID, Name string }

// ActivationOf acha a ativação de um poder pelo ID e, se falhar, pelo NOME — ver o cabeçalho deste arquivo.
//
// E há o caso dos DEGRAUS: "Inspiração +1" e "Fúria +3" são linhas de mesma postura, e o nome delas traz o sufixo
// do degrau. Sem tirar o sufixo, cada degrau cairia como passiva silenciosa em vez de resolver para a postura.
//
//	book.ActivationOf("class.paladino.julgamento-divino-coragem", "Julgamento Divino: Coragem") // a Coragem
func ActivationOf(id, name string) *Activation {
	registry := activationRegistry()
	if spec, found := registry.byID[id]; found && id != "" {
		return &spec
	}
	candidates := registry.byName[withoutStepSuffix(name)]
	// DE TRÁS PARA A FRENTE: com dois candidatos válidos (a Coragem Total de cada deus), o último do registro é o
	// que o índice por nome devolvia, e trocar a escolha trocaria o endereço do gesto na tela.
	for i := len(candidates) - 1; i >= 0; i-- {
		if nameLinks(candidates[i], id) {
			return &candidates[i]
		}
	}
	return nil
}

// OwnerIndex são os poderes de uma ficha montados UMA vez, para perguntar várias ativações em seguida.
//
//	owned := book.OwnerIndexOf(poderes)
//	for _, a := range book.Activations() { if owned.Owns(a) { … } }
type OwnerIndex struct {
	ids    map[string]bool
	byName map[string][]PowerRef
}

// OwnerIndexOf monta o índice dos poderes de uma ficha.
func OwnerIndexOf(powers []PowerRef) OwnerIndex {
	index := OwnerIndex{ids: make(map[string]bool, len(powers)), byName: make(map[string][]PowerRef, len(powers))}
	for _, power := range powers {
		index.ids[power.ID] = true
		key := withoutStepSuffix(power.Name)
		index.byName[key] = append(index.byName[key], power)
	}
	return index
}

// Owns diz se algum dos poderes é dono da ativação — pelo id, ou pelo nome como o cabeçalho deste arquivo explica.
func (o OwnerIndex) Owns(spec Activation) bool {
	if o.ids[spec.ID] {
		return true
	}
	for _, power := range o.byName[withoutStepSuffix(spec.Name)] {
		if nameLinks(spec, power.ID) {
			return true
		}
	}
	return false
}

// catalogEntry é o VERBETE de um poder no catálogo — o poder de classe, o geral, o do deus, o item, a habilidade de
// raça —, com o que a ativação lê dele: a classe e a página.
type catalogEntry struct {
	id, name string
	// class é a classe do poder de classe, e vazia em todos os outros: é por ela que o fator de uma classe alcança a
	// habilidade (a metade do Artista Completo nas de bardo, p45).
	class string
	page  int
}

// entryOf é o verbete dono da ativação: pelo id; senão pelo nome sem o sufixo de degrau, na MESMA procedência.
//
// Ele NÃO atravessa a procedência, ao contrário do `ActivationOf`. A ida da ficha para a ativação atravessa porque o
// druida não tem ativação própria de Empatia Selvagem e usa a do caçador; a volta da ativação para o verbete não,
// porque a ativação do caçador não é o verbete do druida. Pela mesma razão a Engenhosidade do kliren não tem a
// classe do inventor.
//
// Sem verbete, ele devolve o verbete vazio: classe vazia e página zero, que é o que o preço e a herança leem.
//
//	entryOf(*ActivationOf("class.paladino.julgamento-coragem", "")) // Paladino, p83, ligada pelo nome
func entryOf(spec Activation) catalogEntry {
	entries := catalogEntries()
	if entry, found := entries.byID[spec.ID]; found {
		return entry
	}
	for _, entry := range entries.byName[withoutStepSuffix(spec.Name)] {
		if sameScope(spec.ID, entry.id) {
			return entry
		}
	}
	return catalogEntry{}
}

// catalogEntryIndex são os verbetes por id, e por nome sem o sufixo de degrau em ordem de id, para a escolha não
// depender da ordem de um `map`.
type catalogEntryIndex struct {
	byID   map[string]catalogEntry
	byName map[string][]catalogEntry
}

// catalogEntries junta os cinco catálogos que têm verbete de poder. O id do item leva o `item.` do registro de
// ativações.
var catalogEntries = sync.OnceValue(func() catalogEntryIndex {
	var entries []catalogEntry
	for _, power := range ClassPowers() {
		entries = append(entries, catalogEntry{power.ID, power.Name, power.ClassName, power.BookPage})
	}
	for _, power := range GeneralPowers() {
		entries = append(entries, catalogEntry{power.ID, power.Name, "", power.BookPage})
	}
	for _, power := range DivinePowers() {
		entries = append(entries, catalogEntry{power.ID, power.Name, "", power.BookPage})
	}
	for _, item := range Catalogs().Items {
		entries = append(entries, catalogEntry{"item." + item.ID, item.Name, "", item.BookPage})
	}
	for key, race := range RaceTraitsByKey() {
		// O índice das raças tem cada uma duas vezes, pelo id e pelo nome.
		if key != race.ID {
			continue
		}
		for _, ability := range race.Abilities {
			entries = append(entries, catalogEntry{ability.ID, ability.Name, "", ability.BookPage})
		}
	}
	slices.SortFunc(entries, func(a, b catalogEntry) int { return strings.Compare(a.id, b.id) })
	index := catalogEntryIndex{byID: map[string]catalogEntry{}, byName: map[string][]catalogEntry{}}
	for _, entry := range entries {
		index.byID[entry.id] = entry
		key := withoutStepSuffix(entry.name)
		index.byName[key] = append(index.byName[key], entry)
	}
	return index
})

// registryIndexes são os índices do registro de ativações, montados uma vez.
type registryIndexes struct {
	byID   map[string]Activation
	byName map[string][]Activation
}

var activationRegistry = sync.OnceValue(func() registryIndexes {
	registry := registryIndexes{byID: map[string]Activation{}, byName: map[string][]Activation{}}
	for _, a := range Activations() {
		registry.byID[a.ID] = a
		registry.byName[a.Name] = append(registry.byName[a.Name], a)
	}
	return registry
})

// nameLinks diz se o NOME liga a ativação ao poder de id `powerID`: na mesma procedência, ou fora dela quando a
// procedência do poder não tem ativação nenhuma com aquele nome — ver o cabeçalho deste arquivo.
func nameLinks(spec Activation, powerID string) bool {
	return sameScope(spec.ID, powerID) || !activationScopeNames()[scopeOf(powerID)+"|"+spec.Name]
}

// activationScopeNames diz que a procedência tem uma ativação com o nome: "paladino|Julgamento Divino: Coragem".
// Ele lê o registro TRANSCRITO, e não o `Activations`: a página herdada do dono pergunta pela regra do nome, e o
// registro com a página herdada ainda não existe nessa hora.
var activationScopeNames = sync.OnceValue(func() map[string]bool {
	names := map[string]bool{}
	for _, a := range transcribedActivations() {
		names[scopeOf(a.ID)+"|"+a.Name] = true
	}
	return names
})

// scopeOf é a procedência de um id: a classe de `class.<classe>.<slug>`, ou vazia para tudo o que é de fora das
// classes.
func scopeOf(id string) string {
	rest, isClass := strings.CutPrefix(id, "class.")
	if !isClass {
		return ""
	}
	class, _, _ := strings.Cut(rest, ".")
	return class
}

// sameScope diz que os dois ids são da mesma procedência.
func sameScope(a, b string) bool {
	return scopeOf(a) == scopeOf(b)
}

// withoutStepSuffix tira o " +N" do fim de "Inspiração +2".
func withoutStepSuffix(name string) string {
	if len(name) < 4 {
		return name
	}
	end := len(name)
	for end > 0 && name[end-1] >= '0' && name[end-1] <= '9' {
		end--
	}
	if end == len(name) || end < 2 || name[end-1] != '+' || name[end-2] != ' ' {
		return name
	}
	return name[:end-2]
}
