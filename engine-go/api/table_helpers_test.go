package api

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"t20engine/engine"
	"testing"

	"golang.org/x/net/html"
)

// Os ajudantes de leitura da cena da Mesa, COPIADOS de `web/table`
// (ALE-278).
//
// A cópia tem precedente e razão: importar o ajudante do pacote que está
// sendo testado faz o teste andar junto com o defeito. Foi a regra que a
// fatia da porta deixou escrita, e o que a cena de personagens fez com o
// `corpoDoBotao`.

const blocoMinimo = `"nd":1,"tipo":"humanoide","size":"medio","hp":10,"defesa":10,` +
	`"deslocamento":"9m (6q)","attacks":[],"skills":[],"specialAbilities":[]`

func bodyDraft(dentro string) string {
	return `{"draft":{` + dentro + `}}`
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
	folha, err := os.ReadFile(filepath.Join("piloto", "static", "piloto.css"))
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	return string(folha)
}

// classesThatReceiveBox lê a folha COMPILADA e devolve as classes de toda
// regra que resolve o `--col` em pixels.
func classesThatReceiveBox(t *testing.T) map[string]bool {
	t.Helper()
	folha := compiledStylesheet(t)
	classes := map[string]bool{}
	for _, regra := range strings.Split(folha, "}") {
		abre := strings.Index(regra, "{")
		if abre < 0 || !strings.Contains(regra[abre:], "left:calc(var(--col)") {
			continue
		}
		for _, seletor := range strings.Split(regra[:abre], ",") {
			if nome := strings.TrimPrefix(strings.TrimSpace(seletor), "."); nome != "" {
				classes[nome] = true
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
func collectionRow(t *testing.T, tela, marca string) string {
	t.Helper()
	pos := strings.Index(tela, marca)
	if pos < 0 {
		t.Fatalf("não achei %q na tela: a asserção seguinte mediria uma string vazia", marca)
	}
	inicio := strings.LastIndex(tela[:pos], "<li ")
	fim := strings.Index(tela[pos:], "</li>")
	if inicio < 0 || fim < 0 {
		t.Fatalf("a marca %q não está dentro de um <li> do acervo", marca)
	}
	return tela[inicio : pos+fim]
}

func contem(casas []engine.Square, alvo engine.Square) bool {
	for _, c := range casas {
		if c == alvo {
			return true
		}
	}
	return false
}

// element acha o primeiro elemento cujo atributo `atributo` contém `trecho`,
// e devolve os atributos dele.
//
// Um parser de HTML de verdade e não uma expressão regular, e a razão é o guarda
// do fim deste arquivo: as expressões do Datastar carregam `<`, `>` e aspas
// dentro dos valores, e um `<[^>]*>` corta um elemento no meio de um `data-on:`
// sem avisar — a busca devolveria menos e a ausência viraria conclusão.
func element(t *testing.T, tela, atributo, trecho string) map[string]string {
	t.Helper()
	z := html.NewTokenizer(strings.NewReader(tela))
	for {
		switch z.Next() {
		case html.ErrorToken:
			return nil
		case html.StartTagToken, html.SelfClosingTagToken:
			attrs := map[string]string{}
			for {
				chave, valor, mais := z.TagAttr()
				attrs[string(chave)] = string(valor)
				if !mais {
					break
				}
			}
			if strings.Contains(attrs[atributo], trecho) {
				return attrs
			}
		}
	}
}

func firstRows(s string, n int) string {
	linhas := strings.Split(s, "\n")
	if len(linhas) > n {
		linhas = linhas[:n]
	}
	return strings.Join(linhas, "\n")
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
// Ler a CHAVE e não procurar o texto solto na resposta, e isto custou uma
// sabotagem para descobrir: `Contains(resposta, "Ogro Capitão")` passa verde com
// o sinal renomeado, porque o nome continua no corpo — ligado a coisa nenhuma. O
// que a tela precisa é do valor sob `rascunho`, e é isso que se afirma.
func responseDraft(t *testing.T, resposta string) map[string]any {
	t.Helper()
	const marca = "data: signals "
	i := strings.Index(resposta, marca)
	if i < 0 {
		t.Fatalf("a resposta não trouxe sinais:\n%s", resposta)
	}
	linha := resposta[i+len(marca):]
	if fim := strings.IndexByte(linha, '\n'); fim >= 0 {
		linha = linha[:fim]
	}
	var sinais struct {
		Rascunho map[string]any `json:"draft"`
	}
	if err := json.Unmarshal([]byte(linha), &sinais); err != nil {
		t.Fatalf("os sinais não são JSON: %v\n%s", err, linha)
	}
	if sinais.Rascunho == nil {
		t.Fatalf("a resposta não trouxe `rascunho`:\n%s", linha)
	}
	return sinais.Rascunho
}

// signals escreve os sinais do jeito que o Datastar os manda num GET: um
// parâmetro `datastar` com o JSON inteiro.
//
// A primeira versão deste teste usava query params soltos (`?criatura=zumbi`), e
// eles NÃO são a mesma coisa: o `master.BestiaryCriteriaFromRequest` lê os dois, mas o
// `rascunhode` só existe como sinal — então o teste mandava um pedido que o
// navegador nunca manda, e o painel semeava por não achar o rascunho. O teste
// acusou o código por um defeito que era dele.
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
func temAlgumaClasse(lista string, procuradas map[string]bool) bool {
	for _, c := range strings.Fields(lista) {
		if procuradas[c] {
			return true
		}
	}
	return false
}

// trechoDaSemeadura tira só o pedaço da expressão que semeia o nome, porque a
// página inteira enterra a asserção em vários KB de HTML.
func trechoDaSemeadura(corpo string) string {
	i := strings.Index(corpo, "$edit_name = ")
	if i < 0 {
		return "(a semeadura do nome não está na página)"
	}
	fim := i + 120
	if fim > len(corpo) {
		fim = len(corpo)
	}
	return corpo[i:fim]
}

// trechoDeSinais tira só a linha dos sinais da resposta SSE, porque o quadro
// inteiro traz a cena e enterra a asserção em 8 KB de HTML.
func trechoDeSinais(corpo string) string {
	for _, linha := range strings.Split(corpo, "\n") {
		if strings.HasPrefix(linha, "data: signals ") {
			return linha
		}
	}
	return "(nenhuma linha de sinais na resposta)"
}

// ── editar o combatente (ALE-263) ────────────────────────────────────────────

// stroke monta o CORPO de um gesto de pincel (ALE-305).
//
// Espécie vazia é a BORRACHA, que não nomeia espécie nenhuma — nem no caminho
// nem no corpo. Era a espécie que a fazia apagar a coisa errada em silêncio
// (ALE-203), e o corpo não devolve esse campo de graça.
func stroke(especie string, x, y, x2, y2 int) string {
	if especie == "" {
		return fmt.Sprintf(`{"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`, x, y, x2, y2)
	}
	return fmt.Sprintf(`{"kind":%q,"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`,
		especie, x, y, x2, y2)
}

// strokeErasing é o traço que apaga AQUELA espécie, e não a casa inteira.
//
// São coisas diferentes e por isso não são a mesma rota: esta nomeia a espécie e
// a borracha (`/terreno/limpar`) não nomeia nenhuma.
func strokeErasing(especie string, x, y, x2, y2 int) string {
	return fmt.Sprintf(`{"kind":%q,"erase":true,"from":{"X":%d,"Y":%d},"to":{"X":%d,"Y":%d}}`,
		especie, x, y, x2, y2)
}
