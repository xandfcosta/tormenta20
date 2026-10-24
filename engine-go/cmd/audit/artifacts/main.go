// Command artifacts confere os ARTEFATOS do catálogo contra as p346-349
// (ALE-455).
//
// O denominador é INVERTIDO, e aqui ele não tinha escolha: a p346 diz que "não
// há tabelas para geração aleatória de artefatos", então esta é a única família
// do Capítulo 8 sem o `d%` que fecha em 100. Ele vem do TIPO: o cabeçalho de
// seção do capítulo é desenhado a 22,281 pontos, e dentro das p346-349 não há
// outra seção — os cabeçalhos dali SÃO a lista de artefatos, medida na página.
// Lendo a olho, contei três; são cinco.
//
// O CONTROLE vem antes do resultado: a mesma varredura roda na p335 e na p341,
// onde se sabe o que ela tem de achar. Não se confere a REGRA — isso é dos
// guardas do catálogo; a nota de `unmodeled` é IMPRESSA, para o olho conferir se
// envelheceu.
//
//	go run ./cmd/audit/artifacts
package main

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"t20engine/cmd/internal/pyfmt"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

const pdfOffset = 6

// headerHeight é a altura do cabeçalho de seção do Capítulo 8, medida. A folga
// é apertada de propósito: a legenda da tabela do Baralho, na mesma página, é
// 20,604 — uma janela larga leria a legenda e o cabeçalho como dois artefatos.
const (
	headerHeight = 22.281
	heightSlack  = 0.1
	slack        = 3.0
)

var sectionPages = []int{346, 347, 348, 349}

// controls são páginas de FORA da seção e o cabeçalho que a varredura tem de
// achar nelas: provam que a altura reconhece cabeçalho em geral.
var controls = []struct {
	page     int
	expected string
}{{335, "Encantos"}, {341, "Poções"}}

// A tabela do Baralho, na p347. O `d%` é faixa porque "99-100" começa dois
// pontos à esquerda das outras; não há coluna de preço.
const (
	deckPDFPage = 353
	deckNameX   = 343.0
)

var deckDiceX = [2]float64{310.0, 316.0}

var wordWithHeight = regexp.MustCompile(`<word xMin="([\d.]+)" yMin="([\d.]+)" xMax="[\d.]+" yMax="([\d.]+)">(.*?)</word>`)

// failExit é o `SystemExit(mensagem)` do Python: a mensagem no stderr e o
// código 1.
func failExit(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func main() {
	book, err := bookpdf.Open()
	if err != nil {
		failExit(err.Error())
	}
	os.Exit(min(1, audit(book)))
}

// pageHeaders são os cabeçalhos de seção de UMA página, pela ALTURA do tipo.
// Agrupa por `y` porque um cabeçalho de quatro palavras são quatro `<word>`.
//
// Lê o `-bbox` sem o `-layout`: é o único que traz o `yMax` de cada palavra, e
// o `bookpdf` não o expõe. O texto da palavra fica CRU, sem desfazer entidade,
// como no auditor original: os cabeçalhos medidos não têm entidade nenhuma.
func pageHeaders(book *bookpdf.Book, pdfPage int) []string {
	p := strconv.Itoa(pdfPage)
	out, err := exec.Command("pdftotext", "-bbox", "-f", p, "-l", p, book.Path, "-").Output()
	if err != nil {
		failExit(fmt.Sprintf("pdftotext -bbox na página %d do PDF falhou: %v", pdfPage, err))
	}
	type word struct {
		x    float64
		text string
	}
	byY := map[float64][]word{}
	for _, m := range wordWithHeight.FindAllStringSubmatch(string(out), -1) {
		y0, _ := strconv.ParseFloat(m[2], 64)
		y1, _ := strconv.ParseFloat(m[3], 64)
		if math.Abs((y1-y0)-headerHeight) > heightSlack {
			continue
		}
		x, _ := strconv.ParseFloat(m[1], 64)
		y := math.RoundToEven(y0)
		byY[y] = append(byY[y], word{x, m[4]})
	}
	ys := make([]float64, 0, len(byY))
	for y := range byY {
		ys = append(ys, y)
	}
	sort.Float64s(ys)
	headers := make([]string, 0, len(ys))
	for _, y := range ys {
		ws := byY[y]
		sort.SliceStable(ws, func(i, j int) bool {
			if ws[i].x != ws[j].x {
				return ws[i].x < ws[j].x
			}
			return ws[i].text < ws[j].text
		})
		texts := make([]string, len(ws))
		for i, w := range ws {
			texts[i] = w.text
		}
		headers = append(headers, strings.Join(texts, " "))
	}
	return headers
}

// proveTheMeterSees para o auditor se a altura não achar um cabeçalho
// conhecido: sem isto, "nenhum artefato faltando" e "não medi nada" são a mesma
// frase.
func proveTheMeterSees(book *bookpdf.Book) {
	for _, c := range controls {
		found := pageHeaders(book, c.page+pdfOffset)
		seen := false
		for _, h := range found {
			if strings.Contains(bookpdf.Key(h), bookpdf.Key(c.expected)) {
				seen = true
			}
		}
		if !seen {
			failExit(fmt.Sprintf("O MEDIDOR ESTÁ CEGO: a p%d tem o cabeçalho %s e a varredura por altura %s achou %s.\n"+
				"Sem isto, \"nenhum artefato faltando\" e \"não medi nada\" são a mesma frase — o auditor para aqui de propósito.",
				c.page, pyfmt.Repr(c.expected), strconv.FormatFloat(headerHeight, 'f', -1, 64), pyfmt.List(found)))
		}
	}
}

var deckRange = regexp.MustCompile(`^(\d{1,3})-(\d{1,3})$`)

// deckTable são as 22 linhas do Baralho, lidas PALAVRA por palavra: o
// `LootRows` lê o nome pelo x da LINHA, e "Tentáculo" encosta na coluna do
// efeito — o `pdftotext` emite as duas células numa `<line>` só e a carta sai
// "Tentáculo –1 em um atributo aleatório". As outras 21 passam, que é o pior
// jeito de errar. A linha é `LootRow` porque o `RangesThatDontClose` lê essa.
func deckTable(book *bookpdf.Book) []bookpdf.LootRow {
	type span struct{ start, end int }
	ranges, names := map[float64]span{}, map[float64]string{}
	for _, w := range book.Words(deckPDFPage) {
		y := math.RoundToEven(w.Y)
		if deckDiceX[0] <= w.X && w.X <= deckDiceX[1] {
			if m := deckRange.FindStringSubmatch(w.Text); m != nil {
				start, _ := strconv.Atoi(m[1])
				end, _ := strconv.Atoi(m[2])
				ranges[y] = span{start, end}
			}
		} else if math.Abs(w.X-deckNameX) < slack {
			names[y] = w.Text
		}
	}
	ys := make([]float64, 0, len(ranges))
	for y := range ranges {
		ys = append(ys, y)
	}
	sort.Float64s(ys)
	rows := make([]bookpdf.LootRow, 0, len(ys))
	for _, y := range ys {
		rows = append(rows, bookpdf.LootRow{Start: ranges[y].start, End: ranges[y].end, Name: names[y]})
	}
	return rows
}

// articles não são parte do nome: o livro escreve "A Espada-Deus", "os Rubis
// da Virtude".
var articles = []string{"a ", "o ", "as ", "os "}

// itemName declara o ÚNICO cabeçalho que não é o nome do item, para o
// casamento poder ser EXATO: a seção dos Rubis é plural (são vinte gemas) e o
// catálogo guarda UMA, porque a p349 soma o efeito de cada uma. Casar por
// prefixo resolveria o caso e abriria outro — "Rubi" casa com qualquer verbete
// futuro que comece com as mesmas letras.
var itemName = map[string]string{"Rubis da Virtude": "Rubi da Virtude"}

func nameOfItem(title string) string {
	lower := strings.ToLower(title)
	for _, a := range articles {
		if strings.HasPrefix(lower, a) {
			title = title[len(a):]
			break
		}
	}
	if n, ok := itemName[title]; ok {
		return n
	}
	return title
}

func audit(book *bookpdf.Book) int {
	proveTheMeterSees(book)

	fromBook := map[string]int{}
	for _, page := range sectionPages {
		for _, title := range pageHeaders(book, page+pdfOffset) {
			fromBook[title] = page
		}
	}
	inApp := map[string]transcript.Item{}
	var appOrder []string
	for _, it := range transcript.Items {
		if !it.Artifact {
			continue
		}
		k := bookpdf.Key(it.Name)
		if _, seen := inApp[k]; !seen {
			appOrder = append(appOrder, k)
		}
		inApp[k] = it
	}
	var failures []string

	first, last := sectionPages[0], sectionPages[len(sectionPages)-1]
	fmt.Printf("══ ARTEFATOS (p%d-%d) %s\n", first, last, strings.Repeat("═", 28))
	fmt.Printf("%4s %-30s catálogo\n", "pág", "cabeçalho do livro")
	titles := make([]string, 0, len(fromBook))
	for t := range fromBook {
		titles = append(titles, t)
	}
	sort.Slice(titles, func(i, j int) bool {
		if fromBook[titles[i]] != fromBook[titles[j]] {
			return fromBook[titles[i]] < fromBook[titles[j]]
		}
		return titles[i] < titles[j]
	})
	matched := 0
	for _, title := range titles {
		page := fromBook[title]
		entry, ok := inApp[bookpdf.Key(nameOfItem(title))]
		shown := "— AUSENTE"
		if ok {
			shown = entry.ID
		}
		fmt.Printf("%4d %-30s %s\n", page, title, shown)
		if !ok {
			failures = append(failures, fmt.Sprintf("%s: a p%d traz este artefato e o catálogo não o tem", title, page))
			continue
		}
		matched++
		if entry.BookPage != page {
			failures = append(failures, fmt.Sprintf("%s: o catálogo cita p%s e o cabeçalho está na p%d", title, pyfmt.Optional(float64(entry.BookPage)), page))
		}
	}

	// E O LADO DE LÁ: artefato no app que o livro não imprime. Sem esta metade,
	// um verbete inventado passaria.
	bookKeys := map[string]bool{}
	for _, t := range titles {
		bookKeys[bookpdf.Key(nameOfItem(t))] = true
	}
	sortedApp := append([]string(nil), appOrder...)
	sort.Strings(sortedApp)
	for _, k := range sortedApp {
		if !bookKeys[k] {
			failures = append(failures, fmt.Sprintf("%s: está marcado como artefato no catálogo e nenhum cabeçalho das p%d-%d corresponde a ele",
				inApp[k].Name, first, last))
		}
	}

	// O CONTADOR é dos que CASARAM, e não dos dois tamanhos: "5/5" já saiu com
	// um AUSENTE na tabela acima, na primeira corrida deste auditor.
	fmt.Printf("\n%d/%d transcritos\n", matched, len(fromBook))
	failures = append(failures, auditDeck(book)...)
	printNotes(inApp, appOrder)

	for _, f := range failures {
		fmt.Printf("  ✗ %s\n", f)
	}
	return len(failures)
}

// auditDeck confere as 22 cartas pela faixa de `d%` e pelo nome.
func auditDeck(book *bookpdf.Book) []string {
	rows := deckTable(book)
	if gap := bookpdf.RangesThatDontClose(rows); gap != "" {
		failExit("A TABELA DO BARALHO NÃO FECHA NO LIVRO: " + gap + "\nNada do que viria abaixo valeria — a leitura está torta.")
	}
	cards := transcript.GMTables.ChaosDeck
	var failures []string
	fmt.Printf("\n══ O BARALHO DO CAOS (p347) %s\n", strings.Repeat("═", 32))
	fmt.Printf("%7s %-16s app\n", "d%", "carta no livro")
	for _, r := range rows {
		span := strconv.Itoa(r.Start)
		if r.End != r.Start {
			span = fmt.Sprintf("%d-%d", r.Start, r.End)
		}
		var card *transcript.ChaosCard
		for i := range cards {
			if cards[i].RollMin == r.Start && cards[i].RollMax == r.End {
				card = &cards[i]
				break
			}
		}
		shown := "— AUSENTE"
		if card != nil {
			shown = card.Card
		}
		fmt.Printf("%7s %-16s %s\n", span, r.Name, shown)
		if card == nil {
			failures = append(failures, fmt.Sprintf("carta %s (%s): o app não tem nenhuma carta nessa faixa", r.Name, span))
		} else if bookpdf.Key(card.Card) != bookpdf.Key(r.Name) {
			failures = append(failures, fmt.Sprintf("a faixa %s é %s no livro e %s no app", span, r.Name, pyfmt.Repr(card.Card)))
		}
	}
	if len(cards) != len(rows) {
		failures = append(failures, fmt.Sprintf("o app tem %d cartas e a p347 imprime %d", len(cards), len(rows)))
	}
	fmt.Printf("\n%d/%d cartas, d%% 1-100 tilado\n", len(cards), len(rows))
	return failures
}

// printNotes imprime a nota de `unmodeled` de cada artefato: "a nota
// envelheceu" não se mede por varredura de texto, e o relatório é onde o olho
// passa.
func printNotes(inApp map[string]transcript.Item, order []string) {
	fmt.Printf("\n══ O QUE NÃO FOI MODELADO %s\n", strings.Repeat("═", 34))
	byName := append([]string(nil), order...)
	sort.SliceStable(byName, func(i, j int) bool { return inApp[byName[i]].Name < inApp[byName[j]].Name })
	for _, k := range byName {
		entry := inApp[k]
		note := "(nada: o motor aplica o verbete inteiro)"
		if entry.Unmodeled != nil && entry.Unmodeled.Note != "" {
			note = entry.Unmodeled.Note
		}
		fmt.Printf("\n  %s (p%s)\n", entry.Name, pyfmt.Optional(float64(entry.BookPage)))
		for _, line := range strings.Split(bookpdf.Join([]string{note}), ". ") {
			fmt.Printf("    %s\n", bookpdf.Strip(line))
		}
	}
}
