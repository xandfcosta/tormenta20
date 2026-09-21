package markdown

import (
	"regexp"
	"strings"
)

// O MARKDOWN DAS NOTAS DA SESSÃO.
//
// É um markdown PEQUENO, do tamanho de uma nota de mesa, e ele produz uma
// ÁRVORE — nunca HTML. Quem desenha é o templ, montando elementos a partir
// daqui, então não existe `innerHTML` no caminho e injeção é impossível por
// construção: sem parser de terceiro e sem sanitizador atrás dele.
//
// POR QUE ESCRITO À MÃO E NÃO UM GOLDMARK: o comportamento PADRÃO de um parser
// CommonMark junta linhas soltas num parágrafo só, e numa nota de mesa a quebra
// é intencional — ver `closeParagraph`.
//
// A gramática tem linha de base congelada em
// `api/testdata/markdown-from-the-js.json`, contra a qual o
// `markdown/markdown_test.go` compara a árvore.

// Span é um trecho de uma linha. `Href` só existe em `elo`.
//
// As etiquetas JSON são as da linha de base de propósito: é o que deixa o teste
// comparar as duas árvores sem uma terceira tradução no meio.
type Span struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

// Task é o estado de um `- [ ]`, com a LINHA de origem junto.
//
// A linha não é enfeite: é ela que deixa o clique no quadrinho reescrever o
// texto do mestre. Sem ela o controle seria decorativo e o estado moraria fora
// da nota, que é onde ele não sobrevive a um F5.
type Task struct {
	Marked bool `json:"checked"`
	Row    int  `json:"line"`
}

type Item struct {
	Spans []Span `json:"spans"`
	Task  *Task  `json:"task,omitempty"`
}

// Block é um bloco da nota. `Kind` diz qual dos campos vale — uma struct só
// em vez de uma interface porque quem consome é um `switch` de template, e uma
// hierarquia de tipos aqui compraria indireção sem comprar nada.
type Block struct {
	Kind   string   `json:"kind"`
	Level  int      `json:"level,omitempty"`
	Sorted bool     `json:"ordered"`
	Spans  []Span   `json:"spans,omitempty"`
	Rows   [][]Span `json:"lines,omitempty"`
	Items  []Item   `json:"items,omitempty"`
}

var (
	mdTitulo   = regexp.MustCompile(`^(#{1,3})\s+(.*)$`)
	mdTarefaRe = regexp.MustCompile(`^[-*]\s+\[([ xX])\]\s*(.*)$`)
	mdItemRe   = regexp.MustCompile(`^[-*]\s+(.*)$`)
	mdOrdenada = regexp.MustCompile(`^\d+[.)]\s+(.*)$`)
	mdCitacao  = regexp.MustCompile(`^>\s?(.*)$`)
	mdRegua    = regexp.MustCompile(`^(-{3,}|\*{3,})$`)
	mdMarca    = regexp.MustCompile("(`[^`]+`)|(\\*\\*[^*]+\\*\\*)|(\\*[^*]+\\*|_[^_]+_)|(\\[[^\\]]+\\]\\([^)]+\\))")
	mdElo      = regexp.MustCompile(`^\[([^\]]+)\]\(([^)]+)\)$`)
	mdHTTP     = regexp.MustCompile(`(?i)^https?://`)
)

// Parse traduz o texto da nota na árvore que a tela desenha.
//
// @example Parse("# Cena 1\n- Ogro **fugiu**")
func Parse(source string) []Block {
	blocks := []Block{}
	var paragraph []string
	rows := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	for i := 0; i < len(rows); i++ {
		row := strings.TrimSpace(rows[i])
		if row == "" {
			blocks, paragraph = closeParagraph(blocks, paragraph)
			continue
		}
		if block, ok := oneLineBlock(row); ok {
			blocks, paragraph = closeParagraph(blocks, paragraph)
			blocks = append(blocks, block)
			continue
		}
		if mdItemRe.MatchString(row) || mdOrdenada.MatchString(row) {
			blocks, paragraph = closeParagraph(blocks, paragraph)
			var list Block
			list, i = gatherList(rows, i)
			blocks = append(blocks, list)
			continue
		}
		paragraph = append(paragraph, row)
	}
	blocks, _ = closeParagraph(blocks, paragraph)
	return blocks
}

// oneLineBlock resolve os três blocos que cabem numa linha só. Falso devolve
// a decisão a quem chamou — lista precisa de várias linhas, e o resto é texto.
func oneLineBlock(row string) (Block, bool) {
	if m := mdTitulo.FindStringSubmatch(row); m != nil {
		return Block{Kind: "heading", Level: len(m[1]), Spans: parseSpans(m[2])}, true
	}
	if mdRegua.MatchString(row) {
		return Block{Kind: "rule"}, true
	}
	if m := mdCitacao.FindStringSubmatch(row); m != nil {
		return Block{Kind: "quote", Spans: parseSpans(m[1])}, true
	}
	return Block{}, false
}

// closeParagraph despeja as linhas acumuladas num bloco.
//
// CADA LINHA DIGITADA É UMA LINHA NA TELA. Numa nota de mesa a quebra é
// intencional, e juntá-las como o markdown padrão manda transforma trinta
// linhas de anotação num parágrafo só. É a divergência que faz este parser
// existir em vez de uma dependência.
func closeParagraph(blocks []Block, paragraph []string) ([]Block, []string) {
	if len(paragraph) == 0 {
		return blocks, paragraph
	}
	rows := make([][]Span, 0, len(paragraph))
	for _, l := range paragraph {
		rows = append(rows, parseSpans(l))
	}
	return append(blocks, Block{Kind: "paragraph", Rows: rows}), nil
}

// gatherList junta as linhas seguidas de uma lista num bloco só e devolve o
// índice da ÚLTIMA consumida — itens soltos viravam um bloco por linha, e a
// marcação de lista se perdia.
func gatherList(rows []string, start int) (Block, int) {
	sorted := mdOrdenada.MatchString(strings.TrimSpace(rows[start]))
	items := []Item{}
	i := start
	for ; i < len(rows); i++ {
		row := strings.TrimSpace(rows[i])
		re := mdItemRe
		if sorted {
			re = mdOrdenada
		}
		m := re.FindStringSubmatch(row)
		if m == nil {
			break
		}
		items = append(items, listItem(row, i, m[1]))
	}
	return Block{Kind: "list", Sorted: sorted, Items: items}, i - 1
}

// listItem: `- [ ] dar XP` é um item com ESTADO; qualquer outro é comum.
func listItem(row string, index int, text string) Item {
	m := mdTarefaRe.FindStringSubmatch(row)
	if m == nil {
		return Item{Spans: parseSpans(text)}
	}
	return Item{
		Spans: parseSpans(m[2]),
		Task:  &Task{Marked: strings.EqualFold(m[1], "x"), Row: index},
	}
}

// parseSpans quebra uma linha nos trechos marcados, deixando o resto como
// texto.
func parseSpans(text string) []Span {
	spans := []Span{}
	rest := text
	for len(rest) > 0 {
		pos := mdMarca.FindStringIndex(rest)
		if pos == nil {
			break
		}
		if pos[0] > 0 {
			spans = append(spans, Span{Kind: "text", Text: rest[:pos[0]]})
		}
		spans = append(spans, markedSpan(rest[pos[0]:pos[1]]))
		rest = rest[pos[1]:]
	}
	if len(rest) > 0 {
		spans = append(spans, Span{Kind: "text", Text: rest})
	}
	return joinTexts(spans)
}

// joinTexts cola textos vizinhos.
//
// Um trecho RECUSADO — um link que não é http, um parêntese fechando cedo — sai
// partido em dois pedaços de texto, e o que o mestre escreveu tem de voltar
// inteiro. Nunca comer o que foi escrito é a regra desta gramática.
func joinTexts(spans []Span) []Span {
	outside := []Span{}
	for _, s := range spans {
		last := len(outside) - 1
		if s.Kind == "text" && last >= 0 && outside[last].Kind == "text" {
			outside[last].Text += s.Text
			continue
		}
		outside = append(outside, s)
	}
	return outside
}

func markedSpan(token string) Span {
	switch {
	case strings.HasPrefix(token, "`"):
		return Span{Kind: "code", Text: token[1 : len(token)-1]}
	case strings.HasPrefix(token, "**"):
		return Span{Kind: "strong", Text: token[2 : len(token)-2]}
	case strings.HasPrefix(token, "["):
		return linkSpan(token)
	}
	return Span{Kind: "em", Text: token[1 : len(token)-1]}
}

// linkSpan aceita SÓ http(s). Um `javascript:` vira TEXTO e não um link
// morto: quem escreveu vê o que escreveu, e nada navegável sai daqui.
func linkSpan(token string) Span {
	m := mdElo.FindStringSubmatch(token)
	if m == nil || m[1] == "" || m[2] == "" || !mdHTTP.MatchString(m[2]) {
		return Span{Kind: "text", Text: token}
	}
	return Span{Kind: "link", Text: m[1], Href: m[2]}
}

// ToggleTask marca ou desmarca a tarefa da linha, devolvendo o texto novo —
// o estado do quadrinho mora NA NOTA, e não ao lado dela.
//
// Linha que não é tarefa, ou fora da faixa, devolve a fonte INTACTA: o pedido
// veio de um clique numa tela que pode estar um remendo atrás, e reescrever por
// palpite estragaria a nota de quem está digitando.
//
// @example ToggleTask("- [ ] dar XP", 0, true) // "- [x] dar XP"
func ToggleTask(source string, row int, marked bool) string {
	rows := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	if row < 0 || row >= len(rows) {
		return source
	}
	if !mdTarefaRe.MatchString(strings.TrimSpace(rows[row])) {
		return source
	}
	novo := "[ ]"
	if marked {
		novo = "[x]"
	}
	// SÓ A PRIMEIRA ocorrência: um `ReplaceAll` aqui reescreveria também um
	// `[x]` que o mestre tenha escrito no MEIO do texto do item.
	pos := checkbox.FindStringIndex(rows[row])
	rows[row] = rows[row][:pos[0]] + novo + rows[row][pos[1]:]
	return strings.Join(rows, "\n")
}

// checkbox é só o `[ ]`/`[x]`, para a troca não tocar no resto da linha.
var checkbox = regexp.MustCompile(`\[[ xX]\]`)
