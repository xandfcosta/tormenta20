package master

import (
	"slices"
	"strconv"
	"strings"
	"t20engine/domain/book"
)

// OS FILTROS de cada catálogo.
//
// O bestiário sempre teve os dele — ND e tipo —, e os outros oito nasceram com
// busca e mais nada. O dono pediu filtro específico por catálogo, e a forma é a
// MESMA do bestiário: crachás que ligam e desligam, porque o mestre já aprendeu
// esse gesto lá.
//
// A regra de combinação é a do bestiário também: OU dentro de um filtro (2º ou
// 3º círculo), E entre filtros (2º círculo E da escola de evocação). É o que se
// espera de uma linha de crachás, e é o que a lista de tipos já fazia.
//
// Os filtros SOMEM durante a busca. Com termo digitado a cena responde outra
// pergunta — os oito catálogos, não este —, e um crachá de círculo aceso sobre
// uma lista que tem itens e condições diria que ele está filtrando o que não
// filtra.

// filterOption é um crachá: o valor que vai para o endereço e o rótulo que se lê.
type filterOption struct {
	Value string
	Label string
}

// collectionFilter é uma LINHA de crachás.
type collectionFilter struct {
	// Key é o nome no endereço e no sinal (`?circulo=3`). Minúscula e uma
	// palavra: ela vira nome de sinal do Datastar, e o analisador de HTML
	// minuscula nome de atributo.
	Key     string
	Label   string
	Options []filterOption
}

// filtersForTab diz quais linhas de crachá aquele catálogo mostra.
//
// Classes (14) e Efeitos (18) não têm nenhuma, e é decisão e não esquecimento:
// numa lista que cabe na tela, filtro é cromo que não poupa rolagem.
func filtersForTab(aba string) []collectionFilter {
	switch aba {
	case "condicoes":
		return []collectionFilter{{Key: "efeito", Label: "Tipo de efeito", Options: effectOptions()}}
	case "magias":
		return []collectionFilter{
			{Key: "circulo", Label: "Círculo", Options: circleOptions()},
			{Key: "escola", Label: "Escola", Options: schoolOptions()},
			{Key: "classe", Label: "Classe", Options: spellClassOptions()},
		}
	case "pericias":
		return []collectionFilter{
			{Key: "atributo", Label: "Atributo", Options: attributeOptions()},
			{Key: "treino", Label: "Treino", Options: []filterOption{
				{"so-treinada", "Só treinada"}, {"armadura", "Penalidade de armadura"},
			}},
		}
	case "poderes":
		return []collectionFilter{{Key: "fonte", Label: "Fonte", Options: powerSourceOptions()}}
	case "itens":
		return []collectionFilter{{Key: "familia", Label: "Família", Options: itemFamilies}}
	case "deuses":
		return []collectionFilter{{Key: "energia", Label: "Energia", Options: []filterOption{
			{"positiva", "Positiva"}, {"negativa", "Negativa"}, {"qualquer", "Qualquer"},
		}}}
	case "racas":
		return []collectionFilter{{Key: "linhagem", Label: "Linhagem", Options: []filterOption{
			{"comum", "Comum"}, {"extra", "Exótica"},
		}}}
	}
	return nil
}

// ── as opções que saem do próprio catálogo ───────────────────────────────────
//
// Lidas do dado e não escritas à mão: escola nova no livro aparece sozinha, e
// uma lista fixa aqui seria a que fica para trás em silêncio.

func effectOptions() []filterOption {
	used := map[string]bool{}
	for _, c := range book.Catalogs().Conditions {
		for _, t := range c.Tags {
			used[t] = true
		}
	}
	var outside []filterOption
	for _, e := range book.EffectKinds() {
		if used[e.ID] {
			outside = append(outside, filterOption{e.ID, e.Name})
		}
	}
	return outside
}

func circleOptions() []filterOption {
	var outside []filterOption
	for _, circle := range distinctValues(book.Catalogs().Spells, func(m book.Spell) string {
		return strconv.Itoa(m.Circle)
	}) {
		outside = append(outside, filterOption{circle, circle + "º"})
	}
	return outside
}

func schoolOptions() []filterOption {
	var outside []filterOption
	for _, school := range distinctValues(book.Catalogs().Spells, func(m book.Spell) string {
		return m.School
	}) {
		outside = append(outside, filterOption{school, book.SchoolName(school)})
	}
	return outside
}

func spellClassOptions() []filterOption {
	seen := map[string]bool{}
	var outside []filterOption
	for _, m := range book.Catalogs().Spells {
		for _, c := range m.Classes {
			if !seen[c] {
				seen[c] = true
				outside = append(outside, filterOption{c, c})
			}
		}
	}
	slices.SortFunc(outside, func(a, b filterOption) int { return strings.Compare(a.Label, b.Label) })
	return outside
}

// powerSourceOptions: as 14 classes mais "geral" e "divino".
//
// A fonte de um poder é a string que o cartão mostra ("Arcanista", "Geral ·
// combate", "Divino · Khalmyr"), e filtrar por ela inteira daria um crachá por
// deus. O filtro casa pelo COMEÇO — ver `powerMatches`.
func powerSourceOptions() []filterOption {
	outside := []filterOption{{"Geral", "Geral"}, {"Divino", "Divino"}}
	_, classes, _ := book.CharacterCatalogs()
	for _, c := range classes {
		outside = append(outside, filterOption{c.Name, c.Name})
	}
	return outside
}

// itemFamilies agrupa as quinze categorias em seis famílias.
//
// Quinze crachás numa linha é uma lista, não um filtro: o mestre procura "uma
// arma" e não "uma arma marcial de fogo". As categorias continuam escritas no
// cartão, que é onde a distinção fina importa.
var itemFamilies = []filterOption{
	{"armas", "Armas"},
	{"protecao", "Armaduras e escudos"},
	{"vestuario", "Vestuário"},
	{"consumo", "Consumíveis"},
	{"montaria", "Montarias e veículos"},
	{"outros", "Outros"},
}

// familyOfCategory mapeia a categoria do dado para a família do crachá.
func familyOfCategory(category string) string {
	switch {
	case strings.HasPrefix(category, "weapon"):
		return "armas"
	case strings.HasPrefix(category, "armor"), category == "shield":
		return "protecao"
	case category == "apparel":
		return "vestuario"
	case category == "consumable", category == "meal", category == "catalyst":
		return "consumo"
	case category == "animal", category == "vehicle":
		return "montaria"
	}
	return "outros"
}

func distinctValues[T any](list []T, de func(T) string) []string {
	seen := map[string]bool{}
	var outside []string
	for _, e := range list {
		if v := de(e); v != "" && !seen[v] {
			seen[v] = true
			outside = append(outside, v)
		}
	}
	slices.Sort(outside)
	return outside
}

// ── o casamento, catálogo a catálogo ─────────────────────────────────────────

// attributeOptions são os seis, na ordem da ficha e do livro — nunca alfabética.
func attributeOptions() []filterOption {
	used := map[string]bool{}
	for _, p := range book.Expertises() {
		used[p.Attribute] = true
	}
	var outside []filterOption
	for _, a := range book.AttributeOrder {
		if used[a.Key] {
			outside = append(outside, filterOption{a.Key, a.Abbreviation})
		}
	}
	return outside
}

func expertiseMatches(p book.Expertise, key, value string) bool {
	switch key {
	case "atributo":
		return p.Attribute == value
	case "treino":
		// As duas marcas do livro num filtro só: são as duas coisas que mudam
		// COMO a perícia se usa, e separá-las em duas linhas de um crachá cada
		// gastaria duas linhas para dizer o que uma diz.
		return (value == "so-treinada" && p.TrainedOnly) ||
			(value == "armadura" && p.ArmorPenalty)
	}
	return true
}

func conditionMatches(c book.Condition, key, value string) bool {
	if key == "efeito" {
		return slices.Contains(c.Tags, value)
	}
	return true
}

func spellMatches(m book.Spell, key, value string) bool {
	switch key {
	case "circulo":
		return strconv.Itoa(m.Circle) == value
	case "escola":
		return m.School == value
	case "classe":
		return slices.Contains(m.Classes, value)
	}
	return true
}

// powerMatches pelo COMEÇO da fonte: ela é "Geral · combate" e "Divino · Khalmyr",
// e o crachá diz "Geral" e "Divino". Para classe a fonte é o nome puro.
func powerMatches(p book.Power, key, value string) bool {
	if key == "fonte" {
		return p.Source == value || strings.HasPrefix(p.Source, value+" ·")
	}
	return true
}

func itemMatches(i book.Item, key, value string) bool {
	if key == "familia" {
		return familyOfCategory(i.Category) == value
	}
	return true
}

func godMatches(d book.God, key, value string) bool {
	if key == "energia" {
		return d.Energy == value
	}
	return true
}

func raceMatches(r book.Race, key, value string) bool {
	if key == "linhagem" {
		return r.Tier == value
	}
	return true
}

// applyFilters é a regra de combinação: OU dentro de um filtro, E entre eles.
//
// A ordem em que as chaves saem do mapa não importa porque tudo entre filtros é
// E — mas vale dizer, porque um dia alguém acrescenta um "ou" e a ordem passa a
// decidir o resultado.
func applyFilters[T any](list []T, chosen map[string][]string, square func(T, string, string) bool) []T {
	for key, values := range chosen {
		if len(values) == 0 {
			continue
		}
		var remaining []T
		for _, entry := range list {
			for _, value := range values {
				if square(entry, key, value) {
					remaining = append(remaining, entry)
					break
				}
			}
		}
		list = remaining
	}
	return list
}
