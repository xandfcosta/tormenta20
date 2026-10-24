package main

import (
	"regexp"
	"strconv"
	"strings"

	"t20engine/infra/bookpdf"
)

// Os `\s` do Python são Unicode; o RE2 os leria como ASCII.

var (
	listHeader = regexp.MustCompile(`(?i)^(Arcana|Divina|Universal)` + bookpdf.PySpace + `+(\d)` + bookpdf.PySpace + `+\(([^)]+)\)` + bookpdf.PySpace + `*$`)
	augmentRe  = regexp.MustCompile(`^(?:\+(\d+)` + bookpdf.PySpace + `*PM(?:` + bookpdf.PySpace + `*\(Apenas` + bookpdf.PySpace + `+([^)]+)\))?|Truque)` +
		bookpdf.PySpace + `*:` + bookpdf.PySpace + `*(.*)$`)
)

// chapterLine é uma linha do capítulo e a página do PDF de onde ela saiu.
type chapterLine struct {
	page int
	text string
}

// bookSpell é uma magia lida no livro: o cabeçalho e o corpo até a próxima.
type bookSpell struct {
	id     string
	title  string
	page   int
	list   string
	circle int
	school string
	body   []string
}

// bookAugment é um aprimoramento lido no livro.
type bookAugment struct {
	pmCost    int
	cantrip   bool
	classOnly string // "" é o `None` do Python
	text      string
}

// chapterText é o capítulo inteiro em ordem de leitura.
//
// LARGA A LINHA INTEIRA que tem marca d'água, e não só o trecho que casou: a
// marca é "mateush.santos42@gmail.com", e tirar só o nome deixa "42@gmail.com"
// grudado no fim do aprimoramento.
func chapterText(book *bookpdf.Book) []chapterLine {
	var out []chapterLine
	for page := firstPage; page <= lastPage; page++ {
		for _, line := range book.PageLines(page) {
			if clean := bookpdf.Strip(line); !bookpdf.IsJunk(clean) {
				out = append(out, chapterLine{page, clean})
			}
		}
	}
	return out
}

// spellsInBook acha cada magia pelo par NOME + `Lista N (Escola)`. Pelo
// cabeçalho e não por nome solto: "Luz" aparece dezenas de vezes no capítulo, e
// só uma delas tem a linha de lista logo abaixo.
func spellsInBook(book *bookpdf.Book, names map[string]string) map[string]*bookSpell {
	lines := chapterText(book)
	found := map[string]*bookSpell{}
	var open *bookSpell
	for i, line := range lines {
		m := listHeader.FindStringSubmatch(line.text)
		if m != nil && i > 0 {
			title, key, used := titleAbove(lines, i, names)
			if open != nil {
				// O título da magia que começa agora já entrou no corpo da
				// anterior, antes de se saber que era título. São `used` linhas:
				// o livro quebra nome longo em duas.
				for range used {
					if len(open.body) > 0 {
						open.body = open.body[:len(open.body)-1]
					}
				}
			}
			id, ok := names[key]
			if !ok {
				// Cabeçalho de uma magia que o catálogo não tem: fecha a
				// anterior, senão o corpo dela engole a seguinte inteira.
				open = nil
				continue
			}
			circle, _ := strconv.Atoi(m[2])
			open = &bookSpell{id: id, title: title, page: line.page, list: m[1], circle: circle, school: m[3]}
			found[id] = open
			continue
		}
		if open != nil {
			open.body = append(open.body, line.text)
		}
	}
	return found
}

// titleAbove é o título da magia cujo cabeçalho está em `i`, e quantas linhas
// ele ocupa. O nome longo quebra ("Barragem elemental" / "de Vectorius"), e só
// olhar a linha de cima deixava QUARENTA magias sem medir. Do mais LONGO para o
// mais curto: senão "Luz" casaria antes de "Luz do Amanhecer".
func titleAbove(lines []chapterLine, i int, names map[string]string) (string, string, int) {
	for _, count := range []int{3, 2, 1} {
		if i-count < 0 {
			continue
		}
		parts := make([]string, 0, count)
		for j := i - count; j < i; j++ {
			parts = append(parts, lines[j].text)
		}
		joined := bookpdf.Strip(strings.Join(parts, " "))
		if _, ok := names[bookpdf.Key(joined)]; ok {
			return joined, bookpdf.Key(joined), count
		}
	}
	return lines[i-1].text, bookpdf.Key(lines[i-1].text), 1
}

// augmentsOfBody são os aprimoramentos de um corpo, cada um com o texto até o
// próximo. O texto se junta desfazendo a hifenização de quebra ("perma- nente"),
// senão ele não casa com nada e parece defeito de transcrição.
func augmentsOfBody(body []string) []bookAugment {
	var out []bookAugment
	var texts [][]string
	for _, line := range body {
		if m := augmentRe.FindStringSubmatchIndex(line); m != nil {
			a := bookAugment{cantrip: m[2] < 0, classOnly: strings.ToLower(groupAt(line, m, 2))}
			if !a.cantrip {
				a.pmCost, _ = strconv.Atoi(groupAt(line, m, 1))
			}
			out = append(out, a)
			texts = append(texts, []string{groupAt(line, m, 3)})
			continue
		}
		if len(out) > 0 && line != "" {
			texts[len(texts)-1] = append(texts[len(texts)-1], line)
		}
	}
	for i := range out {
		out[i].text = bookpdf.Strip(bookpdf.Join(texts[i]))
	}
	return out
}

// groupAt é o grupo `n` de um casamento por índice, "" quando ele não casou.
func groupAt(s string, m []int, n int) string {
	if m[2*n] < 0 {
		return ""
	}
	return s[m[2*n]:m[2*n+1]]
}
