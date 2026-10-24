// Package bookpdf lê o PDF do livro por COORDENADA, para os auditores do
// catálogo (`cmd/audit/...`).
//
// O que ele resolve, e por que não serve `pdftotext -layout`: o `-layout` junta
// colunas VIZINHAS na mesma linha de texto, e a linha de atributos de uma
// criatura aparece colada à criatura errada — isso já quase fez corrigir a Hidra
// AO CONTRÁRIO. E a geometria VARIA: há página de duas e de três colunas, e o
// título de seção é centralizado e atravessa a calha. As colunas se acham por
// FREQUÊNCIA do x onde o corpo começa.
//
// Precisa do `pdftotext` e do `pdftohtml` (poppler) e do PDF do livro, que é
// gitignorado — ele não é nosso para distribuir.
//
//	book, err := bookpdf.Open()
//	lines := book.PageLines(184)
package bookpdf

import (
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Book é o PDF aberto, com a saída de cada ferramenta guardada por página.
type Book struct {
	Path  string
	mu    sync.Mutex
	cache map[string]string
}

// Open acha o livro: `T20_BOOK_PDF`, `LIVRO_PDF`, um `t20-book.pdf` subindo do
// diretório atual, ou o do checkout principal quando se está numa worktree.
//
// Um auditor que caísse em silêncio num PDF vazio reportaria tudo "não medido"
// com cara de resultado; por isso não achar é ERRO, com o caminho dito.
func Open() (*Book, error) {
	for _, env := range []string{"T20_BOOK_PDF", "LIVRO_PDF"} {
		if p := os.Getenv(env); p != "" {
			if _, err := os.Stat(p); err != nil {
				return nil, fmt.Errorf("não achei o livro em %s (%s): %w", p, env, err)
			}
			return &Book{Path: p, cache: map[string]string{}}, nil
		}
	}
	if p := findUpwards("t20-book.pdf"); p != "" {
		return &Book{Path: p, cache: map[string]string{}}, nil
	}
	if p := mainCheckoutBook(); p != "" {
		return &Book{Path: p, cache: map[string]string{}}, nil
	}
	return nil, fmt.Errorf("não achei o t20-book.pdf subindo do diretório atual nem no checkout " +
		"principal. Ele é gitignorado — aponte o T20_BOOK_PDF para o PDF")
}

func findUpwards(name string) string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// mainCheckoutBook é o livro do checkout principal: numa worktree o catálogo a
// auditar é o de lá, e o livro continua no principal.
func mainCheckoutBook() string {
	out, err := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(strings.TrimSpace(string(out))), "t20-book.pdf")
	if _, err := os.Stat(candidate); err != nil {
		return ""
	}
	return candidate
}

// run executa uma ferramenta do poppler sobre uma página, uma vez por processo.
func (b *Book) run(tool string, page int, args ...string) string {
	key := tool + " " + strings.Join(args, " ") + " " + strconv.Itoa(page)
	b.mu.Lock()
	cached, ok := b.cache[key]
	b.mu.Unlock()
	if ok {
		return cached
	}
	p := strconv.Itoa(page)
	var cmd *exec.Cmd
	if tool == "pdftohtml" {
		cmd = exec.Command(tool, append(append([]string{}, args...), "-f", p, "-l", p, "-stdout", b.Path)...)
	} else {
		cmd = exec.Command(tool, append(append([]string{}, args...), "-f", p, "-l", p, b.Path, "-")...)
	}
	out, _ := cmd.Output()
	b.mu.Lock()
	b.cache[key] = string(out)
	b.mu.Unlock()
	return string(out)
}

func (b *Book) bboxLayout(page int) string { return b.run("pdftotext", page, "-bbox-layout") }
func (b *Book) htmlXML(page int) string    { return b.run("pdftohtml", page, "-xml", "-i") }

// ── as classes do Python ─────────────────────────────────────────────────────
//
// Os auditores nasceram em Python, onde `\s`, `\w` e `strip()` são UNICODE; no
// RE2 do Go `\s` e `\w` são ASCII. O livro tem espaço não separável e letra
// acentuada, e trocar a classe trocaria o que casa sem erro nenhum.

const pySpace = `[\t\n\v\f\r \x{1c}-\x{1f}\x{85}\p{Z}]`
const pyWord = `[\p{L}\p{N}_]`

var spaces = regexp.MustCompile(pySpace + `+`)

// Collapse troca cada sequência de espaço por um espaço só, e apara as pontas.
func Collapse(s string) string { return Strip(spaces.ReplaceAllString(s, " ")) }

// Strip apara o espaço das pontas, com a definição de espaço do Python.
func Strip(s string) string {
	return strings.TrimFunc(s, isPySpace)
}

func isPySpace(r rune) bool {
	return unicode.IsSpace(r) || unicode.Is(unicode.Z, r) || (r >= 0x1c && r <= 0x1f)
}

// Round1 é o `round(x, 1)` do Python: o arredondamento CORRETO do valor binário,
// com empate para o par — o mesmo que o `FormatFloat` faz.
func Round1(x float64) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'f', 1, 64), 64)
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// ── blocos e linhas ──────────────────────────────────────────────────────────

var (
	blockRe = regexp.MustCompile(`(?s)<block xMin="([\d.]+)" yMin="([\d.]+)" xMax="([\d.]+)"[^>]*>(.*?)</block>`)
	lineRe  = regexp.MustCompile(`(?s)<line[^>]*>(.*?)</line>`)
	wordRe  = regexp.MustCompile(`(?s)<word xMin="([\d.]+)" yMin="([\d.]+)"[^>]*>(.*?)</word>`)
)

// Block é um bloco que o PDF declara: onde começa e termina em x, a altura da
// primeira linha, e as linhas.
type Block struct {
	X0, X1, Y float64
	Lines     []string
}

// Positioned é um texto com o (x, y) de onde ele começa.
type Positioned struct {
	X, Y float64
	Text string
}

// Blocks são os blocos da página, na ordem do PDF.
func (b *Book) Blocks(page int) []Block {
	var out []Block
	for _, m := range blockRe.FindAllStringSubmatch(b.bboxLayout(page), -1) {
		x0, yBlock, x1 := parseFloat(m[1]), parseFloat(m[2]), parseFloat(m[3])
		var lines []string
		y0, haveY := 0.0, false
		for _, lm := range lineRe.FindAllStringSubmatch(m[4], -1) {
			var words []Positioned
			for _, w := range wordRe.FindAllStringSubmatch(lm[1], -1) {
				words = append(words, Positioned{parseFloat(w[1]), parseFloat(w[2]), html.UnescapeString(w[3])})
			}
			if len(words) == 0 {
				continue
			}
			if !haveY {
				y0, haveY = words[0].Y, true
			}
			sort.SliceStable(words, func(i, j int) bool { return words[i].X < words[j].X })
			texts := make([]string, len(words))
			for i, w := range words {
				texts[i] = w.Text
			}
			lines = append(lines, strings.Join(texts, " "))
		}
		if len(lines) == 0 {
			continue
		}
		if !haveY {
			y0 = yBlock
		}
		out = append(out, Block{X0: x0, X1: x1, Y: y0, Lines: lines})
	}
	return out
}

// minBlocksPerColumn é quantos blocos começam num x para ele ser coluna.
const minBlocksPerColumn = 3

// ColumnStarts são os x onde as colunas COMEÇAM, achados por frequência.
//
// Não serve fundir faixas de x: o título de seção é centralizado e atravessa a
// calha, e uma única linha dessas funde as duas colunas numa só — foi assim que
// o Orc e o Glop saíram interfoliados.
func ColumnStarts(blocks []Block) []float64 {
	count := map[float64]int{}
	for _, b := range blocks {
		count[Round1(b.X0)]++
	}
	var starts []float64
	for x, n := range count {
		if n >= minBlocksPerColumn {
			starts = append(starts, x)
		}
	}
	sort.Float64s(starts)
	if len(starts) > 0 {
		return starts
	}
	if len(blocks) == 0 {
		return []float64{0}
	}
	least := blocks[0].X0
	for _, b := range blocks[1:] {
		least = min(least, b.X0)
	}
	return []float64{least}
}

// ColumnOf é a coluna de um bloco que começa em `x0`: tudo que começa mais à
// DIREITA de um início — o "ND 1/4" alinhado à direita — é da coluna anterior.
func ColumnOf(starts []float64, x0 float64) int {
	column := 0
	for i, start := range starts {
		if x0 >= start-2 {
			column = i
		}
	}
	return column
}

// PageLines são as linhas na ordem de LEITURA: coluna por coluna, de cima para
// baixo.
func (b *Book) PageLines(page int) []string {
	blocks := b.Blocks(page)
	if len(blocks) == 0 {
		return nil
	}
	starts := ColumnStarts(blocks)
	sort.SliceStable(blocks, func(i, j int) bool {
		ci, cj := ColumnOf(starts, blocks[i].X0), ColumnOf(starts, blocks[j].X0)
		if ci != cj {
			return ci < cj
		}
		return blocks[i].Y < blocks[j].Y
	})
	var out []string
	for _, bl := range blocks {
		out = append(out, bl.Lines...)
	}
	return out
}

// junk é a MARCA D'ÁGUA e os cabeçalhos correntes, que entram na leitura como
// conteúdo. A d'água cola no texto vizinho: "Golpista Divino. Mateus Santos
// mateush.santos42@gmail.com" chegou como o nome de um poder.
var junk = regexp.MustCompile(`Mateus Santos|mateush\.santos|^Capítulo|^\d{1,3}$|fim de coluna|Construção de Personagem`)

// IsJunk diz se a linha é marca d'água ou cabeçalho corrente.
func IsJunk(text string) bool { return text == "" || junk.MatchString(text) }

// WithoutJunk tira das linhas a marca d'água e os cabeçalhos.
func WithoutJunk(lines []string) []string {
	var out []string
	for _, l := range lines {
		if !IsJunk(l) {
			out = append(out, l)
		}
	}
	return out
}

// PositionedWithoutJunk é o `WithoutJunk` das linhas com coordenada.
func PositionedWithoutJunk(lines []Positioned) []Positioned {
	var out []Positioned
	for _, l := range lines {
		if !IsJunk(l.Text) {
			out = append(out, l)
		}
	}
	return out
}

// Num lê "+2", "–1" e "−1": o livro escreve o negativo com TRAVESSÃO.
func Num(t string) (int, error) {
	s := Strip(t)
	for _, dash := range []string{"–", "−", "—"} {
		s = strings.ReplaceAll(s, dash, "-")
	}
	return strconv.Atoi(strings.ReplaceAll(s, "+", ""))
}

var hyphenBreak = regexp.MustCompile(`(` + pyWord + `)-` + pySpace + `+(` + pyWord + `)`)

// Join junta linhas DESFAZENDO a hifenização de quebra ("subter-" / "râneos"),
// senão um `find` pelo nome de uma habilidade não acha a habilidade que está lá.
func Join(lines []string) string {
	return hyphenBreak.ReplaceAllString(strings.Join(lines, " "), "$1$2")
}

// fold tira o acento e minuscula: NFD, sem as marcas combinantes.
func fold(s string) string {
	var out strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		if !unicode.Is(unicode.Mn, r) {
			out.WriteRune(r)
		}
	}
	return out.String()
}

var notAlnum = regexp.MustCompile(`[^a-z0-9]+`)

// Key normaliza para casar o nome do livro com o do catálogo.
func Key(name string) string { return notAlnum.ReplaceAllString(fold(name), "") }

var notAlnumSpace = regexp.MustCompile(`[^a-z0-9 ]+`)

// NormalizePhrase é o texto comparável: sem acento, sem pontuação, sem espaço
// duplicado. A comparação é por CONTEÚDO — o catálogo reescreve as frases do
// livro em forma mais curta de propósito.
func NormalizePhrase(t string) string {
	return Collapse(notAlnumSpace.ReplaceAllString(fold(t), " "))
}

// Words são as PALAVRAS da página com (x, y), um degrau abaixo das linhas: na
// p119 um `<line>` traz o título de uma perícia e o da coluna VIZINHA juntos, e
// palavra por palavra a coluna se decide pelo x de cada uma.
func (b *Book) Words(page int) []Positioned {
	var out []Positioned
	for _, w := range wordRe.FindAllStringSubmatch(b.bboxLayout(page), -1) {
		out = append(out, Positioned{parseFloat(w[1]), parseFloat(w[2]), html.UnescapeString(w[3])})
	}
	return out
}

var positionedLineRe = regexp.MustCompile(`(?s)<line xMin="([\d.]+)" yMin="([\d.]+)"[^>]*>(.*?)</line>`)
var tag = regexp.MustCompile(`<[^>]+>`)

// PositionedLines são as LINHAS da página com (x, y), sem agrupar por bloco.
//
// Numa tabela, a coluna do nível e a da habilidade são blocos DIFERENTES, e o
// que diz que "6º" e "Fúria +3" são a mesma linha é o y das duas ser igual.
// Parear por índice erra na primeira que quebra em duas linhas (ALE-392).
func (b *Book) PositionedLines(page int) []Positioned {
	var out []Positioned
	for _, m := range positionedLineRe.FindAllStringSubmatch(b.bboxLayout(page), -1) {
		text := Collapse(html.UnescapeString(tag.ReplaceAllString(m[3], " ")))
		if text != "" {
			out = append(out, Positioned{parseFloat(m[1]), parseFloat(m[2]), text})
		}
	}
	return out
}

// PageHeight é a altura em pontos, lida do mesmo `-bbox` que o resto lê.
func (b *Book) PageHeight(page int) (float64, error) {
	m := pageSizeRe.FindStringSubmatch(b.run("pdftotext", page, "-bbox"))
	if m == nil {
		return 0, fmt.Errorf("a página %d não declarou tamanho", page)
	}
	return parseFloat(m[2]), nil
}

var pageSizeRe = regexp.MustCompile(`<page width="([\d.]+)" height="([\d.]+)"`)
