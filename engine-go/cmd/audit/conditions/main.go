// Command conditions confere as CONDIÇÕES contra o livro (p394-395). Seção da
// ALE-391.
//
// As condições são o único catálogo cujo NÚMERO não está no verbete: o
// `transcript.Conditions` tem nome, prosa e página, e a REGRA mora no
// `conditionModifierTable` do `domain/engine/collect.go`. O que se mede é a
// HERANÇA — "Enredado. O personagem fica lento, vulnerável e sofre –2…" contra
// a tabela que referencia `lentoMods`, `vulneravelMods` —, conjunto a conjunto.
// Foi assim que o "e lento" do Cego ficou anos de fora (ALE-390).
//
// A tabela se lê por PROFUNDIDADE de parêntese, da chave até a vírgula de nível
// zero: lendo uma linha por entrada, o Cego (duas linhas) sairia herdando só
// `desprevenido`, um defeito inventado. E o leitor FALHA no identificador que
// não conhece em vez de descartá-lo — lista de proibidos subconta em silêncio.
//
//	go run ./cmd/audit/conditions
package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"t20engine/cmd/internal/pyfmt"
	"unicode"
	"unicode/utf8"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// firstPage e lastPage são o apêndice no PDF; o livro é PDF − 6.
const (
	firstPage = 400
	lastPage  = 401
)

const tableDeclaration = "var conditionModifierTable = map[string][]Modifier{"

// inheritance é o vocabulário do lado do motor: variável de modificadores → a
// condição que ela representa. O que não estiver aqui nem em `helpers` REPROVA.
var inheritance = map[string]string{
	"lentoMods": "lento", "imovelMods": "imovel", "vulneravelMods": "vulneravel",
	"desprevenidoMods": "desprevenido", "indefesoMods": "indefeso",
	"fracoMods": "fraco", "debilitadoMods": "debilitado",
}

// helpers são o que a tabela pode chamar. Identificador fora dos dois conjuntos
// é uma forma que o leitor não sabe ler, e seguir em frente produziria uma
// lista de falhas com cara de descoberta (ALE-294).
var helpers = map[string]bool{
	"juntar": true, "append": true, "Modifier": true, "condAllSkills": true, "condIntSabCar": true,
	"condAttack": true, "condSkill": true, "condByAttr": true, "condDefense": true, "condDefenseVs": true,
	"condDamageReduction": true, "condPmCost": true, "condFactor": true, "condFlag": true, "condMod": true,
	"ModifierTarget": true, "K": true, "Name": true, "Scope": true, "Attribute": true,
}

// accepted são as diferenças que NÃO são defeito, nomeadas. Uma terceira
// reprova.
var accepted = map[string]string{
	"indefeso": `o livro diz "fica desprevenido, MAS sofre −10 na Defesa" — ` +
		`o "mas" SUBSTITUI o −5, então não herdar é o certo`,
	"petrificado": "o livro manda herdar de `inconsciente`, que é `indefeso` — " +
		"o motor achata a cadeia e chega no mesmo lugar",
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dir, err := transcriptdir.Find()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	source, err := os.ReadFile(filepath.Join(dir, "..", "..", "engine", "collect.go"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	byID := map[string]transcript.Condition{}
	byName := map[string]string{}
	var ids []string
	for _, c := range transcript.Conditions {
		byID[c.ID] = c
		byName[strings.ToLower(c.Name)] = c.ID
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	blocks := bookBlocks(book)
	entries, complaints, err := readTable(string(source))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	diverge := 0
	var withoutRule []string
	for _, id := range ids {
		expr, ok := entries[id]
		if !ok {
			withoutRule = append(withoutRule, byID[id].Name)
			continue
		}
		block, ok := blocks[id]
		if !ok {
			complaints = append(complaints, byID[id].Name+": sem verbete no livro")
			continue
		}
		fromBook := inheritedInBook(block, id, byName)
		fromEngine := map[string]bool{}
		for variable, condition := range inheritance {
			if strings.Contains(expr, variable) && condition != id {
				fromEngine[condition] = true
			}
		}
		if sameSet(fromBook, fromEngine) {
			continue
		}
		if why, ok := accepted[id]; ok {
			fmt.Printf("  DIFERENÇA ACEITA %s: %s\n", byID[id].Name, why)
			continue
		}
		diverge++
		var missing []string
		for c := range fromBook {
			if !fromEngine[c] {
				missing = append(missing, c)
			}
		}
		fmt.Printf("  %s: o livro diz que ela herda %s e o motor herda %s — falta %s\n",
			byID[id].Name, pyfmt.SortedList(setKeys(fromBook)), pyfmt.SortedList(setKeys(fromEngine)), pyfmt.SortedList(missing))
	}

	fmt.Printf("\ncondições: %d | com verbete no livro: %d | com regra no motor: %d | divergem: %d\n",
		len(transcript.Conditions), len(blocks), len(entries), diverge)
	fmt.Printf("sem efeito mecânico, e por isso fora da tabela (%d): %s\n", len(withoutRule), strings.Join(withoutRule, ", "))
	for _, c := range complaints {
		fmt.Printf("  NÃO MEDIDO %s\n", c)
	}
	if diverge > 0 || len(complaints) > 0 {
		os.Exit(1)
	}
}

// bookBlocks é o texto de cada verbete, cortado no início do seguinte. O livro
// escreve o nome em Title Case seguido de ponto — "Enredado. ".
func bookBlocks(book *bookpdf.Book) map[string]string {
	var parts []string
	for page := firstPage; page <= lastPage; page++ {
		lines := book.PositionedLines(page)
		sort.SliceStable(lines, func(i, j int) bool {
			ci, cj := math.Floor(lines[i].X/200), math.Floor(lines[j].X/200)
			if ci != cj {
				return ci < cj
			}
			return lines[i].Y < lines[j].Y
		})
		var texts []string
		for _, l := range bookpdf.PositionedWithoutJunk(lines) {
			texts = append(texts, l.Text)
		}
		parts = append(parts, bookpdf.Join(texts))
	}
	whole := strings.Join(parts, " ")
	type cut struct {
		at int
		id string
	}
	var cuts []cut
	for _, c := range transcript.Conditions {
		needle := c.Name + ". "
		for pos := 0; ; {
			i := strings.Index(whole[pos:], needle)
			if i < 0 {
				break
			}
			cuts = append(cuts, cut{pos + i, c.ID})
			pos += i + len(needle)
		}
	}
	sort.Slice(cuts, func(i, j int) bool {
		if cuts[i].at != cuts[j].at {
			return cuts[i].at < cuts[j].at
		}
		return cuts[i].id < cuts[j].id
	})
	out := map[string]string{}
	for i, c := range cuts {
		end := len(whole)
		if i+1 < len(cuts) {
			end = cuts[i+1].at
		}
		if _, seen := out[c.id]; !seen {
			out[c.id] = whole[c.at:end]
		}
	}
	return out
}

// inheritedInBook são as condições que a PRIMEIRA frase do verbete diz herdar.
// Só a primeira por causa do Fatigado: a segunda diz "Se ficar fatigado
// novamente, em vez disso fica exausto", que é AGRAVAMENTO e não herança.
func inheritedInBook(block, id string, byName map[string]string) map[string]bool {
	_, afterName, _ := strings.Cut(block, ". ")
	first, _, _ := strings.Cut(afterName, ".")
	first = strings.ToLower(first)
	out := map[string]bool{}
	for name, other := range byName {
		if other != id && containsWord(first, name) {
			out[other] = true
		}
	}
	return out
}

// containsWord é o `re.search(r'\b' + nome + r'\b')` do Python, com a fronteira
// de palavra UNICODE: a do RE2 é ASCII, e "imóvel" teria fronteira no "ó".
func containsWord(text, word string) bool {
	for pos := 0; ; {
		i := strings.Index(text[pos:], word)
		if i < 0 {
			return false
		}
		i += pos
		if boundary(text, i) && boundary(text, i+len(word)) {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		pos = i + size
	}
}

// boundary é o `\b` do Python na posição `i`: uma ponta é de palavra e a outra não.
func boundary(s string, i int) bool {
	before, after := false, false
	if i > 0 {
		r, _ := utf8.DecodeLastRuneInString(s[:i])
		before = isWord(r)
	}
	if i < len(s) {
		r, _ := utf8.DecodeRuneInString(s[i:])
		after = isWord(r)
	}
	return before != after
}

func isWord(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r) }

var (
	entryKey   = regexp.MustCompile(`"([a-z-]+)":\s*`)
	stringLit  = regexp.MustCompile(`"[^"]*"`)
	identifier = regexp.MustCompile(`\b([A-Za-z][A-Za-z0-9_]*)\b`)
)

// readTable devolve a expressão de cada entrada da tabela, lida por
// profundidade de parêntese, e as queixas de identificador desconhecido.
func readTable(source string) (map[string]string, []string, error) {
	declared := strings.Index(source, tableDeclaration)
	if declared < 0 {
		return nil, nil, fmt.Errorf("não achei %q no collect.go", tableDeclaration)
	}
	body := source[declared+strings.Index(source[declared:], "{")+1:]
	entries := map[string]string{}
	var order []string
	for pos := 0; ; {
		m := entryKey.FindStringSubmatchIndex(body[pos:])
		if m == nil {
			break
		}
		key, exprStart := body[pos+m[2]:pos+m[3]], pos+m[1]
		end, depth := exprStart, 0
	scan:
		for ; end < len(body); end++ {
			switch c := body[end]; {
			case strings.ContainsRune("({[", rune(c)):
				depth++
			case strings.ContainsRune(")}]", rune(c)):
				if depth == 0 {
					break scan
				}
				depth--
			case c == ',' && depth == 0:
				break scan
			}
		}
		if _, seen := entries[key]; !seen {
			order = append(order, key)
		}
		entries[key] = body[exprStart:end]
		pos = end + 1
		if end < len(body) && strings.ContainsRune(")}]", rune(body[end])) {
			break
		}
		if pos > len(body) {
			break
		}
	}
	var complaints []string
	for _, key := range order {
		// O ARGUMENTO não é identificador: `condSkill("Iniciativa", -2)` traz o
		// nome de uma perícia. Varrer dentro das aspas faria o leitor parar em
		// nomes legítimos e não medir nada.
		withoutStrings := stringLit.ReplaceAllString(entries[key], `""`)
		for _, m := range identifier.FindAllStringSubmatch(withoutStrings, -1) {
			word := m[1]
			if _, ok := inheritance[word]; !ok && !helpers[word] {
				complaints = append(complaints, fmt.Sprintf("%s: a tabela usa %s, que este auditor não "+
					"sabe ler — ele pararia de medir a herança em silêncio", key, pyfmt.Repr(word)))
			}
		}
	}
	return entries, complaints, nil
}
