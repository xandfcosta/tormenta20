package table

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// NENHUM SINAL DA MESA É DECLARADO DUAS VEZES.
//
// A cena declara dezenas de sinais numa expressão só, e eles vivem todos no
// MESMO documento. Dois gestos com o mesmo nome não dão erro em lugar nenhum: o
// segundo `data-signals` simplesmente vence, e o que se vê é um gesto
// escrevendo no alvo do outro — o de criar peça apagando o alvo do de salvar.
func TestNoTableSignalIsDeclaredTwice(t *testing.T) {
	// O `_` PRECISA estar na classe, e a falta dele já mentiu: com
	// `([a-zA-Z][a-zA-Z0-9]*)` o `ruler_aim_x: 0` casa só o `x:`, e o guarda
	// acusa sete duplicatas inexistentes. Parser que não entende a forma nova
	// produz lista de falhas com cara de descoberta.
	names := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_]*)\s*:`)
	seen := map[string]bool{}
	measured := 0
	for _, found := range names.FindAllStringSubmatch(tableSignalsExpr(), -1) {
		name := found[1]
		measured++
		if seen[name] {
			t.Errorf("o sinal %q é declarado duas vezes — o segundo vence, e um gesto passa a escrever no alvo do outro", name)
		}
		seen[name] = true
	}
	// O DENOMINADOR. Sem ele, "nenhum repetido" e "o regex não casou com nada"
	// são a mesma linha verde.
	if measured < 25 {
		t.Fatalf("só %d sinais lidos — o guarda ficou cego", measured)
	}
}

// TODO SINAL QUE A MESA DECLARA TEM QUEM O LEIA.
//
// # O modo de falha
//
// Um renome que troca o LEITOR (`$error`) e o escritor do servidor
// (`json:"error"`) e deixa a DECLARAÇÃO para trás não estoura em lugar nenhum:
// a expressão passa a ler `undefined`, e `undefined != ”` é VERDADEIRO — então
// o `<p>` da recusa nasce mostrado, vazio. É a forma desta família: o gesto não
// faz nada, ou faz demais, e não há erro em parte alguma.
//
// # Por que o guarda da `convention` não alcança
//
// O `TestEverySignalDeclaredByValueHasAReader` lê o VALOR de um atributo
// (`data-ref="x"`) — o canal 3 dos sete da tabela do `CLAUDE.md`. A declaração
// da Mesa é o canal 4: uma string MONTADA EM GO. Os dois canais precisam de
// varredura própria, e este é o da Mesa.
//
// # A CONSTANTE é leitor, e é o que separa órfão de falso positivo
//
// Sinal lido por constante (`const brushSignal = "pincelando"`, usada como
// `"$"+…`) é invisível para um inventário por `$nome` — cinco sinais vivos
// apareceriam como órfãos sem este canal.
func TestEverySignalTheTableDeclaresHasAReader(t *testing.T) {
	signalName := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_]*)\s*:`)
	var declared []string
	for _, found := range signalName.FindAllStringSubmatch(tableSignalsExpr(), -1) {
		declared = append(declared, found[1])
	}

	root := filepath.Join("..", "..", "..", "..")
	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go", "*.ts").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}

	// Os QUATRO canais de leitura. O último é a constante, e ele é o que separa
	// "órfão" de "lido por um nome que o `grep` de `$` não conhece".
	expression := regexp.MustCompile(`(?:^|[^$\w])\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	serverTag := regexp.MustCompile(`json:"([a-zA-Z_][a-zA-Z0-9_]*)"`)
	attributeKey := regexp.MustCompile(`data-(?:bind|indicator|ref|computed)[a-z-]*[:=]"?\$?([a-zA-Z_][a-zA-Z0-9_]*)`)
	// O `const` NÃO entra no padrão, e não é descuido: metade destas constantes
	// mora num bloco `const (…)`, onde a palavra fica na linha de cima. Exigi-la
	// custa falso positivo — um extrator ancorado no jeito comum de escrever não
	// mede o jeito incomum, e o que sobra parece resposta.
	signalConstant := regexp.MustCompile(`^\s*(?:const\s+)?([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*"([a-zA-Z_][a-zA-Z0-9_]*)"`)

	readers := map[string]bool{}
	byConstant := map[string]string{}
	uses := map[string]int{}
	files := 0
	for _, relative := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		files++
		for _, row := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(row), "//") {
				continue
			}
			for _, re := range []*regexp.Regexp{expression, serverTag, attributeKey} {
				for _, m := range re.FindAllStringSubmatch(row, -1) {
					readers[m[1]] = true
				}
			}
			for _, m := range signalConstant.FindAllStringSubmatch(row, -1) {
				byConstant[m[2]] = m[1]
			}
			for _, word := range regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_]*`).FindAllString(row, -1) {
				uses[word]++
			}
		}
	}
	// Uma constante só é LEITORA se alguém a usa: `const x = "sinal"` sozinha,
	// sem nenhuma referência a `x`, é a mesma dívida com outra roupa.
	for signal, constant := range byConstant {
		if uses[constant] > 1 {
			readers[signal] = true
		}
	}

	// O DENOMINADOR. "Nenhum solto" e "o regex parou de casar" são a mesma cor
	// no terminal.
	if len(declared) < 60 || files < 300 {
		t.Fatalf("a varredura leu %d sinais declarados em %d arquivos — o extrator é o primeiro suspeito",
			len(declared), files)
	}

	var loose []string
	for _, name := range declared {
		if !readers[name] {
			loose = append(loose, name)
		}
	}
	sort.Strings(loose)
	if len(loose) > 0 {
		t.Errorf("sinal declarado pela Mesa que NINGUÉM lê — %d de %d:\n  %s\n"+
			"Leitor é `$nome`, uma tag `json:\"nome\"`, uma chave `data-bind:nome` ou uma constante "+
			"Go usada em algum lugar. Sem nenhum dos quatro: ou o renome deixou a declaração para "+
			"trás, ou o sinal morreu e ela ficou. Nos dois casos a expressão passa a ler `undefined` "+
			"— e `undefined != ''` é VERDADEIRO, então o nó que devia nascer escondido nasce visível.",
			len(loose), len(declared), strings.Join(loose, "\n  "))
		return
	}
	t.Logf("sinais da Mesa: %d declarados, todos com leitor, de %d arquivos", len(declared), files)
}
