package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"t20engine/infra/bookpdf"
)

// citation é uma linha do Índice Remissivo: o qualificador entre parênteses e
// as páginas impressas que ela cita.
type citation struct {
	qualifier string
	pages     []int
}

// book é o livro como duas visões: o texto de cada página impressa e o índice.
//
// Não é o `bookpdf.Book`: aquele lê por COORDENADA, uma página por vez, e aqui
// a pergunta é "o nome aparece no texto desta página?" sobre o livro inteiro —
// uma passada só do `pdftotext`, sem leiaute.
type book struct {
	opening, support int
	// flat é o texto de conferência ACHATADO: uma linha só, sem hífen de
	// quebra. O livro é diagramado em duas colunas estreitas, e um nome de duas
	// palavras quase sempre atravessa a quebra ("Abençoar Ali-\nmentos").
	// Comparar contra o texto cru reprovava 35 magias cuja página estava certa.
	flat  []string
	lines [][]string
	// raw são as páginas CRUAS, com acento e caixa: os catálogos de tipos de
	// efeito e de escolas saem daqui, e um texto dobrado viraria "cansaco" na
	// tela.
	raw   []string
	index map[string][]citation
	pages int
	// indexStart é a primeira impressa do PRÓPRIO índice, proibida como
	// resposta: uma condição chegou a ser resolvida para a 396, que é uma lista
	// de nomes — o botão "abrir no livro" abriria o índice com a conferência de
	// pé.
	indexStart int
}

// fatal é a parada do script Python (SystemExit): a mensagem vai para o
// stderr e o processo sai com 1.
type fatal struct{ msg string }

func (f fatal) Error() string { return f.msg }

func openBook(path string) (*book, error) {
	out, err := exec.Command("pdftotext", path, "-").Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext %s: %w", path, err)
	}
	pages := strings.Split(string(out), "\f")
	opening, support, err := measureOpening(pages)
	if err != nil {
		return nil, err
	}
	first, err := firstIndexPage(pages)
	if err != nil {
		return nil, err
	}
	b := &book{opening: opening, support: support, raw: pages, pages: len(pages)}
	for _, p := range pages {
		b.flat = append(b.flat, flatten(p))
		var folded []string
		for _, l := range strings.Split(p, "\n") {
			folded = append(folded, fold(l))
		}
		b.lines = append(b.lines, folded)
	}
	b.index = readIndex(pages, first)
	b.indexStart = first - opening + 1
	return b, nil
}

var footerNumber = regexp.MustCompile(`^[0-9]{1,3}$`)

// measureOpening é quantas páginas o ARQUIVO tem antes da impressa 1.
//
// MEDIDA e não suposta: o rodapé traz o número impresso, e a diferença entre ele
// e o índice do arquivo é a abertura. Tira-se a MODA porque nem toda página
// imprime o rodapé (as de abertura de capítulo não), e uma página solta com um
// número no fim do texto não pode decidir sozinha.
func measureOpening(pages []string) (int, int, error) {
	votes := map[int]int{}
	var order []int
	for i, page := range pages {
		lines := strings.Split(page, "\n")
		tail := lines[max(0, len(lines)-6):]
		for j := len(tail) - 1; j >= 0; j-- {
			line := bookpdf.Strip(tail[j])
			if !footerNumber.MatchString(line) {
				continue
			}
			n, _ := strconv.Atoi(line)
			if _, seen := votes[i+1-n]; !seen {
				order = append(order, i+1-n)
			}
			votes[i+1-n]++
			break
		}
	}
	if len(order) == 0 {
		return 0, 0, fatal{"não achei rodapé nenhum: o PDF não é o livro esperado"}
	}
	opening := order[0]
	for _, o := range order[1:] {
		if votes[o] > votes[opening] {
			opening = o
		}
	}
	if votes[opening] < 100 {
		return 0, 0, fatal{fmt.Sprintf("abertura %d apoiada por só %d páginas — não confio", opening, votes[opening])}
	}
	return opening, votes[opening], nil
}

// firstIndexPage é o índice do PDF (base 0) onde o Índice Remissivo começa: a
// PRIMEIRA da METADE DE TRÁS. O SUMÁRIO escreve "Índice Remissivo" na abertura
// do livro, e o próprio índice repete a frase como cabeçalho em cada página.
// Pegar a primeira ocorrência lia o sumário (130 nomes); pegar a última começava
// no meio do índice (306). São 1.217.
func firstIndexPage(pages []string) (int, error) {
	wanted := fold("Índice Remissivo")
	for i, page := range pages {
		if i >= len(pages)/2 && strings.Contains(fold(page), wanted) {
			return i, nil
		}
	}
	return 0, fatal{"não achei o Índice Remissivo no PDF"}
}

var (
	trailingDots  = regexp.MustCompile(`\.{2,}$`)
	indexEntry    = regexp.MustCompile(`^(.+?)[.` + spaceChars + `]*\.{3,}` + pySpace + `*([0-9,` + spaceChars + `]+)$`)
	digitRun      = regexp.MustCompile(`[0-9]+`)
	qualifierTail = regexp.MustCompile(`\(([^)]*)\)` + pySpace + `*$`)
	qualifierCut  = regexp.MustCompile(pySpace + `*\([^)]*\)` + pySpace + `*$`)
)

// readIndex lê o Índice Remissivo: nome dobrado → as citações dele.
//
// O qualificador entre parênteses é do próprio livro e resolve as colisões que
// mais importam aqui: "grifo (criatura)" contra "grifo (parceiro)". A leitura
// vai até o FIM do livro, e não um número fixo de páginas: página que não é
// índice não tem linha pontilhada e não contribui com nada.
func readIndex(pages []string, first int) map[string][]citation {
	var lines []string
	for _, page := range pages[first:] {
		for _, l := range strings.Split(page, "\n") {
			lines = append(lines, bookpdf.Strip(l))
		}
	}
	// A linha pode terminar no pontilhado com o número na seguinte: o texto sai
	// coluna a coluna e a quebra cai no meio da entrada.
	var joined []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line != "" && trailingDots.MatchString(line) && i+1 < len(lines) {
			line += " " + lines[i+1]
			i++
		}
		joined = append(joined, line)
	}

	found := map[string][]citation{}
	for _, line := range joined {
		m := indexEntry.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		name := bookpdf.Strip(strings.TrimRight(bookpdf.Strip(m[1]), "."))
		var cited []int
		for _, n := range digitRun.FindAllString(m[2], -1) {
			v, _ := strconv.Atoi(n)
			cited = append(cited, v)
		}
		if name == "" || len(cited) == 0 {
			continue
		}
		qualifier := ""
		if q := qualifierTail.FindStringSubmatch(name); q != nil {
			qualifier = q[1]
		}
		key := fold(qualifierCut.ReplaceAllString(name, ""))
		found[key] = append(found[key], citation{qualifier, cited})
	}
	return found
}

func (b *book) position(printed int) int { return printed + b.opening - 1 }

// page é o texto achatado de uma página impressa, ou vazio fora do livro.
func (b *book) page(printed int) string {
	i := b.position(printed)
	if i < 0 || i >= len(b.flat) {
		return ""
	}
	return b.flat[i]
}

// rawPage é a página impressa crua, com acento e caixa.
func (b *book) rawPage(printed int) string { return b.raw[b.position(printed)] }

// confirms pergunta se o nome aparece no texto daquela página impressa. É o
// CONTROLE de tudo o que sai daqui: sem ele, uma página do índice mal lida
// entraria no catálogo com cara de dado conferido.
func (b *book) confirms(name string, printed int) bool {
	if printed >= b.indexStart {
		return false
	}
	if i := b.position(printed); i < 0 || i >= len(b.flat) {
		return false
	}
	text, target := b.page(printed), flatten(name)
	return strings.Contains(text, target) || withSealInMiddle(target, text)
}

// byIndex são as páginas que o índice cita para o nome, na ordem dele: as da
// primeira citação cujo qualificador é aceito (nenhum qualificador aceita todas).
func (b *book) byIndex(name string, qualifiers ...string) []int {
	for _, c := range b.index[fold(name)] {
		if len(qualifiers) > 0 && !contains(qualifiers, c.qualifier) {
			continue
		}
		return c.pages
	}
	return nil
}

// byTitle são as páginas de uma faixa em que alguma LINHA começa com o nome.
//
// Serve para o que o índice não lista. A faixa é o que a torna confiável: um
// nome de item começa linha em dezenas de páginas de prosa, e dentro do
// capítulo de equipamento ele é a linha da tabela.
func (b *book) byTitle(name string, span pageSpan) []int {
	target := fold(name)
	var out []int
	for printed := span.first; printed <= span.last; printed++ {
		i := b.position(printed)
		if i < 0 || i >= len(b.lines) {
			continue
		}
		for _, line := range b.lines[i] {
			if line == target || strings.HasPrefix(line, target+" ") || strings.HasPrefix(line, target+".") {
				out = append(out, printed)
				break
			}
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// sortedKeys são as chaves de um conjunto, em ordem — o `sorted(set)` do Python.
func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
