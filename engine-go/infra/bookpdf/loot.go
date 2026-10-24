package bookpdf

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

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
