package bookpdf

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ── o ESTILO, de outro leitor ────────────────────────────────────────────────
//
// O `-bbox` do `pdftotext` não marca estilo nenhum, e neste livro o estilo é
// DADO: o título de uma magia é `Tormenta20` a 24, e o itálico de um item
// específico é a composição dele (p337). Ele vem do `pdftohtml -xml`, e o x e o
// y voltam convertidos para a escala do `-bbox` — a razão entre as alturas que
// os dois declaram —, senão quem misturasse as leituras compararia réguas
// diferentes.

var (
	htmlPageRe   = regexp.MustCompile(`<page number="\d+"[^>]*height="([\d.]+)" width="([\d.]+)"`)
	htmlTextRe   = regexp.MustCompile(`(?s)<text top="([\d.-]+)" left="([\d.-]+)"[^>]*>(.*?)</text>`)
	italicRe     = regexp.MustCompile(`(?s)<i>(.*?)</i>`)
	fontSpecRe   = regexp.MustCompile(`<fontspec id="(\d+)" size="(\d+)" family="([^"]+)"`)
	textFontRe   = regexp.MustCompile(`(?s)<text top="(-?[\d.]+)" left="(-?[\d.]+)"[^>]*font="(\d+)"[^>]*>(.*?)</text>`)
	boldLinkRe   = regexp.MustCompile(`(?s)<a href="[^"#]*#(\d+)"><b>(.*?)</b></a>`)
	diceRangeRe  = regexp.MustCompile(`^(\d{1,3})(?:-(\d{1,3}))?(?:` + pySpace + `+(.*))?$`)
	bookPriceRe  = regexp.MustCompile(`^T\$` + pySpace + `*([\d.]+)`)
	powerHeadKey = "Poder"
	prereqKey    = "Pré-requisitos"
)

func (b *Book) htmlScale(page int, xml string) (float64, bool) {
	m := htmlPageRe.FindStringSubmatch(xml)
	if m == nil {
		return 0, false
	}
	height, err := b.PageHeight(page)
	if err != nil || height == 0 {
		return 0, false
	}
	return parseFloat(m[1]) / height, true
}

func cleanHTML(s string) string { return Collapse(html.UnescapeString(tag.ReplaceAllString(s, ""))) }

// Italics são os trechos em ITÁLICO, ordenados por (y, x). O trecho vem
// QUEBRADO como a linha quebra; quem junta é quem lê.
func (b *Book) Italics(page int) []Positioned {
	xml := b.htmlXML(page)
	scale, ok := b.htmlScale(page, xml)
	if !ok {
		return nil
	}
	var out []Positioned
	for _, m := range htmlTextRe.FindAllStringSubmatch(xml, -1) {
		for _, it := range italicRe.FindAllStringSubmatch(m[3], -1) {
			if text := cleanHTML(it[1]); text != "" {
				out = append(out, Positioned{parseFloat(m[2]) / scale, parseFloat(m[1]) / scale, text})
			}
		}
	}
	sortByYX(out)
	return out
}

func sortByYX(out []Positioned) {
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].X < out[j].X
	})
}

// StyledRun é um pedaço de texto com o ESTILO em que o livro o desenhou.
type StyledRun struct {
	X, Y   float64
	Size   int
	Family string
	Text   string
}

// StyledRuns são os trechos da página com TAMANHO e FAMÍLIA do tipo, por (y, x).
//
//	for _, r := range book.StyledRuns(184) { if r.Size == 24 { … } } // as magias da p178
func (b *Book) StyledRuns(page int) []StyledRun {
	xml := b.htmlXML(page)
	scale, ok := b.htmlScale(page, xml)
	if !ok {
		return nil
	}
	type font struct {
		size   int
		family string
	}
	fonts := map[string]font{}
	for _, m := range fontSpecRe.FindAllStringSubmatch(xml, -1) {
		size, _ := strconv.Atoi(m[2])
		fonts[m[1]] = font{size, m[3]}
	}
	var out []StyledRun
	for _, m := range textFontRe.FindAllStringSubmatch(xml, -1) {
		f := fonts[m[3]]
		if text := cleanHTML(m[4]); text != "" {
			out = append(out, StyledRun{parseFloat(m[2]) / scale, parseFloat(m[1]) / scale, f.size, f.family, text})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Y != out[j].Y {
			return out[i].Y < out[j].Y
		}
		return out[i].X < out[j].X
	})
	return out
}

// BoldLink é um link em NEGRITO e a página do PDF para onde ele aponta.
type BoldLink struct {
	Text string
	Page int
}

// BoldLinks são os links em negrito da página — o índice de magias das
// p174-177. Negrito SEM link não entra: o cabeçalho de página também é negrito,
// e contá-lo inflou o índice de 198 para 203 (ALE-456).
func (b *Book) BoldLinks(page int) []BoldLink {
	var out []BoldLink
	for _, m := range boldLinkRe.FindAllStringSubmatch(b.htmlXML(page), -1) {
		if text := cleanHTML(m[2]); text != "" {
			target, _ := strconv.Atoi(m[1])
			out = append(out, BoldLink{text, target})
		}
	}
	return out
}

// ── a TABELA DE SAQUE: `d%` + nome + preço ───────────────────────────────────
//
// Cinco tabelas do capítulo de Recompensas têm a mesma forma (8-9, 8-11, 8-12,
// 8-13/14/15), e as três armadilhas delas moram aqui: a linha se monta pelo Y;
// a última vem COLAPSADA num `<line>` só ("96-100 Vingadora sagrada"); e a
// célula do d% é alinhada à DIREITA, então o x é FAIXA.

// SameLine é a tolerância de y para duas células serem a mesma linha.
const SameLine = 2.0

// SameX é a tolerância de x para duas células serem a mesma coluna da grade.
const SameX = 1.0

// LootRow é uma linha de tabela de saque. Price é nil quando a célula não tem.
type LootRow struct {
	Start, End int
	Name       string
	Price      *int
}

// LootRows lê uma tabela de saque. A célula do nome quebra em duas linhas quando
// é longa, e a de baixo não tem d% ao lado: o nome é lido até a PRÓXIMA FAIXA.
//
//	rows := book.LootRows(343, [2]float64{313, 320}, 348.7, [2]float64{480, 490}, 3)
func (b *Book) LootRows(page int, diceX [2]float64, nameX float64, priceX [2]float64, slack float64) []LootRow {
	cells := b.PositionedLines(page)
	var dices []diceCell
	var names, prices []Positioned
	for _, c := range cells {
		if diceX[0] <= c.X && c.X <= diceX[1] {
			if m := diceRangeRe.FindStringSubmatchIndex(c.Text); m != nil {
				d := diceCell{y: c.Y, start: c.Text[m[2]:m[3]]}
				if m[4] >= 0 {
					d.end, d.hasEnd = c.Text[m[4]:m[5]], true
				}
				if m[6] >= 0 {
					d.trail, d.hasTrail = c.Text[m[6]:m[7]], true
				}
				dices = append(dices, d)
			}
		}
		if abs(c.X-nameX) < slack {
			names = append(names, c)
		}
		if priceX[0] <= c.X && c.X <= priceX[1] && bookPriceRe.MatchString(c.Text) {
			prices = append(prices, c)
		}
	}
	sort.SliceStable(dices, func(i, j int) bool { return lessDice(dices[i], dices[j]) })
	sort.SliceStable(names, func(i, j int) bool {
		if names[i].Y != names[j].Y {
			return names[i].Y < names[j].Y
		}
		return names[i].Text < names[j].Text
	})
	var out []LootRow
	for i, d := range dices {
		next := 1e9
		if i+1 < len(dices) {
			next = dices[i+1].y
		}
		var body []string
		for _, n := range names {
			if d.y-SameLine < n.Y && n.Y < next-SameLine {
				body = append(body, n.Text)
			}
		}
		name, ok := d.trail, d.hasTrail && d.trail != ""
		if !ok && len(body) > 0 {
			name, ok = Join(body), true
		}
		if !ok {
			continue
		}
		var price *int
		for _, p := range prices {
			if abs(p.Y-d.y) < SameLine {
				digits := strings.ReplaceAll(bookPriceRe.FindStringSubmatch(p.Text)[1], ".", "")
				v, _ := strconv.Atoi(digits)
				price = &v
				break
			}
		}
		start, _ := strconv.Atoi(d.start)
		end := start
		if d.hasEnd && d.end != "" {
			end, _ = strconv.Atoi(d.end)
		}
		out = append(out, LootRow{start, end, Strip(name), price})
	}
	return out
}

// diceCell é a célula de d% de uma linha de tabela de saque, e a cauda quando a
// última linha vem colapsada ("96-100 Vingadora sagrada").
type diceCell struct {
	y                 float64
	start, end, trail string
	hasEnd, hasTrail  bool
}

// lessDice é a ordem de `sorted((y, grupos))` do auditor original: pelo y, e no
// empate pelos textos.
func lessDice(a, b diceCell) bool {
	if a.y != b.y {
		return a.y < b.y
	}
	if a.start != b.start {
		return a.start < b.start
	}
	if a.end != b.end {
		return a.end < b.end
	}
	return a.trail < b.trail
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// RangesThatDontClose é o denominador DE GRAÇA das tabelas de saque: o d%
// cobre 1..100, e uma leitura que perca uma linha não fecha. Vazio = fecha.
func RangesThatDontClose(rows []LootRow) string {
	covered := map[int]string{}
	for _, r := range rows {
		for d := r.Start; d <= r.End; d++ {
			if prev, ok := covered[d]; ok {
				return fmt.Sprintf("%d%% aparece duas vezes (%s e %s)", d, prev, r.Name)
			}
			covered[d] = r.Name
		}
	}
	var missing []int
	for d := 1; d <= 100; d++ {
		if _, ok := covered[d]; !ok {
			missing = append(missing, d)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("%d valores de d%% sem linha: %s…", len(missing), pyIntList(missing[:min(8, len(missing))]))
}

// pyIntList escreve uma lista de inteiros como o Python a imprime: [1, 2, 3].
func pyIntList(xs []int) string {
	parts := make([]string, len(xs))
	for i, x := range xs {
		parts[i] = strconv.Itoa(x)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ── os VERBETES por nome conhecido ───────────────────────────────────────────

// ColumnPair é uma coluna de verbetes: o x do corpo e o do recuo da primeira
// linha do parágrafo — que é justamente a que abre o verbete.
type ColumnPair struct{ Body, Indent float64 }

// Region é um retângulo (x0, x1, y0, y1) de tabela a pular.
type Region struct{ X0, X1, Y0, Y1 float64 }

// Entry é um verbete achado: o corpo, a página do LIVRO, e onde ele começa.
type Entry struct {
	Body   []string
	Page   int
	PDF    int
	Column ColumnPair
	Y0, Y1 float64
}

// Entries são os verbetes achados, por nome e na ordem em que foram achados.
type Entries struct {
	ByName map[string]*Entry
	Order  []string
}

// EntriesByName são os verbetes `Nome. Regra` das páginas, PROCURADOS por nome
// conhecido — um nome com preposição no meio ("Garrafa da Fumaça Eterna") não
// se acha com "maiúscula seguida de ponto". O título de seção corta o verbete em
// andamento, senão ele vira a última frase do anterior.
func (b *Book) EntriesByName(pages []int, columns map[int][]ColumnPair, regions map[int][]Region,
	names []string, titles []string, offset int, slack float64) Entries {
	byKey := map[string]string{}
	for _, n := range names {
		byKey[Key(n)] = n
	}
	found := Entries{ByName: map[string]*Entry{}}
	current := ""
	for _, page := range pages {
		for _, l := range b.Reading(page) {
			column, ok := ColumnOfPair(l.X, columns[page], slack)
			if !ok || InsideAny(l.X, l.Y, regions[page]) {
				continue
			}
			if startsWithAny(l.Text, titles) {
				current = ""
				continue
			}
			head, rest, dot := strings.Cut(l.Text, ". ")
			name := ""
			if dot {
				name = byKey[Key(head)]
			}
			if name != "" {
				current = name
				if _, again := found.ByName[name]; !again {
					found.Order = append(found.Order, name)
				}
				found.ByName[name] = &Entry{Body: []string{rest}, Page: page - offset, PDF: page, Column: column, Y0: l.Y, Y1: l.Y}
			} else if current != "" {
				e := found.ByName[current]
				e.Body = append(e.Body, l.Text)
				if page == e.PDF && column == e.Column {
					e.Y1 = l.Y
				}
			}
		}
	}
	return found
}

func startsWithAny(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// ColumnOfPair é o par em que esta linha está.
func ColumnOfPair(x float64, columns []ColumnPair, slack float64) (ColumnPair, bool) {
	for _, c := range columns {
		if abs(x-c.Body) < slack || abs(x-c.Indent) < slack {
			return c, true
		}
	}
	return ColumnPair{}, false
}

// InsideAny diz se o ponto cai em algum retângulo de tabela.
func InsideAny(x, y float64, regions []Region) bool {
	for _, r := range regions {
		if r.X0 <= x && x <= r.X1 && r.Y0 <= y && y <= r.Y1 {
			return true
		}
	}
	return false
}

// ── a COBERTURA de uma frase ─────────────────────────────────────────────────

// stopWords são artigo, preposição, pronome e verbo de ligação: estão em toda
// frase, e contá-los daria a qualquer par uma semelhança de base.
var stopWords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields("a o as os um uma uns umas de do da dos das em no na nos nas e ou que se por " +
		"para com sem seu sua seus suas voce ele ela isso este esta esse essa ao aos " +
		"nao mas como mesmo ate entre sobre pelo pela mais menos tambem cada qualquer " +
		"todos todas outro outra sao ser tem pode podem quando onde apenas alem disso " +
		"muito seja ja la") {
		m[w] = true
	}
	return m
}()

var alnumRun = regexp.MustCompile(`[a-z0-9]+`)

// ContentWords são as palavras que carregam a regra, sem as vazias e sem acento.
func ContentWords(text string) map[string]bool {
	out := map[string]bool{}
	for _, w := range alnumRun.FindAllString(fold(text), -1) {
		if len(w) > 2 && !stopWords[w] {
			out[w] = true
		}
	}
	return out
}

// Coverage é quanto das palavras de conteúdo do LIVRO a frase do catálogo
// repete. O denominador é o livro: a pergunta é "a regra está aqui?".
func Coverage(fromBook, fromCatalog string) (float64, int, int) {
	book := ContentWords(fromBook)
	if len(book) == 0 {
		return 1.0, 0, 0
	}
	catalog := ContentWords(fromCatalog)
	shared := 0
	for w := range book {
		if catalog[w] {
			shared++
		}
	}
	return float64(shared) / float64(len(book)), shared, len(book)
}

// ── a LEITURA em ordem, e as tabelas de poderes ──────────────────────────────

// indentStep é o degrau de indentação da tabela de poderes, medido: 10,8
// (55,0 → 65,8). A primeira linha de um parágrafo começa 13,0 à direita, e 13,0
// não é múltiplo de 10,8 — sem essa conta, a prosa vira poder.
const indentStep = 10.8

const treeLevels = 3

const minColumnGap = 40.0

// Columns são os x das colunas com separação mínima: numa página com tabela, a
// frequência sozinha devolve colunas a quatro pontos uma da outra, e a página
// sai fora de ordem sem cara de desordem.
func Columns(blocks []Block) []float64 {
	starts := ColumnStarts(blocks)
	out := []float64{starts[0]}
	for _, x := range starts[1:] {
		if x-out[len(out)-1] >= minColumnGap {
			out = append(out, x)
		}
	}
	return out
}

// Reading são as linhas da página em ordem de leitura, com o (x, y) preservado.
func (b *Book) Reading(page int) []Positioned {
	blocks := b.Blocks(page)
	starts := []float64{0}
	if len(blocks) > 0 {
		starts = Columns(blocks)
	}
	lines := PositionedWithoutJunk(b.PositionedLines(page))
	sort.SliceStable(lines, func(i, j int) bool {
		ci, cj := ColumnOf(starts, lines[i].X), ColumnOf(starts, lines[j].X)
		if ci != cj {
			return ci < cj
		}
		return lines[i].Y < lines[j].Y
	})
	return lines
}

// PowerPair é um par de colunas (Poder | Pré-requisitos) de uma tabela.
type PowerPair struct{ PowerX, PrereqX float64 }

// PowerTable é um bloco de tabela de poderes: os pares de colunas e o corpo.
type PowerTable struct {
	Pairs []PowerPair
	Body  []Positioned
}

// PowerTables são os blocos (Poder | Pré-requisitos) × 2 da página. A p126 traz
// TRÊS — Combate, Destino e Magia —, e o y do cabeçalho é o REAL: arredondá-lo
// deixava o próprio cabeçalho de fora, em silêncio.
func (b *Book) PowerTables(page int) []PowerTable {
	lines := PositionedWithoutJunk(b.PositionedLines(page))
	var ys []float64
	seen := map[float64]bool{}
	for _, l := range lines {
		if l.Text == powerHeadKey && !seen[l.Y] {
			seen[l.Y] = true
			ys = append(ys, l.Y)
		}
	}
	sort.Float64s(ys)
	var heads []float64
	for i, y := range ys {
		if i == 0 || y-ys[i-1] > SameLine {
			heads = append(heads, y)
		}
	}
	var out []PowerTable
	for i, head := range heads {
		end := 10000.0
		if i+1 < len(heads) {
			end = heads[i+1]
		}
		var band []Positioned
		for _, l := range lines {
			if head-SameLine <= l.Y && l.Y < end-SameLine {
				band = append(band, l)
			}
		}
		var powerXs, prereqXs []float64
		for _, l := range band {
			if abs(l.Y-head) < SameLine {
				switch l.Text {
				case powerHeadKey:
					powerXs = append(powerXs, l.X)
				case prereqKey:
					prereqXs = append(prereqXs, l.X)
				}
			}
		}
		sort.Float64s(powerXs)
		sort.Float64s(prereqXs)
		var pairs []PowerPair
		for j := 0; j < len(powerXs) && j < len(prereqXs); j++ {
			pairs = append(pairs, PowerPair{powerXs[j], prereqXs[j]})
		}
		var body []Positioned
		for _, l := range band {
			if l.Y > head+SameLine {
				body = append(body, l)
			}
		}
		out = append(out, PowerTable{pairs, body})
	}
	return out
}

// PowerRow é uma linha de tabela de poderes: o nome, o pré-requisito e o nível
// de indentação na árvore.
type PowerRow struct {
	Power  string
	Prereq string
	Level  int
}

// Cell é um (x, y) arredondado de célula que a tabela leu.
type Cell struct{ X, Y float64 }

// PairRows são as linhas de um par de colunas, e as células que ele consumiu.
//
// As duas colunas são GRADE, e não faixa: a do pré-requisito ia até a próxima
// coluna de poder, e nessa sobra cabia a prosa do verbete vizinho — cinco
// poderes ficaram sem verbete assim.
func PairRows(body []Positioned, powerX, prereqX float64) ([]PowerRow, map[Cell]bool) {
	steps := make([]float64, treeLevels)
	for n := range steps {
		steps[n] = powerX + float64(n)*indentStep
	}
	type cell struct {
		hasPower bool
		level    int
		power    string
		hasPre   bool
		pre      string
	}
	cells := map[float64]*cell{}
	consumed := map[Cell]bool{}
	for _, l := range body {
		level := -1
		for n, d := range steps {
			if abs(l.X-d) < SameX {
				level = n
				break
			}
		}
		y := Round1(l.Y)
		if level >= 0 {
			c := cells[y]
			if c == nil {
				c = &cell{}
				cells[y] = c
			}
			c.hasPower, c.level, c.power = true, level, l.Text
			consumed[Cell{Round1(l.X), y}] = true
		} else if abs(l.X-prereqX) < SameX {
			c := cells[y]
			if c == nil {
				c = &cell{}
				cells[y] = c
			}
			c.hasPre, c.pre = true, l.Text
			consumed[Cell{Round1(l.X), y}] = true
		}
	}
	var ys []float64
	for y := range cells {
		ys = append(ys, y)
	}
	sort.Float64s(ys)
	var out []PowerRow
	for _, y := range ys {
		c := cells[y]
		if c.hasPower {
			out = append(out, PowerRow{c.power, c.pre, c.level})
		} else if c.hasPre && len(out) > 0 {
			// A célula de pré-requisito quebra em duas linhas; sem juntar,
			// compara-se meia frase.
			last := &out[len(out)-1]
			last.Prereq = Strip(last.Prereq + " " + c.pre)
		}
	}
	return out, consumed
}

// TableCells são só os (x, y) que a tabela de poderes da página de fato LEU.
func (b *Book) TableCells(page int) map[Cell]bool {
	out := map[Cell]bool{}
	for _, t := range b.PowerTables(page) {
		for _, p := range t.Pairs {
			_, consumed := PairRows(t.Body, p.PowerX, p.PrereqX)
			for c := range consumed {
				out[c] = true
			}
		}
	}
	return out
}
