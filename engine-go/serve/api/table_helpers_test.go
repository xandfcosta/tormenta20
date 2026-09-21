package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"t20engine/domain/engine"
	"testing"

	"golang.org/x/net/html"
)

// Os ajudantes de leitura da cena da Mesa, COPIADOS de `web/table`.
//
// A cópia é deliberada: importar o ajudante do pacote que está sendo testado faz
// o teste andar junto com o defeito.

const blocoMinimo = `"nd":1,"tipo":"humanoide","size":"medio","hp":10,"defesa":10,` +
	`"deslocamento":"9m (6q)","attacks":[],"skills":[],"specialAbilities":[]`

func bodyDraft(inside string) string {
	return `{"draft":{` + inside + `}}`
}

// compiledStylesheet é a folha que o NAVEGADOR recebe, e é sempre ela que os
// guardas leem — nunca a fonte.
//
// Uma classe que o scanner do Tailwind não viu não existe na folha, e é
// justamente esse o modo de falhar que não dá erro (ver o `engine-go/CLAUDE.md`).
// Ler a fonte faria todo guarda desta família passar verde sobre tinta que o
// navegador nunca recebeu.
func compiledStylesheet(t *testing.T) string {
	t.Helper()
	sheet, err := os.ReadFile(filepath.Join("../web/assets", "static", "app.css"))
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	return string(sheet)
}

// classesThatReceiveBox lê a folha COMPILADA e devolve as classes de toda
// regra que resolve o `--col` em pixels.
func classesThatReceiveBox(t *testing.T) map[string]bool {
	t.Helper()
	sheet := compiledStylesheet(t)
	classes := map[string]bool{}
	for _, rule := range strings.Split(sheet, "}") {
		opens := strings.Index(rule, "{")
		if opens < 0 || !strings.Contains(rule[opens:], "left:calc(var(--col)") {
			continue
		}
		for _, selector := range strings.Split(rule[:opens], ",") {
			if name := strings.TrimPrefix(strings.TrimSpace(selector), "."); name != "" {
				classes[name] = true
			}
		}
	}
	return classes
}

// collectionRow recorta o `<li>` que contém uma marca, e FALHA quando não acha.
//
// Falhar em vez de devolver vazio é o que separa este helper de um instrumento
// mudo: uma busca que não acha nada faria toda asserção seguinte passar sobre
// uma string vazia — o `strings.Contains(vazio, x)` é falso, e "não contém" é
// exatamente o que a maioria dos guardas daqui afirma.
func collectionRow(t *testing.T, screen, mark string) string {
	t.Helper()
	pos := strings.Index(screen, mark)
	if pos < 0 {
		t.Fatalf("não achei %q na tela: a asserção seguinte mediria uma string vazia", mark)
	}
	start := strings.LastIndex(screen[:pos], "<li ")
	end := strings.Index(screen[pos:], "</li>")
	if start < 0 || end < 0 {
		t.Fatalf("a marca %q não está dentro de um <li> do acervo", mark)
	}
	return screen[start : pos+end]
}

func contem(squares []engine.Square, target engine.Square) bool {
	for _, c := range squares {
		if c == target {
			return true
		}
	}
	return false
}

// element acha o primeiro elemento cujo atributo `attribute` contém `excerpt`,
// e devolve os atributos dele.
//
// Um parser de HTML de verdade e não uma expressão regular, e a razão é o guarda
// do fim deste arquivo: as expressões do Datastar carregam `<`, `>` e aspas
// dentro dos valores, e um `<[^>]*>` corta um elemento no meio de um `data-on:`
// sem avisar — a busca devolveria menos e a ausência viraria conclusão.
func element(t *testing.T, screen, attribute, excerpt string) map[string]string {
	t.Helper()
	z := html.NewTokenizer(strings.NewReader(screen))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil
		case html.StartTagToken, html.SelfClosingTagToken:
			attrs := map[string]string{}
			for {
				key, value, more := z.TagAttr()
				attrs[string(key)] = string(value)
				if !more {
					break
				}
			}
			if strings.Contains(attrs[attribute], excerpt) {
				return attrs
			}
		}
	}
}

func firstRows(s string, n int) string {
	rows := strings.Split(s, "\n")
	if len(rows) > n {
		rows = rows[:n]
	}
	return strings.Join(rows, "\n")
}

// primeirosAtributos encurta a tag para a mensagem caber na saída do teste.
func primeirosAtributos(tag string) string {
	if len(tag) > 160 {
		return tag[:160] + "…"
	}
	return tag
}
func quadrados(pares ...[2]int) []engine.Square {
	qs := make([]engine.Square, len(pares))
	for i, p := range pares {
		qs[i] = engine.Square{X: p[0], Y: p[1]}
	}
	return qs
}

// responseDraft extrai o rascunho do quadro de sinais do SSE.
//
// Ler a CHAVE e não procurar o texto solto na resposta: `Contains(resposta,
// "Ogro Capitão")` passa verde com o sinal renomeado, porque o nome continua no
// corpo — ligado a coisa nenhuma. O que a tela precisa é do valor sob
// `rascunho`.
func responseDraft(t *testing.T, response string) map[string]any {
	t.Helper()
	const marker = "data: signals "
	i := strings.Index(response, marker)
	if i < 0 {
		t.Fatalf("a resposta não trouxe sinais:\n%s", response)
	}
	row := response[i+len(marker):]
	if end := strings.IndexByte(row, '\n'); end >= 0 {
		row = row[:end]
	}
	var signals struct {
		Draft map[string]any `json:"draft"`
	}
	if err := json.Unmarshal([]byte(row), &signals); err != nil {
		t.Fatalf("os sinais não são JSON: %v\n%s", err, row)
	}
	if signals.Draft == nil {
		t.Fatalf("a resposta não trouxe `rascunho`:\n%s", row)
	}
	return signals.Draft
}

// signals escreve os sinais do jeito que o Datastar os manda num GET: um
// parâmetro `datastar` com o JSON inteiro.
//
// Query param solto (`?criatura=zumbi`) NÃO é a mesma coisa: o
// `master.BestiaryCriteriaFromRequest` lê os dois, mas o rascunho só existe como
// SINAL — um pedido que o navegador nunca manda, e o painel semeia por não achar
// o rascunho. O teste acusaria o código por um defeito dele mesmo.
func signals(json string) string {
	return "?datastar=" + url.QueryEscape(json)
}

var tableRegionNames = []string{
	"table-header",
	"table-register",
	"table-party",
	"table-board",
	"table-populate",
	"table-archive",
	"table-session-config",
	"table-tracker",
	"table-commands",
}

// temAlgumaClasse: basta UMA classe posicionada, porque o elemento veste várias —
// o fantasma é `board-token board-token-ghost`, e quem lhe dá caixa é a
// primeira.
func temAlgumaClasse(list string, sought map[string]bool) bool {
	for _, c := range strings.Fields(list) {
		if sought[c] {
			return true
		}
	}
	return false
}

// trechoDaSemeadura tira só o pedaço da expressão que semeia o nome, porque a
// página inteira enterra a asserção em vários KB de HTML.
func trechoDaSemeadura(body string) string {
	i := strings.Index(body, "$edit_name = ")
	if i < 0 {
		return "(a semeadura do nome não está na página)"
	}
	end := i + 120
	if end > len(body) {
		end = len(body)
	}
	return body[i:end]
}

// trechoDeSinais tira só a linha dos sinais da resposta SSE, porque o quadro
// inteiro traz a cena e enterra a asserção em 8 KB de HTML.
func trechoDeSinais(body string) string {
	for _, row := range strings.Split(body, "\n") {
		if strings.HasPrefix(row, "data: signals ") {
			return row
		}
	}
	return "(nenhuma linha de sinais na resposta)"
}

// ── editar o combatente ─────────────────────────────────────────────────────

// stroke monta o CORPO de um gesto de pincel.
//
// Espécie vazia é a BORRACHA, que não nomeia espécie nenhuma — nem no caminho
// nem no corpo. É a espécie que a faz apagar a coisa errada em silêncio, e o
// corpo não devolve esse campo de graça.
func stroke(species string, x, y, x2, y2 int) string {
	if species == "" {
		return fmt.Sprintf(`{"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`, x, y, x2, y2)
	}
	return fmt.Sprintf(`{"kind":%q,"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`,
		species, x, y, x2, y2)
}

// strokeErasing é o traço que apaga AQUELA espécie, e não a casa inteira.
//
// São coisas diferentes e por isso não são a mesma rota: esta nomeia a espécie e
// a borracha (`/terreno/limpar`) não nomeia nenhuma.
func strokeErasing(species string, x, y, x2, y2 int) string {
	return fmt.Sprintf(`{"kind":%q,"erase":true,"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`,
		species, x, y, x2, y2)
}

// templateBody monta o CORPO do gabarito: a forma, o tamanho, a origem e a mira.
// Os dois pontos usam os mesmos `from`/`to` do traço — um formato só para o
// tabuleiro inteiro.
func templateBody(form, size string, x, y, mx, my int) string {
	return fmt.Sprintf(`{"shape":%q,"size":%q,"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`,
		form, size, x, y, mx, my)
}
