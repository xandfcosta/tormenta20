// Command classes confere as CLASSES do app contra o livro, poder por poder
// (ALE-392).
//
// Rodar é ATO DELIBERADO: a ferramenta PROPÕE, o diff se revisa contra o PDF, e
// é a revisão que decide. Ela não escreve.
//
//	go run ./cmd/audit/classes                  # relatório
//	go run ./cmd/audit/classes --classe Bárbaro
//
// O livro escreve poder de classe de DUAS formas, com âncoras diferentes: a
// PROSA (`• Nome. texto`), ancorada pelo NOME na faixa de páginas da classe; e o
// DEGRAU de tabela (`6º · Fúria +3`), ancorado pelo NÍVEL — a mais forte, porque
// o catálogo também afirma o número (`grantedAtLevel`). Os poderes que produzem
// número são quase todos degraus: um auditor só de prosa mediria 14 de 49, que é
// o verde sobre nada.
//
// A heurística de modificador: o ALVO tem de ser nomeado no texto do livro — foi
// assim que a Fúria foi pega concedendo Fortitude e Vontade que o livro não dá
// (ALE-388). O falso positivo CONHECIDO é o poder que se define por REFERÊNCIA a
// outro ("O bônus de Pele de Ferro aumenta para +8"), que sai marcado à parte.
//
// Precisa do `pdftotext` (poppler) e do PDF do livro.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// firstPageOfClass sai do SUMÁRIO do livro, lido por coordenada. Classes vão de
// 32 a 84 — Origens começam em 85. A ordem é a das páginas.
var firstPageOfClass = []struct {
	name string
	page int
}{
	{"Arcanista", 36}, {"Bárbaro", 40}, {"Bardo", 43}, {"Bucaneiro", 46}, {"Caçador", 49},
	{"Cavaleiro", 52}, {"Clérigo", 56}, {"Druida", 60}, {"Guerreiro", 64}, {"Inventor", 67},
	{"Ladino", 72}, {"Lutador", 75}, {"Nobre", 78}, {"Paladino", 81},
}

const afterTheClasses = 85
const pdfOffset = 6 // página do PDF = página do livro + 6

type pageRange struct{ first, last int }

func classRanges() map[string]pageRange {
	out := map[string]pageRange{}
	for i, c := range firstPageOfClass {
		last := afterTheClasses - 1
		if i+1 < len(firstPageOfClass) {
			last = firstPageOfClass[i+1].page - 1
		}
		out[c.name] = pageRange{c.page, last}
	}
	return out
}

// classPowerGrants são as CONCESSÕES de poder de classe com a regra do verbete
// já juntada. Desde a ALE-403 há duas espécies de linha: o VERBETE, sem
// `className`, que carrega a regra uma vez; e a CONCESSÃO, com classe e nível,
// apontando por `powerUid`. O auditor pergunta "a classe X concede Y no nível
// Z?", então lê concessões — mas o NOME mora no verbete. O que a concessão
// declara vence o verbete, campo a campo.
func classPowerGrants() []transcript.ClassPower {
	entryByUID := map[string]transcript.ClassPower{}
	for _, p := range transcript.ClassPowers {
		if p.ClassName == "" {
			entryByUID[p.UID] = p
		}
	}
	var out []transcript.ClassPower
	for _, p := range transcript.ClassPowers {
		if p.ClassName == "" {
			continue
		}
		entry, ok := entryByUID[p.PowerUID]
		if !ok || p.PowerUID == "" {
			out = append(out, p)
			continue
		}
		merged := p
		if merged.Name == "" {
			merged.Name = entry.Name
		}
		if merged.GrantedAtLevel == 0 {
			merged.GrantedAtLevel = entry.GrantedAtLevel
		}
		if merged.Modifiers == nil {
			merged.Modifiers = entry.Modifiers
		}
		out = append(out, merged)
	}
	return out
}

var (
	levelCell = regexp.MustCompile(`^(\d{1,2})º$`)
	space     = `[\t\n\v\f\r \x{1c}-\x{1f}\x{85}\p{Z}]`
	proseHead = regexp.MustCompile(`^(?:•` + space + `*)?([A-ZÁÉÍÓÚÂÊÔÃÕÇ][^.]{2,44})\.` + space + `+(.*)$`)
	// reference é o poder que se define por referência a outro: o alvo não é
	// nomeado, e está certo que não seja. A borda de palavra é checada à mão
	// em `mentionsReference`, porque o `\b` do RE2 é ASCII e o do Python não.
	reference = regexp.MustCompile(`(?i)(b[oô]nus de|aumenta para|passa a ser|em vez disso)`)
	wordRune  = regexp.MustCompile(`^[\p{L}\p{N}_]$`)
)

var attributeName = map[string]string{"strength": "Força", "dexterity": "Destreza", "constitution": "Constituição",
	"intelligence": "Inteligência", "wisdom": "Sabedoria", "charisma": "Carisma"}

var targetTerms = map[string][]string{
	"attack": {"ataque"}, "damage": {"dano"}, "defense": {"Defesa"},
	"maxPv": {"pontos de vida", "PV"}, "maxPm": {"pontos de mana", "PM"},
	"pmCost": {"PM"}, "spellDC": {"CD"}, "damageReduction": {"redução de dano"},
	"displacement": {"deslocamento"}, "pmLimit": {"PM"}, "tempHp": {"pontos de vida temporários"},
}

func termsOfTarget(m transcript.Modifier) []string {
	switch m.Target.Kind {
	case "expertise":
		return []string{m.Target.Name}
	case "attribute":
		return []string{attributeName[m.Target.Attribute]}
	}
	return targetTerms[m.Target.Kind]
}

func isWordAt(s string, i int, before bool) bool {
	if before {
		if i == 0 {
			return false
		}
		r := []rune(s[:i])
		return wordRune.MatchString(string(r[len(r)-1]))
	}
	if i >= len(s) {
		return false
	}
	r := []rune(s[i:])
	return wordRune.MatchString(string(r[0]))
}

func mentionsReference(text string) bool {
	for _, loc := range reference.FindAllStringIndex(text, -1) {
		if !isWordAt(text, loc[0], true) && !isWordAt(text, loc[1], false) {
			return true
		}
	}
	return false
}

func main() {
	only := flag.String("classe", "", "auditar só uma")
	flag.Parse()

	ranges := classRanges()
	var target []string
	if *only != "" {
		if _, ok := ranges[*only]; !ok {
			fmt.Fprintf(os.Stderr, "não conheço a classe %q; as classes são as do capítulo 1 (Arcanista … Paladino)\n", *only)
			os.Exit(1)
		}
		target = []string{*only}
	} else {
		for _, c := range firstPageOfClass {
			target = append(target, c.name)
		}
	}
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	powers := classPowerGrants()

	tables := map[string]levelTable{}
	proses := map[string]map[string][]string{}
	for _, c := range target {
		r := ranges[c]
		tables[c] = classTable(book, c, r.first, r.last)
		proses[c] = classProse(book, r.first, r.last)
	}
	for _, c := range target {
		if len(tables[c].levels) != 20 {
			fmt.Printf("ATENÇÃO %s: a tabela leu %d níveis e o livro tem 20 — "+
				"o relatório abaixo está medindo menos do que diz.\n", c, len(tables[c].levels))
		}
	}

	type finding struct{ kind, class, name, what string }
	levelOK, levelDiverges, notInTable := 0, 0, 0
	inProse, noAnchor := 0, 0
	var divergent, references []finding
	for _, p := range powers {
		c := p.ClassName
		if !slices.Contains(target, c) {
			continue
		}
		k := bookpdf.Key(p.Name)

		if p.GrantedAtLevel != 0 {
			var found []int
			for _, n := range tables[c].levels {
				if strings.Contains(bookpdf.Key(tables[c].text[n]), k) {
					found = append(found, n)
				}
			}
			switch {
			case len(found) == 0:
				notInTable++
			case slices.Contains(found, p.GrantedAtLevel):
				levelOK++
			default:
				levelDiverges++
				divergent = append(divergent, finding{"NÍVEL", c, p.Name,
					fmt.Sprintf("catálogo=%d livro=%s", p.GrantedAtLevel, intList(found))})
			}
		}

		body, ok := proses[c][k]
		if !ok {
			noAnchor++
			continue
		}
		inProse++
		text := strings.Join(body, " ")
		normalized := bookpdf.NormalizePhrase(text)
		for _, m := range p.Modifiers {
			var terms []string
			for _, t := range termsOfTarget(m) {
				if t != "" {
					terms = append(terms, t)
				}
			}
			if len(terms) == 0 || slices.ContainsFunc(terms, func(t string) bool {
				return strings.Contains(normalized, bookpdf.NormalizePhrase(t))
			}) {
				continue
			}
			f := finding{"ALVO", c, p.Name, fmt.Sprintf("%s '%s' %s não é nomeado no livro",
				m.Target.Kind, terms[0], signed(m.Amount))}
			if mentionsReference(text) {
				references = append(references, f)
			} else {
				divergent = append(divergent, f)
			}
		}
	}

	withLevel, ofTarget := 0, 0
	for _, p := range powers {
		if slices.Contains(target, p.ClassName) {
			ofTarget++
			if p.GrantedAtLevel != 0 {
				withLevel++
			}
		}
	}
	fmt.Printf("poderes no catálogo:        %d\n", ofTarget)
	fmt.Printf("  ancorados na prosa:       %d\n", inProse)
	fmt.Printf("  sem âncora na prosa:      %d  (degraus de tabela, na maioria)\n", noAnchor)
	fmt.Printf("com grantedAtLevel:         %d\n", withLevel)
	fmt.Printf("  nível confere:            %d\n", levelOK)
	fmt.Printf("  nível DIVERGE:            %d\n", levelDiverges)
	fmt.Printf("  não nomeado na tabela:    %d\n", notInTable)
	fmt.Printf("\ndivergências de ALVO/NÍVEL: %d\n", len(divergent))
	for _, f := range divergent {
		fmt.Printf("  %-6s %-11s %-36s %s\n", f.kind, f.class, pyfmt.FirstRunes(f.name, 34), f.what)
	}
	fmt.Printf("\npor REFERÊNCIA a outro poder (falso positivo conhecido): %d\n", len(references))
	for _, f := range references {
		fmt.Printf("         %-11s %-36s %s\n", f.class, pyfmt.FirstRunes(f.name, 34), f.what)
	}
}

// signed é o `:+d` do Python. O catálogo guarda o valor como float64, e todo
// modificador de classe é inteiro.
func signed(x float64) string {
	s := strconv.FormatFloat(x, 'f', -1, 64)
	if x >= 0 {
		return "+" + s
	}
	return s
}

func intList(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
