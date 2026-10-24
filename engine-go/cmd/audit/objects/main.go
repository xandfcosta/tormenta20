// Command objects confere as ESTATÍSTICAS DE OBJETOS contra o livro (Tab. 5-4,
// p239). ALE-423.
//
// O livro imprime quatro colunas — Tamanho, Def, RD e PV — e o motor guarda DUAS
// ESCADAS e uma coluna: a Defesa é função do tamanho ("definida por sua categoria
// de tamanho"), a RD é função do material, e sobra o PV por verbete. Então o que
// se confere são as 44 células da primeira metade contra o que o
// `domain/engine/objects.go` RECONSTRÓI delas.
//
// O material de cada verbete NÃO está impresso: "madeira" é leitura (ALE-423). O
// que se mede é o par (leitura, escada) reproduzindo a RD impressa — pega um
// degrau errado e um material trocado, e não pega os dois errados na mesma
// direção. A segunda metade (armas, armaduras, escudos) fica de fora até ter
// casa no motor.
//
// Lido por COORDENADA, e não por `-layout`: a p239 tem a Tab. 5-3 à esquerda e a
// 5-4 à direita, e o `-layout` colaria uma linha de modificador a um objeto.
//
//	go run ./cmd/audit/objects
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"
	"t20engine/domain/search"

	"t20engine/cmd/internal/transcriptdir"
	"t20engine/infra/bookpdf"
)

const pdfPage = 245 // PDF; livro = PDF − 6, então esta é a p239

// firstHalf é a faixa em y do "Pergaminho" ao "Celeiro": exclui de propósito o
// cabeçalho (y≈96), o subtítulo "Objetos Gerais" (y≈119) e a segunda metade
// (y≈300 para baixo) — medido na página.
var firstHalf = [2]float64{130.0, 295.0}

// minX é onde a coluna da DIREITA começa; à esquerda está a Tab. 5-3, que é de
// outro auditor.
const minX = 310.0

var headers = []string{"Exemplo", "Tamanho", "Def", "RD", "PV"}

const sameLine = 2.0

// fatal é o `raise SystemExit(msg)` do Python: a mensagem no stderr e saída 1.
func fatal(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

// columnCenters é o x de cada cabeçalho, lido da própria página. Falha quando
// não acha os cinco: um centro chutado espalharia as células pelas colunas
// erradas e produziria divergências com cara de descoberta (ALE-294).
func columnCenters(book *bookpdf.Book) map[string]float64 {
	found := map[string]float64{}
	for _, w := range book.Words(pdfPage) {
		if 90.0 <= w.Y && w.Y <= 105.0 && w.X >= minX && slices.Contains(headers, w.Text) {
			found[w.Text] = w.X
		}
	}
	var missing []string
	for _, h := range headers {
		if _, ok := found[h]; !ok {
			missing = append(missing, h)
		}
	}
	if len(missing) > 0 {
		fatal(fmt.Sprintf("não achei o cabeçalho %s da Tab. 5-4 na p%d do PDF. "+
			"Sem o centro de cada coluna este auditor não mede nada — e mediria "+
			"errado em silêncio se chutasse.", pyfmt.List(missing), pdfPage))
	}
	return found
}

type printedRow struct {
	name, size  string
	defense, rd int
	hitPoints   int
}

type wordAt struct {
	x    float64
	text string
}

// printedRows são as 11 linhas da primeira metade.
func printedRows(book *bookpdf.Book) []printedRow {
	centers := columnCenters(book)
	var rowKeys []float64
	byRow := map[float64]map[string][]wordAt{}
	// A ordem em que cada coluna apareceu na linha, para a mensagem de erro sair
	// como o dict do Python a imprimia.
	columnOrder := map[float64][]string{}
	for _, w := range book.Words(pdfPage) {
		if !(firstHalf[0] <= w.Y && w.Y <= firstHalf[1] && w.X >= minX) {
			continue
		}
		column := headers[0]
		for _, h := range headers[1:] {
			if abs(w.X-centers[h]) < abs(w.X-centers[column]) {
				column = h
			}
		}
		key, known := w.Y, false
		for _, k := range rowKeys {
			if abs(k-w.Y) < sameLine {
				key, known = k, true
				break
			}
		}
		if !known {
			rowKeys = append(rowKeys, key)
			byRow[key] = map[string][]wordAt{}
		}
		if _, seen := byRow[key][column]; !seen {
			columnOrder[key] = append(columnOrder[key], column)
		}
		byRow[key][column] = append(byRow[key][column], wordAt{w.X, w.Text})
	}
	sort.Float64s(rowKeys)

	var out []printedRow
	for _, y := range rowKeys {
		cells := map[string]string{}
		order := columnOrder[y]
		for _, h := range order {
			ws := byRow[y][h]
			sort.Slice(ws, func(i, j int) bool {
				if ws[i].x != ws[j].x {
					return ws[i].x < ws[j].x
				}
				return ws[i].text < ws[j].text
			})
			texts := make([]string, len(ws))
			for i, w := range ws {
				texts[i] = w.text
			}
			cells[h] = strings.Join(texts, " ")
		}
		var missing []string
		for _, h := range headers {
			if _, ok := cells[h]; !ok {
				missing = append(missing, h)
			}
		}
		if len(missing) > 0 {
			fatal(fmt.Sprintf("a linha em y=%.1f da Tab. 5-4 veio sem %s: %s. "+
				"Uma linha lida pela metade reprovaria o motor por um defeito da "+
				"LEITURA — este auditor para em vez de acusar.", y, pyfmt.List(missing), pyDict(cells, order)))
		}
		numbers := map[string]int{}
		for _, column := range []string{"Def", "RD", "PV"} {
			if !digits.MatchString(cells[column]) {
				fatal(fmt.Sprintf("a célula %s de %s veio %s, que não é número — a leitura da página mudou de forma.",
					column, pyfmt.Repr(cells["Exemplo"]), pyfmt.Repr(cells[column])))
			}
			numbers[column], _ = strconv.Atoi(cells[column])
		}
		out = append(out, printedRow{cells["Exemplo"], cells["Tamanho"], numbers["Def"], numbers["RD"], numbers["PV"]})
	}
	return out
}

var digits = regexp.MustCompile(`^[0-9]+$`)

var ladderStep = regexp.MustCompile(`"([^"]+)":\s*(-?\d+)`)

// ladder lê um `map[string]int` do arquivo da regra pelo TEXTO: as escadas são
// privadas do pacote `engine`, e é o arquivo que se audita.
func ladder(source, name string) map[string]int {
	at := strings.Index(source, "var "+name+" = map[string]int")
	if at < 0 {
		fatal(fmt.Sprintf("não achei `var %s = map[string]int` em objects.go", name))
	}
	open := at + strings.Index(source[at:], "{")
	body := source[open+1 : open+strings.Index(source[open:], "}")]
	steps := map[string]int{}
	for _, m := range ladderStep.FindAllStringSubmatch(body, -1) {
		steps[m[1]], _ = strconv.Atoi(m[2])
	}
	if len(steps) == 0 {
		fatal(fmt.Sprintf("não consegui ler nenhum degrau de `%s` em objects.go", name))
	}
	return steps
}

type engineExample struct {
	size, material string
	hitPoints      int
}

var exampleLine = regexp.MustCompile(`Name:\s*"([^"]+)",\s*Size:\s*"([^"]+)",\s*Material:\s*"([^"]+)",\s*HitPoints:\s*(\d+)`)

// engineExamples é o `objectExamplesOfTheBook`: {nome: (tamanho, material, pv)}.
func engineExamples(source string) map[string]engineExample {
	marker := "var objectExamplesOfTheBook = []ObjectExample"
	at := strings.Index(source, marker)
	if at < 0 {
		fatal("não achei `" + marker + "` em objects.go")
	}
	open := at + strings.Index(source[at:], "{")
	body := source[open : open+strings.Index(source[open:], "\n}")]
	out := map[string]engineExample{}
	for _, m := range exampleLine.FindAllStringSubmatch(body, -1) {
		hp, _ := strconv.Atoi(m[4])
		out[m[1]] = engineExample{m[2], m[3], hp}
	}
	if len(out) == 0 {
		fatal("não consegui ler nenhum exemplo de `objectExamplesOfTheBook` em objects.go — a forma " +
			"da lista mudou e este auditor mediria zero verbetes com cara de \"tudo certo\".")
	}
	return out
}

// withoutAccent é a chave como o `normalizeSizeKey` do motor a escreve.
func withoutAccent(t string) string { return search.Fold(t) }

func lookup(m map[string]int, key string) string {
	if v, ok := m[key]; ok {
		return strconv.Itoa(v)
	}
	return "None"
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		fatal(err.Error())
	}
	dir, err := transcriptdir.Find()
	if err != nil {
		fatal(err.Error())
	}
	raw, err := os.ReadFile(filepath.Join(dir, "..", "..", "engine", "objects.go"))
	if err != nil {
		fatal(err.Error())
	}
	source := string(raw)

	printed := printedRows(book)
	defenseBySize := ladder(source, "defenseByObjectSize")
	rdByMaterial := ladder(source, "damageReductionByObjectMaterial")
	examples := engineExamples(source)

	diverge, cells := 0, 0
	for _, r := range printed {
		ex, ok := examples[r.name]
		if !ok {
			fmt.Printf("  %s: está impresso na Tab. 5-4 e não está no motor\n", r.name)
			diverge++
			continue
		}
		cells++
		if withoutAccent(ex.size) != withoutAccent(r.size) {
			fmt.Printf("  %s: o livro diz %s e o motor diz %s\n", r.name, pyfmt.Repr(r.size), pyfmt.Repr(ex.size))
			diverge++
		}
		cells++
		if got := lookup(defenseBySize, withoutAccent(r.size)); got != strconv.Itoa(r.defense) {
			fmt.Printf("  %s (%s): a escada do tamanho dá Defesa %s e o livro imprime %d\n", r.name, r.size, got, r.defense)
			diverge++
		}
		cells++
		if got := lookup(rdByMaterial, withoutAccent(ex.material)); got != strconv.Itoa(r.rd) {
			fmt.Printf("  %s (%s): a escada do material dá RD %s e o livro imprime %d\n", r.name, ex.material, got, r.rd)
			diverge++
		}
		cells++
		if ex.hitPoints != r.hitPoints {
			fmt.Printf("  %s: o motor dá %d PV e o livro imprime %d\n", r.name, ex.hitPoints, r.hitPoints)
			diverge++
		}
	}

	printedNames := map[string]bool{}
	for _, r := range printed {
		printedNames[r.name] = true
	}
	var leftover []string
	for name := range examples {
		if !printedNames[name] {
			leftover = append(leftover, name)
		}
	}
	sort.Strings(leftover)
	for _, name := range leftover {
		fmt.Printf("  %s: está no motor e não achei na Tab. 5-4 impressa\n", name)
		diverge++
	}

	fmt.Printf("\nlinhas impressas lidas da p239: %d de 11 | células conferidas: %d | divergem: %d\n",
		len(printed), cells, diverge)
	fmt.Println("o MATERIAL de cada verbete não está impresso no livro — ele é leitura " +
		"(ALE-423), e o que se mede é (leitura + escada) reproduzindo a RD")
	if diverge > 0 || len(printed) != 11 {
		os.Exit(1)
	}
}
