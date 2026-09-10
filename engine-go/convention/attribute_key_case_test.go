package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// CHAVE DE ATRIBUTO DO DATASTAR É MINÚSCULA, E NÃO É ESTILO (ALE-301).
//
// O analisador de HTML minuscula NOME DE ATRIBUTO. Um `data-bind:novoNome`
// chega ao Datastar como `data-bind:novonome` e liga um sinal NOVO — o
// declarado fica intocado, o fio leva os DOIS, e o servidor lê o errado. Só o
// VALOR de um atributo preserva a caixa, que é por que o `$fichaAberta` pode ser
// camelCase: ele só aparece dentro de expressões.
//
// # Por que este guarda existe, e por que ele existe TARDE
//
// A regra já estava escrita — **treze vezes**, em treze comentários espalhados
// por `web/table`, `web/sheetui`, `web/finder` e `web/master`, cada um contando
// a mesma história com um exemplo diferente. Ela nunca foi VARRIDA, e o
// resultado é o do `CLAUDE.md`: convenção escrita e não varrida vale exatamente
// nos arquivos que alguém apontou. O décimo quarto sítio — o `buscaCriatura` do
// construtor de encontros — tinha o defeito.
//
// **E ele estava MASCARADO**, que é por que ninguém o viu. Medido no navegador:
// o pedido sai com as duas chaves, `"buscaCriatura":""` (a declarada, intocada)
// e `"buscacriatura":"ogro"` (a ligada, o que a pessoa digitou). O
// `encoding/json` do Go casa campo SEM diferenciar caixa quando não há
// correspondência exata, então a segunda chave também caía no campo — e a
// última a chegar vencia. A busca funcionava por ORDEM DAS CHAVES: invertida a
// ordem, o mesmo código lê `""` e a caixa de busca deixa de filtrar, sem erro
// em lugar nenhum.
//
// Nenhum teste podia pegar isso onde ele morava: a minusculação acontece no
// PARSER DO NAVEGADOR, então o HTML servido ainda tem a caixa certa e todo teste
// de Go vê o nome que o autor escreveu. O navegador foi a única testemunha, e
// uma testemunha que só é chamada quando alguém já suspeita não é guarda. Por
// isso a garantia desceu para a camada mais barata que a segura: o TEXTO do
// atributo, que é onde o autor erra.
var attributeKeyWithCase = regexp.MustCompile(`\bdata-[a-z][a-z-]*:([a-zA-Z0-9_-]+)`)

func TestNoDatastarAttributeKeyCarriesUppercase(t *testing.T) {
	root := filepath.Join("..", "..")
	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached",
		"*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}

	filesRead, keysRead := 0, 0
	var offenders []string
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		// O `_templ.go` fica de FORA: ele repete o que o `.templ` já disse, e
		// contá-lo faria toda falha aparecer duas vezes com dois endereços, um
		// deles gerado — o que manda quem for consertar para o arquivo errado.
		if relative == "" || strings.HasSuffix(relative, "_templ.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		for n, line := range strings.Split(string(body), "\n") {
			// A LINHA DE COMENTÁRIO fica de fora, e a exclusão é o que torna
			// este guarda possível: os treze comentários que ENSINAM a regra
			// citam a forma errada de propósito, e um guarda que os proibisse
			// tiraria do repositório justamente a explicação que impede o
			// defeito. É a mesma fresta que a lista de lápides do
			// `TestNoCitationNamesAMissingSymbol` abre para o nome hipotético.
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, m := range attributeKeyWithCase.FindAllStringSubmatch(line, -1) {
				keysRead++
				if strings.ToLower(m[1]) == m[1] {
					continue
				}
				offenders = append(offenders, relative+":"+strconv.Itoa(n+1)+" — "+m[0])
			}
		}
	}

	// O DENOMINADOR: "nenhuma chave com caixa alta" e "nenhuma chave lida" são a
	// mesma cor no terminal, e este guarda depende de um regex que casa uma
	// forma estreita — é justamente o tipo que passa verde por não casar nada.
	if filesRead < 300 || keysRead < 100 {
		t.Fatalf("a varredura leu %d arquivos e %d chaves de atributo — a raiz é o primeiro suspeito",
			filesRead, keysRead)
	}

	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Errorf("chave de atributo do Datastar com CAIXA ALTA — %d de %d:\n  %s\n"+
			"O analisador de HTML a minuscula e o Datastar liga um sinal NOVO: o declarado fica "+
			"intocado, o fio leva os dois e o servidor lê o errado — sem erro em lugar nenhum. "+
			"Escreva a chave toda minúscula (`data-bind:novonome`). Caixa alta só sobrevive "+
			"dentro de EXPRESSÃO (`$fichaAberta`), onde o valor do atributo preserva a caixa.",
			len(offenders), keysRead, strings.Join(offenders, "\n  "))
	}
	t.Logf("chaves de atributo do Datastar: %d, %d com caixa alta, de %d arquivos", keysRead, len(offenders), filesRead)
}
