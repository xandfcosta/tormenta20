package main

import (
	"regexp"
	"strings"

	"t20engine/domain/catalog/transcript"
	"t20engine/infra/bookpdf"
)

// Os `\s` do Python são Unicode; o RE2 os leria como ASCII.

// bookLine é uma linha do capítulo e a página do LIVRO de onde ela saiu.
type bookLine struct {
	page int
	text string
}

// bookLines é o capítulo em ordem de leitura, sem marca d'água e sem cabeçalho.
// O script de origem marcava o "--- fim de coluna ---" entre colunas, e a
// própria limpeza o descartava; a ordem é a mesma sem ele.
func bookLines(book *bookpdf.Book) []bookLine {
	var out []bookLine
	for page := firstPage; page <= lastPage; page++ {
		for _, line := range book.PageLines(page) {
			if t := bookpdf.Collapse(line); !bookpdf.IsJunk(t) {
				out = append(out, bookLine{page - pdfOffset, t})
			}
		}
	}
	return out
}

// treasureLines é o teto de linhas da frase do tesouro. O Reishid passa dele —
// o tesouro é seguido da DESCRIÇÃO do item mágico —, e aí a ferramenta avisa em
// vez de cortar calada.
const treasureLines = 4

var (
	ndLine = regexp.MustCompile(`(?i)^` + bookpdf.PySpace + `*ND` + bookpdf.PySpace + `+\d+(?:/\d+)?` + bookpdf.PySpace + `*$`)
	ndTail = regexp.MustCompile(`(?i)` + bookpdf.PySpace + `*ND` + bookpdf.PySpace + `+\d+(?:/\d+)?` + bookpdf.PySpace + `*$`)
)

// creatureBlock é o corpo de um verbete no livro, do nome ao Tesouro.
type creatureBlock struct {
	page      int
	body      []string
	truncated bool
}

type blockStart struct{ line, nameLines int }

// blocksByName corta o capítulo do nome de cada verbete até a linha de Tesouro
// dele. Exigir linha inteira igual ao nome perdia catorze verbetes; o nome é
// tentado do MAIS LONGO para o mais curto, senão "Troll" come "Troll das
// Cavernas" e deixa "das Cavernas" órfão.
func blocksByName(lines []bookLine, names map[string]*transcript.Creature) map[string]creatureBlock {
	var starts []blockStart
	var keys []string
	for i := 0; i < len(lines); {
		matched := 0
		for _, n := range []int{3, 2, 1} {
			if i+n > len(lines) {
				continue
			}
			// O ND pode cair NO MEIO do nome quebrado em duas linhas.
			var pieces []string
			for _, l := range lines[i : i+n] {
				if !ndLine.MatchString(l.text) {
					pieces = append(pieces, l.text)
				}
			}
			k := bookpdf.Key(ndTail.ReplaceAllString(strings.Join(pieces, " "), ""))
			if _, ok := names[k]; ok {
				matched = n
				starts = append(starts, blockStart{i, n})
				keys = append(keys, k)
				break
			}
		}
		if matched > 0 {
			i += matched
		} else {
			i++
		}
	}

	out := map[string]creatureBlock{}
	for s, start := range starts {
		end := len(lines)
		if s+1 < len(starts) {
			end = starts[s+1].line
		}
		var body []string
		for _, l := range lines[start.line+start.nameLines : end] {
			body = append(body, l.text)
		}
		body, truncated := cutAtTreasure(body)
		if _, seen := out[keys[s]]; !seen {
			out[keys[s]] = creatureBlock{lines[start.line].page, body, truncated}
		}
	}
	return out
}

// cutAtTreasure corta o bloco na frase do Tesouro que vem DEPOIS da linha de
// atributos. Sem o corte ele corre até o próximo verbete do CATÁLOGO e engole
// os números de uma criatura que o catálogo não tem; cortar antes dos atributos
// deixava o Thuwarokk sem eles, porque uma habilidade cita tesouro.
//
// A frase atravessa linhas ("Tesouro Um ninho de grifo tem 25% de chance de
// conter" / "1d4 ovos…"), e por isso o corte vai até a linha que FECHA a frase.
func cutAtTreasure(body []string) ([]string, bool) {
	seenAttributes := false
	for j, t := range body {
		if attributesRe.MatchString(t) {
			seenAttributes = true
		}
		if !strings.Contains(t, "Tesouro") || !seenAttributes {
			continue
		}
		end := j + 1
		for end <= j+treasureLines && end <= len(body) && !endsSentence(body[end-1]) {
			end++
		}
		truncated := !endsSentence(body[min(end, len(body))-1])
		return body[:min(end, len(body))], truncated
	}
	return body, false
}

func endsSentence(line string) bool {
	return strings.HasSuffix(strings.TrimRightFunc(line, bookpdf.IsPySpace), ".")
}

var required = []*regexp.Regexp{
	regexp.MustCompile(`Iniciativa` + bookpdf.PySpace + `*[+–−-]?\d+`),
	regexp.MustCompile(`Defesa` + bookpdf.PySpace + `*\d+.*Fort.*Ref.*Von`),
	regexp.MustCompile(`Pontos de Vida` + bookpdf.PySpace + `*\d+`),
	regexp.MustCompile(`Tesouro`),
}

// completeBlock diz se o bloco tem cabeça de bloco de criatura.
func completeBlock(body []string) bool {
	text := bookpdf.Join(body)
	for _, r := range required {
		if !r.MatchString(text) {
			return false
		}
	}
	return true
}

var attributeLineRe = regexp.MustCompile(`^For` + bookpdf.PySpace + `*[+–−-]?[\d—]`)

// hasAttributeLine é o `\bFor…` do Python, com a fronteira de palavra UNICODE:
// a do RE2 é ASCII e veria fronteira depois de um "é". Sem a linha de
// atributos dá para PREENCHER os campos, mas não para AUDITAR os seis.
func hasAttributeLine(text string) bool {
	for i := 0; ; {
		at := strings.Index(text[i:], "For")
		if at < 0 {
			return false
		}
		at += i
		if !wordBefore(text, at) && attributeLineRe.MatchString(text[at:]) {
			return true
		}
		i = at + 1
	}
}
