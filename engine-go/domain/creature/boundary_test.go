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
// Ela é o irmão mais estrito dos `boundary_test.go` do `live`, do `tabuleiro`,
// da `platform` e do `events`: a lista de permitidos aqui é VAZIA, e é o que a
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
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	set := token.NewFileSet()
	visited := 0
	for _, entry := range files {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		visited++
		file, err := parser.ParseFile(set, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(path, "t20engine/") {
				continue
			}
			t.Errorf("%s importa %q — a criatura é FOLHA.\n"+
				"Ela existe para as cenas poderem importá-la sem herdar nada;\n"+
				"com um import daqui, TODAS herdam %q e o guarda de cada uma continua verde.",
				name, path, path)
		}
	}

	// Sem isto, apagar o pacote deixaria o guarda VERDE — ausência lida como
	// aprovação, que é a família que o `CLAUDE.md` da raiz descreve.
	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
