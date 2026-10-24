package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"t20engine/infra/bookpdf"
)

// O livro impresso está 6 páginas atrás do PDF; a conta é do
// `engine-go/CLAUDE.md`, que é o dono dela.
const (
	pdfOffset = 6
	slack     = 3.0
)

// diceRange aceita CAUDA porque a última linha de cada tabela vem colapsada num
// `<line>` só — "91-100 Arma específica Veja a Tabela 8-9".
var diceRange = regexp.MustCompile(`^(\d{1,3})(?:-(\d{1,3}))?(?:` + bookpdf.PySpace + `+(.*))?$`)

// entryHead é o nome em negrito seguido de ponto. A maiúscula na primeira letra
// separa "Flamejante. A arma…" de uma frase que termine em nome próprio.
var entryHead = regexp.MustCompile(`^([A-ZÁÉÍÓÚÂÊÔÃÕÇ][a-záéíóúâêôãõç]+)\. (.+)$`)

// footnoteCall é a chamada de rodapé, que vem em célula PRÓPRIA (Animado e
// Guardião da 8-10), GRUDADA no nome ("Esmagador 1") ou como o asterisco da
// 8-8. Nenhum encanto termina em dígito; se um dia terminar, as duas âncoras
// discordam e o auditor para antes do catálogo.
var footnoteCall = regexp.MustCompile(`(?:[*]|` + bookpdf.PySpace + `)*\d?$`)

// tableRow é uma linha da tabela: a faixa de d%, o nome e o efeito resumido.
type tableRow struct {
	start, end   int
	name, effect string
}

type diceCell struct {
	y                 float64
	start, end, trail string
}

// readTable lê a tabela do perfil. A linha se monta pelo Y — as três células
// são blocos DIFERENTES, e parear por índice erra na primeira célula que quebra
// em duas linhas (a da Caçadora).
func readTable(book *bookpdf.Book, p profile) []tableRow {
	cells := book.PositionedLines(p.tablePage)
	var dices []diceCell
	var names, effects []bookpdf.Positioned
	for _, c := range cells {
		if p.diceX[0] <= c.X && c.X <= p.diceX[1] {
			if m := diceRange.FindStringSubmatch(c.Text); m != nil {
				dices = append(dices, diceCell{c.Y, m[1], m[2], m[3]})
			}
		}
	}
	for _, c := range cells {
		if abs(c.X-p.nameX) < slack {
			names = append(names, c)
		}
		if abs(c.X-p.effectX) < slack {
			effects = append(effects, c)
		}
	}
	sort.SliceStable(dices, func(i, j int) bool {
		a, b := dices[i], dices[j]
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
	})
	sort.SliceStable(effects, func(i, j int) bool {
		if effects[i].Y != effects[j].Y {
			return effects[i].Y < effects[j].Y
		}
		return effects[i].Text < effects[j].Text
	})

	var rows []tableRow
	for _, d := range dices {
		start, _ := strconv.Atoi(d.start)
		end := start
		if d.end != "" {
			end, _ = strconv.Atoi(d.end)
		}
		// A linha colapsada traz nome e efeito na própria cauda, e os dois se
		// separam por ela ser a última: "Arma específica Veja a Tabela 8-9".
		if d.trail != "" {
			name, effect, _ := strings.Cut(d.trail, " Veja ")
			rows = append(rows, tableRow{start, end, bookpdf.Strip(name), bookpdf.Strip("Veja " + effect)})
			continue
		}
		name, found := "", false
		for _, n := range names {
			if abs(n.Y-d.y) < bookpdf.SameLine {
				name, found = n.Text, true
				break
			}
		}
		if !found {
			continue
		}
		// O efeito vai desta linha até a próxima: a célula que quebra em duas
		// tem a continuação SEM d% ao lado.
		next := 1e9
		for _, other := range dices {
			if other.y > d.y+bookpdf.SameLine {
				next = other.y
				break
			}
		}
		var body []string
		for _, e := range effects {
			if d.y-bookpdf.SameLine < e.Y && e.Y < next-bookpdf.SameLine {
				body = append(body, e.Text)
			}
		}
		name, effect := splitNameAndEffect(name, bookpdf.Join(body))
		rows = append(rows, tableRow{start, end, name, effect})
	}
	return rows
}

// splitNameAndEffect separa nome e efeito quando a página os entrega na MESMA
// célula ("Fantasmagórico Lança Manto de Sombras", duas linhas da 8-10). O corte
// é na primeira palavra porque todo encanto tem nome de uma palavra só; se um
// dia não tiver, o nome cortado não casa com o verbete e o auditor para.
func splitNameAndEffect(cell, effect string) (string, string) {
	name := bookpdf.Strip(footnoteCall.ReplaceAllString(cell, ""))
	if effect != "" || !strings.Contains(name, " ") {
		return name, effect
	}
	first, rest, _ := strings.Cut(name, " ")
	return first, rest
}

// rangesThatDontClose é a SOMA antes das parcelas: as faixas ladrilham 1..100,
// e é o único controle que pega uma linha SUMIDA.
func rangesThatDontClose(rows []tableRow) string {
	expected := 1
	for _, r := range rows {
		if r.start != expected {
			return fmt.Sprintf("a faixa de %s começa em %d e a anterior terminou em %d: falta linha entre as duas", r.name, r.start, expected-1)
		}
		expected = r.end + 1
	}
	if expected != 101 {
		return fmt.Sprintf("as faixas terminam em %d e o d%% vai até 100", expected-1)
	}
	return ""
}

// bookEntry é a regra inteira de um verbete e a página do LIVRO dele.
type bookEntry struct {
	rule string
	page int
}

// readEntries são os verbetes `Nome. Regra`: um termina onde o seguinte
// começa, e a leitura é por COLUNA porque a prosa divide a página com a tabela.
func readEntries(book *bookpdf.Book, p profile) map[string]bookEntry {
	bodies := map[string][]string{}
	pageOf := map[string]int{}
	current := ""
	for _, page := range p.entryPages {
		prose := p.proseColumns[page]
		limit, hasLimit := p.tableRegion[page]
		for _, l := range book.Reading(page) {
			if !nearAny(l.X, prose) {
				continue
			}
			if hasLimit && l.X < limit[0] && l.Y < limit[1] {
				continue
			}
			if p.entriesEnd != "" && strings.HasPrefix(l.Text, p.entriesEnd) {
				current = ""
				continue
			}
			m := entryHead.FindStringSubmatch(l.Text)
			if m != nil {
				if _, seen := bodies[m[1]]; !seen {
					current = m[1]
					bodies[current] = []string{m[2]}
					pageOf[current] = page - pdfOffset
					continue
				}
			}
			if current != "" {
				bodies[current] = append(bodies[current], l.Text)
			}
		}
	}
	out := map[string]bookEntry{}
	for name, body := range bodies {
		out[name] = bookEntry{bookpdf.Join(body), pageOf[name]}
	}
	return out
}

func nearAny(x float64, columns []float64) bool {
	for _, c := range columns {
		if abs(x-c) < slack {
			return true
		}
	}
	return false
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
