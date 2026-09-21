package convention

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TODA FAMÍLIA DE FONTE QUE A FOLHA ESCOLHE É UMA QUE O APP ENTREGA (ALE-362).
//
// # O defeito, e por que nenhum outro guarda o via
//
// `--font-sans: ui-sans-serif, system-ui, sans-serif` não é uma escolha: é um
// PEDIDO à máquina. Cada uma responde com a face que tiver — Liberation Sans
// nesta bancada, outra no runner do GitHub —, e as duas desenham o mesmo texto
// com larguras diferentes.
//
// Isso faz de toda medida de LARGURA uma propriedade da MÁQUINA. O corte
// lateral, o piso de toque e o contraste medem texto desenhado; com a face vindo
// de fora, "verde na bancada" e "verde na CI" deixam de ser a mesma afirmação.
// Foi assim que a `main` passou três merges vermelha com a suíte local verde: o
// número do atributo pedia 27px aqui e 37px lá, numa caixa de 34 (ALE-362).
//
// # Por que aqui e não no navegador
//
// O e2e prova o RESULTADO — `o número do atributo é desenhado com a fonte que o
// app entregou` usa o CDP para ler a face de verdade. Este guarda prende a
// DECISÃO, que é textual e não precisa de navegador: um token novo apontando
// para a máquina reprova no `go test`, antes de alguém abrir a tela.
//
// # A dívida mora numa base que só encolhe, e hoje ela está vazia
//
// Token que ainda pede a face da máquina entra em
// `testdata/system_font_tokens.txt` em vez de virar exceção escrita aqui —
// dívida registrada e visível, que reprova quando for paga e ninguém tirar a
// linha. Nenhum token está lá: o `--font-mono`, que nasceu na base, saiu na
// mesma fatia quando a JetBrains Mono passou a ser entregue.
var (
	fontFaceFamily = regexp.MustCompile(`(?s)@font-face\s*\{[^}]*?font-family:\s*'([^']+)'`)
	fontToken      = regexp.MustCompile(`(--font-[a-z-]+):\s*([^;]+);`)
	varReference   = regexp.MustCompile(`^var\((--[a-z-]+)\)$`)
)

func TestEveryFontTokenNamesAFaceTheAppShips(t *testing.T) {
	source := filepath.Join("..", "serve", "web", "assets", "src", "index.css")
	sheet, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ler a folha-fonte: %v", err)
	}

	shipped := map[string]bool{}
	for _, found := range fontFaceFamily.FindAllStringSubmatch(string(sheet), -1) {
		shipped[strings.ToLower(found[1])] = true
	}
	if len(shipped) == 0 {
		t.Fatalf("nenhum @font-face em %s — o guarda está lendo a folha errada", source)
	}

	declarations := map[string]string{}
	for _, found := range fontToken.FindAllStringSubmatch(string(sheet), -1) {
		declarations[found[1]] = found[2]
	}

	base := systemFontBaseline(t)
	measured, inBaseline := 0, map[string]bool{}
	for _, found := range fontToken.FindAllStringSubmatch(string(sheet), -1) {
		token, stack := found[1], found[2]
		measured++
		first := firstFamily(stack, declarations)
		if shipped[first] {
			continue
		}
		if base[token] {
			inBaseline[token] = true
			continue
		}
		t.Errorf("%s começa em %q, que é a fonte da MÁQUINA e não uma que o app entrega.\n"+
			"Largura de texto desenhado com face de fora é propriedade da máquina: a bancada\n"+
			"e a CI passam a medir coisas diferentes, e a tela muda de tamanho na mão de quem\n"+
			"não tem a mesma fonte. Declare um @font-face e ponha a família na frente da pilha\n"+
			"— ou registre a dívida em `convention/testdata/system_font_tokens.txt`, dizendo por quê.",
			token, first)
	}

	// A DÍVIDA QUE FOI PAGA sai da lista, como na base dos arquivos longos.
	paid := []string{}
	for token := range base {
		if !inBaseline[token] {
			paid = append(paid, token)
		}
	}
	sort.Strings(paid)
	for _, token := range paid {
		t.Errorf("%s está na base de tokens de fonte do sistema e hoje aponta para uma face entregue.\n"+
			"Tire a linha dele de `convention/testdata/system_font_tokens.txt`.", token)
	}

	// O DENOMINADOR: uma folha sem `--font-*` nenhum e uma folha em conformidade
	// se parecem no terminal.
	if measured < 3 {
		t.Fatalf("o guarda achou só %d token de fonte — está lendo a folha errada", measured)
	}
	t.Logf("%d tokens de fonte, %d famílias entregues, %d na dívida", measured, len(shipped), len(base))
}

// firstFamily devolve a primeira família da pilha, seguindo o `var(--outro)`.
//
// A indireção é o desenho e não um acidente: `--font-display: var(--font-sans)`
// existe para a face de corpo ser escolhida em UM lugar, e um guarda que lesse
// "var(--font-sans)" como nome de família cobraria um @font-face chamado
// `var(...)`. O teto de saltos existe porque CSS aceita escrever um ciclo.
func firstFamily(stack string, declarations map[string]string) string {
	for jump := 0; jump < 5; jump++ {
		first := strings.ToLower(strings.Trim(strings.TrimSpace(strings.Split(stack, ",")[0]), `'"`))
		pointed := varReference.FindStringSubmatch(first)
		if pointed == nil {
			return first
		}
		next, ok := declarations[pointed[1]]
		if !ok {
			return first
		}
		stack = next
	}
	return "ciclo de var()"
}

// systemFontBaseline lê a dívida registrada — um token por linha, `#` comenta.
func systemFontBaseline(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "system_font_tokens.txt"))
	if err != nil {
		t.Fatalf("ler a linha de base: %v", err)
	}
	base := map[string]bool{}
	for _, row := range strings.Split(string(raw), "\n") {
		row = strings.TrimSpace(row)
		if row != "" && !strings.HasPrefix(row, "#") {
			base[row] = true
		}
	}
	return base
}
