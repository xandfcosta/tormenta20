package main

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"
	"unicode"
	"unicode/utf8"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// minCoverage é o PISO de cobertura das palavras de conteúdo do livro, e ele
// foi MEDIDO: antes da ALE-405, as 27 regras trocadas ficaram em 38% ou menos e
// as 7 formas curtas fiéis em 67% ou mais. Ele existe porque números e perícias
// NÃO bastaram — os números do "Vendedor de Carcaças" coincidiram por acaso, e
// seis regras trocadas não citavam perícia nenhuma.
const minCoverage = 0.5

var (
	numberRe = regexp.MustCompile(`[+\-–−]?\d+(?:\.\d{3})*`)
	// A REMISSÃO DE PÁGINA não é número de regra: o "Estoico" fecha com "Veja as
	// regras de recuperação na página 106".
	crossRefRe = regexp.MustCompile(`(?:p[áa]g(?:ina|s?\.)?)` + bookpdf.PySpace + `*\d+`)
	// `Ofício (alquimista)` é o `Ofício` do livro com a especialização; o
	// achatado guarda só a perícia, porque é ela que o motor treina.
	specializationRe = regexp.MustCompile(bookpdf.PySpace + `*\([^)]*\)` + bookpdf.PySpace + `*$`)
	itemsRe          = regexp.MustCompile(`Itens\.` + bookpdf.PySpace + `*(.*?)\.` + bookpdf.PySpace)
	wordRe           = regexp.MustCompile(bookpdf.PyWord + `+`)
)

// numbers são os números que uma frase imprime, com o sinal normalizado: o
// travessão do livro e o hífen do catálogo são o mesmo menos.
func numbers(text string) map[string]bool {
	out := map[string]bool{}
	for _, n := range numberRe.FindAllString(crossRefRe.ReplaceAllString(text, " "), -1) {
		n = strings.NewReplacer("–", "-", "−", "-").Replace(n)
		out[strings.TrimLeft(n, "+")] = true
	}
	return out
}

func isPyWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }

// mentionsWord é o `re.search(rf'\b{nome}\b')` do Python, cujo `\b` é Unicode: o
// do RE2 é ASCII, e "Luta" casaria em "Lutação".
func mentionsWord(text, name string) bool {
	if name == "" {
		return true
	}
	first, _ := utf8.DecodeRuneInString(name)
	last, _ := utf8.DecodeLastRuneInString(name)
	for from := 0; from <= len(text); {
		i := strings.Index(text[from:], name)
		if i < 0 {
			return false
		}
		start, end := from+i, from+i+len(name)
		before, after := ' ', ' '
		if start > 0 {
			before, _ = utf8.DecodeLastRuneInString(text[:start])
		}
		if end < len(text) {
			after, _ = utf8.DecodeRuneInString(text[end:])
		}
		boundaryBefore := start == 0 && isPyWord(first) || start > 0 && isPyWord(before) != isPyWord(first)
		boundaryAfter := end == len(text) && isPyWord(last) || end < len(text) && isPyWord(after) != isPyWord(last)
		if boundaryBefore && boundaryAfter {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[start:])
		from = start + size
	}
	return false
}

func mentionedExpertises(text string, names []string) map[string]bool {
	out := map[string]bool{}
	for _, n := range names {
		if mentionsWord(text, n) {
			out[n] = true
		}
	}
	return out
}

func baseExpertise(name string) string { return specializationRe.ReplaceAllString(name, "") }

func keys(items []string) []string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = bookpdf.Key(s)
	}
	return out
}

func sortedKeys(items []string) []string {
	k := keys(items)
	sort.Strings(k)
	return k
}

// checkBothCatalogs confere as duas transcrições uma contra a outra: elas
// descrevem as MESMAS 35 origens e compartilham o `uid`, então uma divergência
// é defeito sem precisar de PDF nenhum.
func checkBothCatalogs(sources map[string]transcript.OriginSource, flat map[string]transcript.Origin) []string {
	all := map[string]bool{}
	for slug := range sources {
		all[slug] = true
	}
	for slug := range flat {
		all[slug] = true
	}
	slugs := make([]string, 0, len(all))
	for slug := range all {
		slugs = append(slugs, slug)
	}
	sort.Strings(slugs)
	var out []string
	for _, slug := range slugs {
		f, inSource := sources[slug]
		a, inFlat := flat[slug]
		if !inFlat {
			out = append(out, slug+": está em transcript.OriginSources e não em transcript.Origins")
			continue
		}
		if !inSource {
			out = append(out, slug+": está em transcript.Origins e não em transcript.OriginSources")
			continue
		}
		if f.Name != a.Name {
			out = append(out, fmt.Sprintf("%s: nome %s na fonte × %s no achatado", slug, pyfmt.Repr(f.Name), pyfmt.Repr(a.Name)))
		}
		if f.UID != a.UID {
			out = append(out, fmt.Sprintf("%s: uid %s na fonte × %s no achatado — o banco aponta para um deles", slug, f.UID, a.UID))
		}
		if f.UniquePower != a.UniquePower.Name {
			out = append(out, fmt.Sprintf("%s: poder único %s na fonte × %s no achatado", slug, pyfmt.Repr(f.UniquePower), pyfmt.Repr(a.UniquePower.Name)))
		}
		expected := []string{}
		for _, p := range f.Expertises {
			expected = append(expected, baseExpertise(p))
		}
		for _, p := range f.Powers {
			if p != f.UniquePower {
				expected = append(expected, p)
			}
		}
		// O benefício de escolha não tem nome de poder na fonte: ela o guarda
		// como `PowerChoiceCategory`, e o achatado o monta com `PowerPick`.
		found, picks := []string{}, []string{}
		for _, b := range a.Benefits {
			if b.PowerPick != "" {
				picks = append(picks, b.PowerPick)
			} else {
				found = append(found, b.Name)
			}
		}
		if !slices.Equal(sortedKeys(expected), sortedKeys(found)) {
			out = append(out, fmt.Sprintf("%s: benefícios %s no achatado × %s derivados da fonte", slug, pyfmt.List(found), pyfmt.List(expected)))
		}
		category := []string{}
		if f.PowerChoiceCategory != "" {
			category = append(category, f.PowerChoiceCategory)
		}
		sort.Strings(picks)
		if !slices.Equal(picks, category) {
			out = append(out, fmt.Sprintf("%s: escolha de poder %s no achatado × %s na fonte", slug, pyfmt.List(picks), pyfmt.List(category)))
		}
	}
	return out
}

// measured é o denominador: quantos BENEFÍCIOS se compararam, e não só quantas
// origens — uma origem que ancorou e cujo regex não casou sai verde com zero.
type measured struct {
	expertises, powers, unique, uniqueRule, page, items int
	coverage                                            []coverageRow
}

type coverageRow struct {
	ratio float64
	name  string
}

func checkList(origin transcript.OriginSource, read benefitList, m *measured) []string {
	var out []string
	m.expertises += len(read.expertises)
	if !slices.Equal(keys(read.expertises), keys(origin.Expertises)) {
		out = append(out, fmt.Sprintf("perícias: catálogo %s × livro %s", pyfmt.List(origin.Expertises), pyfmt.List(read.expertises)))
	}
	m.powers += len(read.powers)
	if !slices.Equal(keys(read.powers), keys(origin.Powers)) {
		out = append(out, fmt.Sprintf("poderes: catálogo %s × livro %s", pyfmt.List(origin.Powers), pyfmt.List(read.powers)))
	}
	if read.hasCategory != (origin.PowerChoiceCategory != "") || read.category != origin.PowerChoiceCategory {
		out = append(out, fmt.Sprintf("poderChoiceCategory: catálogo %s × livro %s",
			pyOptional(origin.PowerChoiceCategory, origin.PowerChoiceCategory != ""), pyOptional(read.category, read.hasCategory)))
	}
	return out
}

// checkUniqueRule confere os NÚMEROS, as PERÍCIAS e a COBERTURA da descrição do
// poder único. A prosa não se compara — o catálogo encurta de propósito —, mas
// uma forma mais curta PRESERVA número e perícia: perdê-los é outra regra.
func checkUniqueRule(fromBook, unique string, flat *transcript.OriginUniquePower, expertiseNames []string, m *measured) []string {
	if flat == nil {
		return []string{fmt.Sprintf("poder único %s: origins.json não traz a descrição dele", pyfmt.Repr(unique))}
	}
	if fromBook == "" {
		return []string{fmt.Sprintf("poder único %s: o verbete não traz descrição depois do título", pyfmt.Repr(unique))}
	}
	fromCatalog := flat.Description
	m.uniqueRule++
	var out []string
	if bookNumbers, catalogNumbers := numbers(fromBook), numbers(fromCatalog); !sameSet(bookNumbers, catalogNumbers) {
		out = append(out, fmt.Sprintf("%s: números %s no catálogo × %s no livro", unique, pyfmt.Sorted(catalogNumbers), pyfmt.Sorted(bookNumbers)))
	}
	bookExpertises := mentionedExpertises(fromBook, expertiseNames)
	catalogExpertises := mentionedExpertises(fromCatalog, expertiseNames)
	if !sameSet(bookExpertises, catalogExpertises) {
		out = append(out, fmt.Sprintf("%s: perícias citadas %s no catálogo × %s no livro", unique, pyfmt.Sorted(catalogExpertises), pyfmt.Sorted(bookExpertises)))
	}
	ratio, shared, total := bookpdf.Coverage(fromBook, fromCatalog)
	m.coverage = append(m.coverage, coverageRow{ratio, unique})
	if ratio < minCoverage {
		out = append(out, fmt.Sprintf("%s: o catálogo repete %d das %d palavras de conteúdo da regra do livro "+
			"(%s, piso %s) — não é uma forma mais curta, é outra regra", unique, shared, total, pyfmt.Percent(ratio), pyfmt.Percent(minCoverage)))
	}
	if len(out) > 0 {
		out = append(out, fmt.Sprintf("  %s NO LIVRO:    %s", unique, pyfmt.Repr(fromBook)))
		out = append(out, fmt.Sprintf("  %s NO CATÁLOGO: %s", unique, pyfmt.Repr(fromCatalog)))
	}
	return out
}

func sameSet(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

// checkItems confere a linha `Itens.` contra os itens iniciais pelo SUBSTANTIVO:
// a primeira palavra significativa do item do catálogo tem de aparecer na frase
// do livro. Pega item inventado e item perdido sem opinar sobre a redação.
func checkItems(origin transcript.OriginSource, body string, m *measured) []string {
	match := itemsRe.FindStringSubmatch(body)
	if match == nil {
		return []string{"itens: o verbete não imprime uma linha `Itens.` fechada em ponto"}
	}
	fromBook := bookpdf.Key(match[1])
	var out []string
	for _, item := range origin.StartingItems {
		var words []string
		for _, w := range wordRe.FindAllString(item, -1) {
			if utf8.RuneCountInString(w) > 3 {
				words = append(words, w)
			}
		}
		if len(words) == 0 {
			continue
		}
		m.items++
		if !strings.Contains(fromBook, bookpdf.Key(words[0])) {
			out = append(out, fmt.Sprintf("itens: %s — %s não aparece na linha `Itens.` do livro: %s",
				pyfmt.Repr(item), pyfmt.Repr(words[0]), pyfmt.Repr(match[1])))
		}
	}
	return out
}
