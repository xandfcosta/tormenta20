package creature

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CRIATURA NÃO IMPORTA NADA DO PROJETO (ALE-278).
//
// Ela é o irmão mais estrito dos `fronteira_test.go` do `aovivo`, do `tabuleiro`,
// da `plataforma` e do `events`: a lista de permitidos aqui é VAZIA, e é o que a
// medição que motivou a extração já dizia — o arquivo importava `fmt` e
// `strings`, e nada mais.
//
// # Por que a lista vazia importa
//
// Este pacote saiu do `api` para que as CENAS pudessem sair depois. Cada cena
// que se mudar vai importá-lo, e no dia em que ele alcançar o catálogo ou o
// banco, todas elas alcançam junto — de graça, e com o guarda de fronteira de
// cada uma continuando verde, porque cada guarda só olha os imports DELE.
//
// É a mesma armadilha que o `events` documenta, e a resposta é a mesma: enquanto
// o pacote for FOLHA, depender dele não cria fronteira errada nenhuma.
func TestTheCreatureImportsNothing(t *testing.T) {
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	conjunto := token.NewFileSet()
	visitados := 0
	for _, entrada := range arquivos {
		nome := entrada.Name()
		if !strings.HasSuffix(nome, ".go") {
			continue
		}
		visitados++
		arquivo, err := parser.ParseFile(conjunto, nome, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for _, imp := range arquivo.Imports {
			caminho := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(caminho, "t20engine/") {
				continue
			}
			t.Errorf("%s importa %q — a criatura é FOLHA.\n"+
				"Ela existe para as cenas poderem importá-la sem herdar nada;\n"+
				"com um import daqui, TODAS herdam %q e o guarda de cada uma continua verde.",
				nome, caminho, caminho)
		}
	}

	// Sem isto, apagar o pacote deixaria o guarda VERDE — ausência lida como
	// aprovação, que é a família que o `CLAUDE.md` da raiz descreve.
	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
