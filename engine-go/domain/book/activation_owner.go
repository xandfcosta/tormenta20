package book

import (
	"slices"
	"strings"
	"sync"
)

// O DONO DE UMA ATIVAÇÃO — qual poder a concede — se diz AQUI, e só aqui: a ida (`ActivationOf`), a volta da ficha
// (`OwnerIndex`), a classe que o preço do uso lê (`activationClass`) e a página que a ativação herda (`ownerPage`)
// perguntam a mesma coisa, e respostas escritas à mão para cada uma já discordaram.
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

// activationClass é a classe da habilidade, quando ela é poder de classe, e vazia quando não é: é por ela que o
// fator de uma classe a alcança (a metade do Artista Completo nas de bardo, p45). A ativação de fora das classes
// não tem classe, ainda que divida o nome com uma — a Engenhosidade do kliren não é a do inventor.
func activationClass(spec Activation) string {
	if class := ClassOfPower(spec.ID); class != "" {
		return class
	}
	if scopeOf(spec.ID) == "" {
		return ""
	}
	for _, power := range classPowersByName()[withoutStepSuffix(spec.Name)] {
		if sameScope(spec.ID, power.ID) {
			return power.ClassName
		}
	}
	return ""
}

// ownerPage é a página do verbete do DONO da ativação — o poder de classe, o geral, o do deus, o item —, achado
// pela regra deste arquivo: pelo id, ou pelo nome preferindo a mesma procedência. Zero quando nenhum dono tem
// página (a raça, a origem): só aí a ativação escreve a sua.
//
// Quando a ativação serve a dois donos (a Esquiva Sobrenatural do bárbaro, p42, e a do ladino, p74), a página dela
// é a do dono da mesma procedência; a ficha mostra a do poder que a pessoa tem.
//
//	ownerPage(*ActivationOf("class.paladino.julgamento-coragem", "")) // 83: a Coragem do paladino, ligada pelo nome
func ownerPage(spec Activation) int {
	owners := pagedOwners()
	if page := owners.byID[spec.ID]; page > 0 {
		return page
	}
	across := 0
	for _, owner := range owners.byName[withoutStepSuffix(spec.Name)] {
		switch {
		case sameScope(spec.ID, owner.id):
			return owner.page
		case across == 0 && nameLinks(spec, owner.id):
			across = owner.page
		}
	}
	return across
}

// pagedOwner é um poder que tem página no livro.
type pagedOwner struct {
	id, name string
	page     int
}

// pagedOwnerIndex são os poderes com página: por id, e por nome sem o sufixo de degrau, em ordem de id para a
// escolha não depender da ordem de um `map`.
type pagedOwnerIndex struct {
	byID   map[string]int
	byName map[string][]pagedOwner
}

// pagedOwners junta os catálogos que dão página a um poder. O id do item leva o `item.` do registro de ativações.
var pagedOwners = sync.OnceValue(func() pagedOwnerIndex {
	var owners []pagedOwner
	for _, power := range ClassPowers() {
		owners = append(owners, pagedOwner{power.ID, power.Name, power.BookPage})
	}
	for _, power := range GeneralPowers() {
		owners = append(owners, pagedOwner{power.ID, power.Name, power.BookPage})
	}
	for _, power := range DivinePowers() {
		owners = append(owners, pagedOwner{power.ID, power.Name, power.BookPage})
	}
	for _, item := range Catalogs().Items {
		owners = append(owners, pagedOwner{"item." + item.ID, item.Name, item.BookPage})
	}
	slices.SortFunc(owners, func(a, b pagedOwner) int { return strings.Compare(a.id, b.id) })
	index := pagedOwnerIndex{byID: map[string]int{}, byName: map[string][]pagedOwner{}}
	for _, owner := range owners {
		if owner.page == 0 {
			continue
		}
		index.byID[owner.id] = owner.page
		key := withoutStepSuffix(owner.name)
		index.byName[key] = append(index.byName[key], owner)
	}
	return index
})

// classPowersByName são os poderes de classe por nome sem o sufixo de degrau.
var classPowersByName = sync.OnceValue(func() map[string][]ClassPower {
	byName := map[string][]ClassPower{}
	for _, power := range ClassPowers() {
		key := withoutStepSuffix(power.Name)
		byName[key] = append(byName[key], power)
	}
	return byName
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
