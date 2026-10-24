package bookpdf

import (
	"sort"
)

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
