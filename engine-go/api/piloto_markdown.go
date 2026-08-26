package api

import (
	"regexp"
	"strings"
)

// O MARKDOWN DAS NOTAS DA SESSÃO (ALE-269), portado de
// `frontend/src/shared/lib/markdown.ts`.
//
// É um markdown PEQUENO, do tamanho de uma nota de mesa, e ele produz uma
// ÁRVORE — nunca HTML. Quem desenha é o templ, montando elementos a partir
// daqui, então não existe `innerHTML` no caminho e injeção é impossível por
// construção: sem parser de terceiro e sem sanitizador atrás dele. É o mesmo
// argumento que a SPA escreveu, e ele atravessa a fronteira intacto porque o
// templ escapa texto pelo mesmo motivo que o Solid escapa.
//
// POR QUE PORTAR EM VEZ DE PUXAR UM GOLDMARK: as duas telas desenham a MESMA
// nota do banco enquanto a migração durar, e as divergências desta gramática
// não são detalhe — são as decisões que custaram caro. O comportamento PADRÃO
// de um parser CommonMark junta linhas soltas num parágrafo só, que é
// exatamente o defeito que a ALE-122 consertou aqui.
//
// A paridade é MEDIDA e não afirmada: `piloto_markdown_test.go` compara esta
// árvore com a que o JS produz, a partir de um oráculo gerado por
// `scripts/dump-markdown-oracle.ts`.

// mdSpan é um trecho de uma linha. `Href` só existe em `elo`.
//
// As etiquetas JSON são as do TS de propósito: é o que deixa o teste comparar
// as duas árvores sem uma terceira tradução no meio.
type mdSpan struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
	Href string `json:"href,omitempty"`
}

// mdTarefa é o estado de um `- [ ]`, com a LINHA de origem junto.
//
// A linha não é enfeite: é ela que deixa o clique no quadrinho reescrever o
// texto do mestre. Sem ela o controle seria decorativo e o estado moraria fora
// da nota, que é onde ele não sobrevive a um F5.
type mdTarefa struct {
	Marcada bool `json:"checked"`
	Linha   int  `json:"line"`
}

type mdItem struct {
	Spans  []mdSpan  `json:"spans"`
	Tarefa *mdTarefa `json:"task,omitempty"`
}

// mdBloco é um bloco da nota. `Kind` diz qual dos campos vale — uma struct só
// em vez de uma interface porque quem consome é um `switch` de template, e uma
// hierarquia de tipos aqui compraria indireção sem comprar nada.
type mdBloco struct {
	Kind     string     `json:"kind"`
	Nivel    int        `json:"level,omitempty"`
	Ordenada bool       `json:"ordered"`
	Spans    []mdSpan   `json:"spans,omitempty"`
	Linhas   [][]mdSpan `json:"lines,omitempty"`
	Itens    []mdItem   `json:"items,omitempty"`
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

// parseNota traduz o texto da nota na árvore que a tela desenha.
//
// @example parseNota("# Cena 1\n- Ogro **fugiu**")
func parseNota(fonte string) []mdBloco {
	blocos := []mdBloco{}
	var paragrafo []string
	linhas := strings.Split(strings.ReplaceAll(fonte, "\r\n", "\n"), "\n")
	for i := 0; i < len(linhas); i++ {
		linha := strings.TrimSpace(linhas[i])
		if linha == "" {
			blocos, paragrafo = fechaParagrafo(blocos, paragrafo)
			continue
		}
		if bloco, ok := blocoDeUmaLinha(linha); ok {
			blocos, paragrafo = fechaParagrafo(blocos, paragrafo)
			blocos = append(blocos, bloco)
			continue
		}
		if mdItemRe.MatchString(linha) || mdOrdenada.MatchString(linha) {
			blocos, paragrafo = fechaParagrafo(blocos, paragrafo)
			var lista mdBloco
			lista, i = colheALista(linhas, i)
			blocos = append(blocos, lista)
			continue
		}
		paragrafo = append(paragrafo, linha)
	}
	blocos, _ = fechaParagrafo(blocos, paragrafo)
	return blocos
}

// blocoDeUmaLinha resolve os três blocos que cabem numa linha só. Falso devolve
// a decisão a quem chamou — lista precisa de várias linhas, e o resto é texto.
func blocoDeUmaLinha(linha string) (mdBloco, bool) {
	if m := mdTitulo.FindStringSubmatch(linha); m != nil {
		return mdBloco{Kind: "heading", Nivel: len(m[1]), Spans: parseTrechos(m[2])}, true
	}
	if mdRegua.MatchString(linha) {
		return mdBloco{Kind: "rule"}, true
	}
	if m := mdCitacao.FindStringSubmatch(linha); m != nil {
		return mdBloco{Kind: "quote", Spans: parseTrechos(m[1])}, true
	}
	return mdBloco{}, false
}

// fechaParagrafo despeja as linhas acumuladas num bloco.
//
// CADA LINHA DIGITADA É UMA LINHA NA TELA (ALE-122). Numa nota de mesa a quebra
// é intencional, e juntá-las como o markdown padrão manda transformava trinta
// linhas de anotação num parágrafo só. É a divergência que faz este port existir
// em vez de uma dependência.
func fechaParagrafo(blocos []mdBloco, paragrafo []string) ([]mdBloco, []string) {
	if len(paragrafo) == 0 {
		return blocos, paragrafo
	}
	linhas := make([][]mdSpan, 0, len(paragrafo))
	for _, l := range paragrafo {
		linhas = append(linhas, parseTrechos(l))
	}
	return append(blocos, mdBloco{Kind: "paragraph", Linhas: linhas}), nil
}

// colheALista junta as linhas seguidas de uma lista num bloco só e devolve o
// índice da ÚLTIMA consumida — itens soltos viravam um bloco por linha, e a
// marcação de lista se perdia.
func colheALista(linhas []string, inicio int) (mdBloco, int) {
	ordenada := mdOrdenada.MatchString(strings.TrimSpace(linhas[inicio]))
	itens := []mdItem{}
	i := inicio
	for ; i < len(linhas); i++ {
		linha := strings.TrimSpace(linhas[i])
		re := mdItemRe
		if ordenada {
			re = mdOrdenada
		}
		m := re.FindStringSubmatch(linha)
		if m == nil {
			break
		}
		itens = append(itens, itemDaLista(linha, i, m[1]))
	}
	return mdBloco{Kind: "list", Ordenada: ordenada, Itens: itens}, i - 1
}

// itemDaLista: `- [ ] dar XP` é um item com ESTADO; qualquer outro é comum.
func itemDaLista(linha string, indice int, texto string) mdItem {
	m := mdTarefaRe.FindStringSubmatch(linha)
	if m == nil {
		return mdItem{Spans: parseTrechos(texto)}
	}
	return mdItem{
		Spans:  parseTrechos(m[2]),
		Tarefa: &mdTarefa{Marcada: strings.EqualFold(m[1], "x"), Linha: indice},
	}
}

// parseTrechos quebra uma linha nos trechos marcados, deixando o resto como
// texto.
func parseTrechos(texto string) []mdSpan {
	spans := []mdSpan{}
	resto := texto
	for len(resto) > 0 {
		pos := mdMarca.FindStringIndex(resto)
		if pos == nil {
			break
		}
		if pos[0] > 0 {
			spans = append(spans, mdSpan{Kind: "text", Text: resto[:pos[0]]})
		}
		spans = append(spans, trechoMarcado(resto[pos[0]:pos[1]]))
		resto = resto[pos[1]:]
	}
	if len(resto) > 0 {
		spans = append(spans, mdSpan{Kind: "text", Text: resto})
	}
	return juntaOsTextos(spans)
}

// juntaOsTextos cola textos vizinhos.
//
// Um trecho RECUSADO — um link que não é http, um parêntese fechando cedo — sai
// partido em dois pedaços de texto, e o que o mestre escreveu tem de voltar
// inteiro. Nunca comer o que foi escrito é a regra desta gramática.
func juntaOsTextos(spans []mdSpan) []mdSpan {
	fora := []mdSpan{}
	for _, s := range spans {
		ultimo := len(fora) - 1
		if s.Kind == "text" && ultimo >= 0 && fora[ultimo].Kind == "text" {
			fora[ultimo].Text += s.Text
			continue
		}
		fora = append(fora, s)
	}
	return fora
}

func trechoMarcado(token string) mdSpan {
	switch {
	case strings.HasPrefix(token, "`"):
		return mdSpan{Kind: "code", Text: token[1 : len(token)-1]}
	case strings.HasPrefix(token, "**"):
		return mdSpan{Kind: "strong", Text: token[2 : len(token)-2]}
	case strings.HasPrefix(token, "["):
		return trechoDeElo(token)
	}
	return mdSpan{Kind: "em", Text: token[1 : len(token)-1]}
}

// trechoDeElo aceita SÓ http(s). Um `javascript:` vira TEXTO e não um link
// morto: quem escreveu vê o que escreveu, e nada navegável sai daqui.
func trechoDeElo(token string) mdSpan {
	m := mdElo.FindStringSubmatch(token)
	if m == nil || m[1] == "" || m[2] == "" || !mdHTTP.MatchString(m[2]) {
		return mdSpan{Kind: "text", Text: token}
	}
	return mdSpan{Kind: "link", Text: m[1], Href: m[2]}
}

// alternaTarefa marca ou desmarca a tarefa da linha, devolvendo o texto novo —
// o estado do quadrinho mora NA NOTA, e não ao lado dela.
//
// Linha que não é tarefa, ou fora da faixa, devolve a fonte INTACTA: o pedido
// veio de um clique numa tela que pode estar um remendo atrás, e reescrever por
// palpite estragaria a nota de quem está digitando.
//
// @example alternaTarefa("- [ ] dar XP", 0, true) // "- [x] dar XP"
func alternaTarefa(fonte string, linha int, marcada bool) string {
	linhas := strings.Split(strings.ReplaceAll(fonte, "\r\n", "\n"), "\n")
	if linha < 0 || linha >= len(linhas) {
		return fonte
	}
	if !mdTarefaRe.MatchString(strings.TrimSpace(linhas[linha])) {
		return fonte
	}
	novo := "[ ]"
	if marcada {
		novo = "[x]"
	}
	// SÓ A PRIMEIRA ocorrência, porque é o que o `String.replace` do JS faz sem
	// a bandeira `g`. Um `ReplaceAll` aqui reescreveria também um `[x]` que o
	// mestre tenha escrito no MEIO do texto do item — divergência silenciosa
	// entre as duas telas sobre a mesma nota.
	pos := mdQuadrinho.FindStringIndex(linhas[linha])
	linhas[linha] = linhas[linha][:pos[0]] + novo + linhas[linha][pos[1]:]
	return strings.Join(linhas, "\n")
}

// mdQuadrinho é só o `[ ]`/`[x]`, para a troca não tocar no resto da linha.
var mdQuadrinho = regexp.MustCompile(`\[[ xX]\]`)
