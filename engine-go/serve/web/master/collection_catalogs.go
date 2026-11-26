package master

import (
	"slices"
	"strconv"
	"strings"

	"github.com/a-h/templ"

	"t20engine/domain/book"
	"t20engine/domain/search"
	"t20engine/serve/web/bookui"
)

// OS CATÁLOGOS do mestre, um DESCRITOR por catálogo.
//
// Tudo o que a coleção faz com um catálogo sai desta lista: a rota, a parada no
// trilho, a aba de queda, a busca unificada, os filtros, o elo para um verbete e
// os cartões. Antes eram nove lugares escritos à mão por catálogo, e só três
// cobrados por teste: o catálogo esquecido no `switch` da aba mostrava as
// Condições, e o esquecido na busca unificada sumia dela sem ninguém acusar.
//
// O BUSCADOR (`finder`) fica de fora de propósito: ele tem a Criatura, que não é
// catálogo daqui, pontua o achado à sua maneira e não pode importar este pacote.

// collectionEntry é um verbete já pronto para a cena: o id que o elo pede, o nome
// que a caixa do elo escreve no título, e o cartão.
type collectionEntry struct {
	ID   string
	Name string
	card func(bookui.BookAddress) templ.Component
}

// Card é o cartão do verbete, com o livro de quem o pede.
func (e collectionEntry) Card(bookRef bookui.BookAddress) templ.Component { return e.card(bookRef) }

// collectionCatalog é um catálogo da coleção. Ele se monta pelo `catalogOf`, que
// guarda o tipo do verbete atrás das três perguntas que a cena faz.
type collectionCatalog struct {
	// Stop é a parada no trilho; o `Slug` é a aba, no caminho e no `?aba=` do elo.
	Stop railStop
	// filters são as linhas de crachá; nulo é catálogo sem filtro.
	filters func() []collectionFilter
	// entries são os verbetes que sobram dos crachás acesos.
	entries func(lit map[string][]string) []collectionEntry
	// matching são os verbetes que a busca unificada acha.
	matching func(term string) []collectionEntry
}

// catalogSpec é como um catálogo se DECLARA: de onde vêm os verbetes, o que a
// busca varre, as linhas de crachá, e como cada verbete se chama e se desenha.
type catalogSpec[T any] struct {
	Stop    railStop
	Load    func() []T
	Fields  func(T) []string
	Filters []filterLine[T]
	ID      func(T) string
	Name    func(T) string
	Card    func(T, bookui.BookAddress) templ.Component
}

// filterLine é uma linha de crachás COM o que cada crachá dela casa.
//
// A chave mora aqui uma vez só. Ela era escrita na linha e de novo no `switch` de
// uma função de casamento por catálogo, que terminava em `return true`: uma
// grafia diferente nos dois lugares fazia o crachá acender e a lista não filtrar,
// sem erro nenhum.
type filterLine[T any] struct {
	// Key é o nome no endereço e no sinal (`?circulo=3`). Minúscula e uma
	// palavra: ela vira nome de sinal do Datastar, e o analisador de HTML
	// minuscula nome de atributo.
	Key     string
	Label   string
	Options func() []filterOption
	Match   func(entry T, value string) bool
}

// fixedOptions são as opções escritas à mão, para a linha cujo dado não as lista.
func fixedOptions(options ...filterOption) func() []filterOption {
	return func() []filterOption { return options }
}

// catalogOf monta o catálogo a partir da declaração.
//
//	catalogOf(catalogSpec[book.God]{Stop: …, Load: book.CatalogGods, …})
func catalogOf[T any](s catalogSpec[T]) collectionCatalog {
	ready := func(list []T) []collectionEntry {
		outside := make([]collectionEntry, 0, len(list))
		for _, v := range list {
			outside = append(outside, collectionEntry{
				ID:   s.ID(v),
				Name: s.Name(v),
				card: func(bookRef bookui.BookAddress) templ.Component { return s.Card(v, bookRef) },
			})
		}
		return outside
	}
	c := collectionCatalog{
		Stop:     s.Stop,
		entries:  func(lit map[string][]string) []collectionEntry { return ready(applyFilters(s.Load(), lit, s.Filters)) },
		matching: func(term string) []collectionEntry { return ready(filter(s.Load(), s.Fields, term)) },
	}
	if len(s.Filters) > 0 {
		c.filters = func() []collectionFilter {
			drawn := make([]collectionFilter, len(s.Filters))
			for i, line := range s.Filters {
				drawn[i] = collectionFilter{Key: line.Key, Label: line.Label, Options: line.Options()}
			}
			return drawn
		}
	}
	return c
}

// Filters são as linhas de crachá do catálogo, ou nenhuma.
func (c collectionCatalog) Filters() []collectionFilter {
	if c.filters == nil {
		return nil
	}
	return c.filters()
}

// group é o catálogo inteiro, com os crachás aplicados.
func (c collectionCatalog) group(lit map[string][]string) collectionGroup {
	return collectionGroup{Label: c.Stop.Label, Entries: c.entries(lit)}
}

// A ORDEM da lista é a dos grupos da busca unificada, com a condição na frente:
// é a consulta mais comum no meio do combate, e a primeira é também a aba de
// queda do endereço desconhecido. O TRILHO não segue esta ordem: ele é
// alfabético (`catalogStops`).
//
// Classes (14), Efeitos (18) e Escolas (8) não têm filtro, e é decisão e não
// esquecimento: numa lista que cabe na tela, filtro é cromo que não poupa rolagem.
var collectionCatalogs = []collectionCatalog{
	catalogOf(catalogSpec[book.Condition]{
		Stop:   railStop{"condicoes", "Condições", "HeartPulse", "As 35 condições do livro e o que cada uma faz."},
		Load:   func() []book.Condition { return book.Catalogs().Conditions },
		Fields: book.ConditionFields,
		Filters: []filterLine[book.Condition]{{Key: "efeito", Label: "Tipo de efeito", Options: effectOptions,
			Match: func(c book.Condition, v string) bool { return slices.Contains(c.Tags, v) }}},
		ID:   func(c book.Condition) string { return c.ID },
		Name: func(c book.Condition) string { return c.Name },
		Card: conditionCard,
	}),
	catalogOf(catalogSpec[book.Hazard]{
		Stop:   railStop{"perigos", "Perigos", "AlertTriangle", "Armadilhas, doenças e o ambiente (p317), com CD e ND."},
		Load:   book.Hazards,
		Fields: book.HazardFields,
		Filters: []filterLine[book.Hazard]{{Key: "tipo", Label: "Tipo", Options: hazardKindOptions,
			Match: func(h book.Hazard, v string) bool { return h.Kind == v }}},
		ID:   func(h book.Hazard) string { return h.ID },
		Name: func(h book.Hazard) string { return h.Name },
		Card: hazardCard,
	}),
	catalogOf(catalogSpec[book.Spell]{
		Stop:   railStop{"magias", "Magias", "Wand2", "As 198 magias, por círculo, com aprimoramentos."},
		Load:   func() []book.Spell { return book.Catalogs().Spells },
		Fields: book.SpellFields,
		Filters: []filterLine[book.Spell]{
			{Key: "circulo", Label: "Círculo", Options: circleOptions,
				Match: func(m book.Spell, v string) bool { return strconv.Itoa(m.Circle) == v }},
			{Key: "escola", Label: "Escola", Options: schoolOptions,
				Match: func(m book.Spell, v string) bool { return m.School == v }},
			{Key: "classe", Label: "Classe", Options: spellClassOptions,
				Match: func(m book.Spell, v string) bool { return slices.Contains(m.Classes, v) }},
		},
		ID:   func(m book.Spell) string { return m.ID },
		Name: func(m book.Spell) string { return m.Name },
		Card: spellCard,
	}),
	catalogOf(catalogSpec[book.Power]{
		Stop:   railStop{"poderes", "Poderes", "Star", "Poderes de classe, gerais e divinos numa lista só."},
		Load:   func() []book.Power { return book.Catalogs().Powers },
		Fields: book.PowerFields,
		// A FONTE casa pelo COMEÇO: ela é "Geral · combate" e "Divino · Khalmyr",
		// e o crachá é a família.
		Filters: []filterLine[book.Power]{{Key: "fonte", Label: "Fonte", Options: powerSourceOptions,
			Match: func(p book.Power, v string) bool { return p.Source == v || strings.HasPrefix(p.Source, v+" ·") }}},
		ID:   func(p book.Power) string { return p.ID },
		Name: func(p book.Power) string { return p.Name },
		Card: powerCard,
	}),
	catalogOf(catalogSpec[book.Item]{
		Stop:   railStop{"itens", "Itens", "Backpack", "Armas, armaduras e equipamento, com preço e espaços."},
		Load:   func() []book.Item { return book.Catalogs().Items },
		Fields: book.ItemFields,
		Filters: []filterLine[book.Item]{{Key: "familia", Label: "Família", Options: fixedOptions(itemFamilies...),
			Match: func(i book.Item, v string) bool { return familyOfCategory(i.Category) == v }}},
		ID:   func(i book.Item) string { return i.ID },
		Name: func(i book.Item) string { return i.Name },
		Card: itemCard,
	}),
	catalogOf(catalogSpec[book.EffectKind]{
		Stop:   railStop{"efeitos", "Efeitos", "Zap", "Os tipos de efeito (p228): medo, mental, metabolismo…"},
		Load:   book.EffectKinds,
		Fields: book.EffectFields,
		ID:     func(e book.EffectKind) string { return e.ID },
		Name:   func(e book.EffectKind) string { return e.Name },
		Card:   effectCard,
	}),
	catalogOf(catalogSpec[book.SpellSchool]{
		Stop:   railStop{"escolas", "Escolas", "Wand2", "As oito escolas de magia (p172), com o que cada uma faz."},
		Load:   book.SpellSchools,
		Fields: book.SchoolFields,
		ID:     func(e book.SpellSchool) string { return e.ID },
		Name:   func(e book.SpellSchool) string { return e.Name },
		Card:   schoolCard,
	}),
	catalogOf(catalogSpec[book.Expertise]{
		Stop:   railStop{"pericias", "Perícias", "Dices", "As 29 perícias, com atributo e as regras de treino."},
		Load:   book.Expertises,
		Fields: book.ExpertiseFields,
		Filters: []filterLine[book.Expertise]{
			{Key: "atributo", Label: "Atributo", Options: attributeOptions,
				Match: func(p book.Expertise, v string) bool { return p.Attribute == v }},
			// As duas marcas do livro num filtro só: são as duas coisas que mudam
			// COMO a perícia se usa, e separá-las em duas linhas de um crachá cada
			// gastaria duas linhas para dizer o que uma diz.
			{Key: "treino", Label: "Treino",
				Options: fixedOptions(filterOption{"so-treinada", "Só treinada"}, filterOption{"armadura", "Penalidade de armadura"}),
				Match: func(p book.Expertise, v string) bool {
					return (v == "so-treinada" && p.TrainedOnly) || (v == "armadura" && p.ArmorPenalty)
				}},
		},
		ID:   func(p book.Expertise) string { return p.ID },
		Name: func(p book.Expertise) string { return p.Name },
		Card: expertiseCard,
	}),
	catalogOf(catalogSpec[book.Race]{
		Stop:   railStop{"racas", "Raças", "Users2", "As 17 raças, com modificadores e habilidades."},
		Load:   func() []book.Race { races, _, _ := book.CharacterCatalogs(); return races },
		Fields: book.RaceFields,
		Filters: []filterLine[book.Race]{{Key: "linhagem", Label: "Linhagem",
			Options: fixedOptions(filterOption{"comum", "Comum"}, filterOption{"extra", "Exótica"}),
			Match:   func(r book.Race, v string) bool { return r.Tier == v }}},
		ID:   func(r book.Race) string { return r.ID },
		Name: func(r book.Race) string { return r.Name },
		Card: raceCard,
	}),
	catalogOf(catalogSpec[book.Class]{
		Stop:   railStop{"classes", "Classes", "Shield", "As 14 classes, com perícias treinadas e poderes."},
		Load:   func() []book.Class { _, classes, _ := book.CharacterCatalogs(); return classes },
		Fields: book.ClassFields,
		ID:     func(c book.Class) string { return c.ID },
		Name:   func(c book.Class) string { return c.Name },
		Card:   classCard,
	}),
	catalogOf(catalogSpec[book.God]{
		Stop:   railStop{"deuses", "Deuses", "Church", "Os 20 deuses, com símbolo, devotos e poderes concedidos."},
		Load:   book.CatalogGods,
		Fields: book.GodFields,
		Filters: []filterLine[book.God]{{Key: "energia", Label: "Energia",
			Options: fixedOptions(filterOption{"positiva", "Positiva"}, filterOption{"negativa", "Negativa"}, filterOption{"qualquer", "Qualquer"}),
			Match:   func(d book.God, v string) bool { return d.Energy == v }}},
		ID:   func(d book.God) string { return d.ID },
		Name: func(d book.God) string { return d.Name },
		Card: godCard,
	}),
}

// catalogNamed é o catálogo da aba. Aba desconhecida cai na primeira em vez de
// mostrar tela vazia: o endereço é digitado, e alguém o digita errado.
func catalogNamed(aba string) collectionCatalog {
	for _, c := range collectionCatalogs {
		if c.Stop.Slug == aba {
			return c
		}
	}
	return collectionCatalogs[0]
}

func knownTab(aba string) string { return catalogNamed(aba).Stop.Slug }

// catalogStops são as paradas dos catálogos no trilho: o bestiário na frente, por
// ser o mais consultado no meio do combate, e o resto em ordem alfabética. A ordem
// é a do nome DOBRADO, sem acento: pelo byte, "Perigos" viria antes de "Perícias".
func catalogStops() []railStop {
	stops := make([]railStop, 0, len(collectionCatalogs))
	for _, c := range collectionCatalogs {
		stops = append(stops, c.Stop)
	}
	slices.SortFunc(stops, func(a, b railStop) int {
		return strings.Compare(search.Fold(a.Label), search.Fold(b.Label))
	})
	bestiary := railStop{"bestiario", "Bestiário", "Skull", "Criaturas por ND, tipo e tamanho, com ataques e habilidades."}
	return append([]railStop{bestiary}, stops...)
}
