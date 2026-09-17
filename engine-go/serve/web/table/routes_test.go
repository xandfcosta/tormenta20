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
	nomes := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_]*)\s*:`)
	vistos := map[string]bool{}
	medidos := 0
	for _, achado := range nomes.FindAllStringSubmatch(tableSignalsExpr(), -1) {
		nome := achado[1]
		medidos++
		if vistos[nome] {
			t.Errorf("o sinal %q é declarado duas vezes — o segundo vence, e um gesto passa a escrever no alvo do outro", nome)
		}
		vistos[nome] = true
	}
	// O DENOMINADOR. Sem ele, "nenhum repetido" e "o regex não casou com nada"
	// são a mesma linha verde.
	if medidos < 25 {
		t.Fatalf("só %d sinais lidos — o guarda ficou cego", medidos)
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
	nomeDeSinal := regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9_]*)\s*:`)
	var declarados []string
	for _, achado := range nomeDeSinal.FindAllStringSubmatch(tableSignalsExpr(), -1) {
		declarados = append(declarados, achado[1])
	}

	root := filepath.Join("..", "..", "..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go", "*.ts").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}

	// Os QUATRO canais de leitura. O último é a constante, e ele é o que separa
	// "órfão" de "lido por um nome que o `grep` de `$` não conhece".
	expressao := regexp.MustCompile(`(?:^|[^$\w])\$([a-zA-Z_][a-zA-Z0-9_]*)`)
	tagDoServidor := regexp.MustCompile(`json:"([a-zA-Z_][a-zA-Z0-9_]*)"`)
	chaveDeAtributo := regexp.MustCompile(`data-(?:bind|indicator|ref|computed)[a-z-]*[:=]"?\$?([a-zA-Z_][a-zA-Z0-9_]*)`)
	// O `const` NÃO entra no padrão, e não é descuido: metade destas constantes
	// mora num bloco `const (…)`, onde a palavra fica na linha de cima. Exigi-la
	// custa falso positivo — um extrator ancorado no jeito comum de escrever não
	// mede o jeito incomum, e o que sobra parece resposta.
	constanteDeSinal := regexp.MustCompile(`^\s*(?:const\s+)?([a-zA-Z][a-zA-Z0-9_]*)\s*=\s*"([a-zA-Z_][a-zA-Z0-9_]*)"`)

	leitores := map[string]bool{}
	porConstante := map[string]string{}
	usos := map[string]int{}
	arquivos := 0
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		arquivos++
		for _, linha := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(linha), "//") {
				continue
			}
			for _, re := range []*regexp.Regexp{expressao, tagDoServidor, chaveDeAtributo} {
				for _, m := range re.FindAllStringSubmatch(linha, -1) {
					leitores[m[1]] = true
				}
			}
			for _, m := range constanteDeSinal.FindAllStringSubmatch(linha, -1) {
				porConstante[m[2]] = m[1]
			}
			for _, palavra := range regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9_]*`).FindAllString(linha, -1) {
				usos[palavra]++
			}
		}
	}
	// Uma constante só é LEITORA se alguém a usa: `const x = "sinal"` sozinha,
	// sem nenhuma referência a `x`, é a mesma dívida com outra roupa.
	for sinal, constante := range porConstante {
		if usos[constante] > 1 {
			leitores[sinal] = true
		}
	}

	// O DENOMINADOR. "Nenhum solto" e "o regex parou de casar" são a mesma cor
	// no terminal.
	if len(declarados) < 60 || arquivos < 300 {
		t.Fatalf("a varredura leu %d sinais declarados em %d arquivos — o extrator é o primeiro suspeito",
			len(declarados), arquivos)
	}

	var soltos []string
	for _, nome := range declarados {
		if !leitores[nome] {
			soltos = append(soltos, nome)
		}
	}
	sort.Strings(soltos)
	if len(soltos) > 0 {
		t.Errorf("sinal declarado pela Mesa que NINGUÉM lê — %d de %d:\n  %s\n"+
			"Leitor é `$nome`, uma tag `json:\"nome\"`, uma chave `data-bind:nome` ou uma constante "+
			"Go usada em algum lugar. Sem nenhum dos quatro: ou o renome deixou a declaração para "+
			"trás, ou o sinal morreu e ela ficou. Nos dois casos a expressão passa a ler `undefined` "+
			"— e `undefined != ''` é VERDADEIRO, então o nó que devia nascer escondido nasce visível.",
			len(soltos), len(declarados), strings.Join(soltos, "\n  "))
		return
	}
	t.Logf("sinais da Mesa: %d declarados, todos com leitor, de %d arquivos", len(declarados), arquivos)
}
