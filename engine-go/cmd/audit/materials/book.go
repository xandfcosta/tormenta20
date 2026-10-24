package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"t20engine/infra/bookpdf"
)

const (
	pdfOffset  = 6
	tablePage  = 173
	tableTop   = 590.0 // o y do título da Tabela 3-9, no rodapé da p167: é o piso
	itemKindX  = 79.4
	slack      = 3.0
	headerFoot = 620.0
	noPrice    = "—"
	opening    = "abertura"
)

var entryPages = []int{172, 173}

// itemKinds são as cinco linhas da matriz, na ordem impressa, e o vocabulário
// do `appliesTo` — a tradução mora aqui e não no leitor.
var itemKinds = []struct{ printed, family, priceKey string }{
	{"Arma", "weapon", "weapon"},
	{"Armadura leve", "armor", "armorLight"},
	{"Armadura pesada", "armor", "armorHeavy"},
	{"Escudo", "shield", "shield"},
	{"Esotéricos", "esoteric", "esoteric"},
}

func isItemKind(t string) bool {
	for _, k := range itemKinds {
		if k.printed == t {
			return true
		}
	}
	return false
}

var priceCell = regexp.MustCompile(`^\+ T\$ ([\d.]+)$`)

// cell é uma célula da matriz: o material (coluna) e o tipo de item (linha).
type cell struct{ material, kind string }

// priceMatrix é a Tabela 3-9: os nomes em ordem, as células (nil = travessão)
// e quantas linhas de tipo de item ela tem.
type priceMatrix struct {
	names  []string
	prices map[cell]*int
	rows   int
}

// readMatrix lê a Tabela 3-9. As colunas se acham pela linha `Arma`, a única
// sem travessão: os x das seis células dela são as âncoras, e toda outra célula
// vai para a mais próxima. Achar coluna pelo CABEÇALHO não serve — "Aço-Rubi
// Adamante" chega num `<line>` só, com os dois nomes colados.
func readMatrix(book *bookpdf.Book, page int) (priceMatrix, error) {
	var lines []bookpdf.Positioned
	for _, l := range book.PositionedLines(page) {
		if l.Y > tableTop {
			lines = append(lines, l)
		}
	}
	kinds := map[float64]string{}
	var kindOrder []float64
	for _, l := range lines {
		if abs(l.X-itemKindX) < slack && isItemKind(l.Text) {
			if _, seen := kinds[l.Y]; !seen {
				kindOrder = append(kindOrder, l.Y)
			}
			kinds[l.Y] = l.Text
		}
	}
	if len(kinds) == 0 {
		return priceMatrix{}, fmt.Errorf("não achei nenhuma linha da Tabela 3-9")
	}
	weaponY, found := 0.0, false
	for _, y := range kindOrder {
		if kinds[y] == "Arma" {
			weaponY, found = y, true
			break
		}
	}
	if !found {
		return priceMatrix{}, fmt.Errorf("a Tabela 3-9 na p%d não tem a linha `Arma`, que é a que dá as colunas", page-pdfOffset)
	}
	var columns []float64
	for _, l := range lines {
		if abs(l.Y-weaponY) < bookpdf.SameLine && priceCell.MatchString(l.Text) {
			columns = append(columns, l.X)
		}
	}
	sort.Float64s(columns)
	if len(columns) == 0 {
		return priceMatrix{}, fmt.Errorf("a linha `Arma` da Tabela 3-9 (y=%.1f) não tem célula `+ T$ N`", weaponY)
	}

	names := columnNames(book, page, columns)
	prices := map[cell]*int{}
	for _, l := range lines {
		kind, ok := kinds[l.Y]
		if !ok || l.X < columns[0]-30 {
			continue
		}
		m := priceCell.FindStringSubmatch(l.Text)
		if m == nil && l.Text != noPrice {
			continue
		}
		var value *int
		if m != nil {
			v, _ := strconv.Atoi(strings.ReplaceAll(m[1], ".", ""))
			value = &v
		}
		prices[cell{names[nearest(columns, l.X)], kind}] = value
	}
	return priceMatrix{names, prices, len(kinds)}, nil
}

// columnNames são os seis nomes do cabeçalho, lidos PALAVRA a palavra: por
// linha não dá, porque "Aço-Rubi Adamante" vem num elemento só e "Gelo Eterno"
// vem quebrado em duas alturas.
func columnNames(book *bookpdf.Book, page int, columns []float64) []string {
	parts := make([][]bookpdf.Positioned, len(columns))
	for _, w := range book.Words(page) {
		if tableTop < w.Y && w.Y < headerFoot && w.X > itemKindX+40 {
			i := nearest(columns, w.X)
			parts[i] = append(parts[i], w)
		}
	}
	names := make([]string, len(columns))
	for i, ws := range parts {
		sort.SliceStable(ws, func(a, b int) bool {
			if ws[a].Y != ws[b].Y {
				return ws[a].Y < ws[b].Y
			}
			return ws[a].Text < ws[b].Text
		})
		texts := make([]string, len(ws))
		for j, w := range ws {
			texts[j] = w.Text
		}
		names[i] = strings.Join(texts, " ")
	}
	return names
}

// nearest é o índice da coluna mais próxima; no empate, a primeira.
func nearest(columns []float64, x float64) int {
	best := 0
	for i := range columns {
		if abs(columns[i]-x) < abs(columns[best]-x) {
			best = i
		}
	}
	return best
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// halfLabel são as METADES que o verbete rotula. Capturar "do título até o
// próximo título" engoliria o boxe "Fabricando Itens Superiores", que divide a
// coluna com o Mitral — e a prosa de sabor não é regra, então medi-la seria
// responder outra pergunta.
var halfLabel = regexp.MustCompile(`^(Arma|Armadura|Escudo|Esotérico|Armadura e Escudo|Escudo e Esotérico)\. (.+)$`)

// materialTitle é uma linha curta, sem ponto final, no x do corpo. O x separa o
// título de SEÇÃO do de BOXE (o boxe mora no x do recuo, 326 = 309 + 17); e "ter
// metade rotulada abaixo" separa material de "Materiais Especiais". É regra
// ESTRUTURAL, não a lista de nomes da outra âncora — copiá-la faria as duas
// deixarem de se conferir.
var materialTitle = regexp.MustCompile(`^[A-ZÁÉÍÓÚÂÊÔÃÕÇ][\p{L}\p{N}_çãõáéíóúâêô-]*(?: [\p{L}\p{N}_çãõáéíóúâêô-]+){0,2}$`)

// entryParts são as partes de um verbete na ordem em que o livro as escreve.
type entryParts struct {
	order []string
	text  map[string]string
}

func (p *entryParts) set(part, text string) {
	if _, seen := p.text[part]; !seen {
		p.order = append(p.order, part)
	}
	p.text[part] = text
}

// bookEntries são os verbetes lidos, na ordem achada, e a página de cada um.
type bookEntries struct {
	order []string
	parts map[string]*entryParts
	page  map[string]int
}

// readEntries é {material: {parte: regra}}, lido BLOCO a bloco. A `abertura`
// entra porque duas regras moram nela e em nenhuma metade ("ocupam –1 espaço"
// do mitral, o "–2 em Carisma" da matéria vermelha); e é ela que obriga a ler
// por bloco, senão as 22 linhas do boxe vizinho entrariam na do Mitral.
func readEntries(book *bookpdf.Book, pages []int) bookEntries {
	found := bookEntries{parts: map[string]*entryParts{}, page: map[string]int{}}
	title, part, hasPart := "", "", false
	var body []string
	closeEntry := func() {
		if title == "" || !hasPart || part == "" || len(body) == 0 {
			return
		}
		p, ok := found.parts[title]
		if !ok {
			p = &entryParts{text: map[string]string{}}
			found.parts[title] = p
			found.order = append(found.order, title)
		}
		p.set(part, bookpdf.Join(body))
	}
	columns := map[int][]float64{}
	for _, page := range pages {
		for _, bl := range blocksInOrder(book, page) {
			// O bloco fora das colunas de corpo é BOXE — salvo a metade rotulada
			// de uma linha só, que nasce no x do recuo (Adamante, Gelo eterno).
			if !halfLabel.MatchString(bl.Lines[0]) && !isBodyColumn(book, columns, page, bl.X0) {
				continue
			}
			for _, text := range bl.Lines {
				if m := halfLabel.FindStringSubmatch(text); m != nil {
					closeEntry()
					part, hasPart, body = m[1], true, []string{m[2]}
				} else if materialTitle.MatchString(text) && !strings.HasSuffix(text, ".") {
					closeEntry()
					title, part, hasPart, body = text, opening, true, nil
					found.page[title] = page - pdfOffset
				} else if hasPart {
					body = append(body, text)
				}
			}
		}
	}
	closeEntry()
	var kept []string
	for _, name := range found.order {
		for _, p := range found.parts[name].order {
			if p != opening {
				kept = append(kept, name)
				break
			}
		}
	}
	found.order = kept
	return found
}

func isBodyColumn(book *bookpdf.Book, cache map[int][]float64, page int, x float64) bool {
	if _, ok := cache[page]; !ok {
		cache[page] = bookpdf.Columns(book.Blocks(page))
	}
	for _, c := range cache[page] {
		if abs(x-c) < slack {
			return true
		}
	}
	return false
}

// blocksInOrder são os blocos na ordem de leitura (coluna, depois y), sem a
// tabela e sem o lixo.
func blocksInOrder(book *bookpdf.Book, page int) []bookpdf.Block {
	blocks := book.Blocks(page)
	starts := bookpdf.Columns(blocks)
	sort.SliceStable(blocks, func(i, j int) bool {
		ci, cj := bookpdf.ColumnOf(starts, blocks[i].X0), bookpdf.ColumnOf(starts, blocks[j].X0)
		if ci != cj {
			return ci < cj
		}
		return blocks[i].Y < blocks[j].Y
	})
	var out []bookpdf.Block
	for _, bl := range blocks {
		if page == tablePage && bl.Y > tableTop {
			continue
		}
		var clean []string
		for _, t := range bookpdf.WithoutJunk(bl.Lines) {
			if bookpdf.Strip(t) != "" {
				clean = append(clean, t)
			}
		}
		if len(clean) > 0 {
			out = append(out, bookpdf.Block{X0: bl.X0, X1: bl.X1, Y: bl.Y, Lines: clean})
		}
	}
	return out
}
