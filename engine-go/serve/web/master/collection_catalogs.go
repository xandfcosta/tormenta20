package master

import (
	"slices"
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
// busca varre, o que cada crachá casa, e como cada verbete se chama e se desenha.
type catalogSpec[T any] struct {
	Stop    railStop
	Load    func() []T
	Fields  func(T) []string
	Filters func() []collectionFilter
	Matches func(T, string, string) bool
	ID      func(T) string
	Name    func(T) string
	Card    func(T, bookui.BookAddress) templ.Component
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
	return collectionCatalog{
		Stop:    s.Stop,
		filters: s.Filters,
		entries: func(lit map[string][]string) []collectionEntry {
			if s.Matches == nil {
				return ready(s.Load())
			}
			return ready(applyFilters(s.Load(), lit, s.Matches))
		},
		matching: func(term string) []collectionEntry { return ready(filter(s.Load(), s.Fields, term)) },
	}
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
		Filters: func() []collectionFilter {
			return []collectionFilter{{Key: "efeito", Label: "Tipo de efeito", Options: effectOptions()}}
		},
		Matches: conditionMatches,
		ID:      func(c book.Condition) string { return c.ID },
		Name:    func(c book.Condition) string { return c.Name },
		Card:    conditionCard,
	}),
	catalogOf(catalogSpec[book.Hazard]{
		Stop:   railStop{"perigos", "Perigos", "AlertTriangle", "Armadilhas, doenças e o ambiente (p317), com CD e ND."},
		Load:   book.Hazards,
		Fields: book.HazardFields,
		Filters: func() []collectionFilter {
			return []collectionFilter{{Key: "tipo", Label: "Tipo", Options: hazardKindOptions()}}
		},
		Matches: hazardMatches,
		ID:      func(h book.Hazard) string { return h.ID },
		Name:    func(h book.Hazard) string { return h.Name },
		Card:    hazardCard,
	}),
	catalogOf(catalogSpec[book.Spell]{
		Stop:   railStop{"magias", "Magias", "Wand2", "As 198 magias, por círculo, com aprimoramentos."},
		Load:   func() []book.Spell { return book.Catalogs().Spells },
		Fields: book.SpellFields,
		Filters: func() []collectionFilter {
			return []collectionFilter{
				{Key: "circulo", Label: "Círculo", Options: circleOptions()},
				{Key: "escola", Label: "Escola", Options: schoolOptions()},
				{Key: "classe", Label: "Classe", Options: spellClassOptions()},
			}
		},
		Matches: spellMatches,
		ID:      func(m book.Spell) string { return m.ID },
		Name:    func(m book.Spell) string { return m.Name },
		Card:    spellCard,
	}),
	catalogOf(catalogSpec[book.Power]{
		Stop:   railStop{"poderes", "Poderes", "Star", "Poderes de classe, gerais e divinos numa lista só."},
		Load:   func() []book.Power { return book.Catalogs().Powers },
		Fields: book.PowerFields,
		Filters: func() []collectionFilter {
			return []collectionFilter{{Key: "fonte", Label: "Fonte", Options: powerSourceOptions()}}
		},
		Matches: powerMatches,
		ID:      func(p book.Power) string { return p.ID },
		Name:    func(p book.Power) string { return p.Name },
		Card:    powerCard,
	}),
	catalogOf(catalogSpec[book.Item]{
		Stop:   railStop{"itens", "Itens", "Backpack", "Armas, armaduras e equipamento, com preço e espaços."},
		Load:   func() []book.Item { return book.Catalogs().Items },
		Fields: book.ItemFields,
		Filters: func() []collectionFilter {
			return []collectionFilter{{Key: "familia", Label: "Família", Options: itemFamilies}}
		},
		Matches: itemMatches,
		ID:      func(i book.Item) string { return i.ID },
		Name:    func(i book.Item) string { return i.Name },
		Card:    itemCard,
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
		Filters: func() []collectionFilter {
			return []collectionFilter{
				{Key: "atributo", Label: "Atributo", Options: attributeOptions()},
				{Key: "treino", Label: "Treino", Options: []filterOption{
					{"so-treinada", "Só treinada"}, {"armadura", "Penalidade de armadura"},
				}},
			}
		},
		Matches: expertiseMatches,
		ID:      func(p book.Expertise) string { return p.ID },
		Name:    func(p book.Expertise) string { return p.Name },
		Card:    expertiseCard,
	}),
	catalogOf(catalogSpec[book.Race]{
		Stop:   railStop{"racas", "Raças", "Users2", "As 17 raças, com modificadores e habilidades."},
		Load:   func() []book.Race { races, _, _ := book.CharacterCatalogs(); return races },
		Fields: book.RaceFields,
		Filters: func() []collectionFilter {
			return []collectionFilter{{Key: "linhagem", Label: "Linhagem", Options: []filterOption{
				{"comum", "Comum"}, {"extra", "Exótica"},
			}}}
		},
		Matches: raceMatches,
		ID:      func(r book.Race) string { return r.ID },
		Name:    func(r book.Race) string { return r.Name },
		Card:    raceCard,
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
		Filters: func() []collectionFilter {
			return []collectionFilter{{Key: "energia", Label: "Energia", Options: []filterOption{
				{"positiva", "Positiva"}, {"negativa", "Negativa"}, {"qualquer", "Qualquer"},
			}}}
		},
		Matches: godMatches,
		ID:      func(d book.God) string { return d.ID },
		Name:    func(d book.God) string { return d.Name },
		Card:    godCard,
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
