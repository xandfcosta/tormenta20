package routes

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// ESTE PACOTE NÃO IMPORTA NADA (ALE-278).
//
// Ele é o mais folha de todos: endereços e as funções que os montam. Nada do
// PROJETO entra — a biblioteca padrão sim, porque o `net/url` escapa a query.
//
// Toda cena que sair daqui para frente vai importá-lo para linkar para as
// vizinhas. No dia em que ele alcançar catálogo, banco ou HTTP, todas as cenas
// alcançam junto, de graça, com o guarda de fronteira de cada uma continuando
// VERDE — porque cada guarda só olha os imports dele. É a mesma aritmética que o
// `book` e o `events` já documentam, e aqui ela é pior, porque um pacote de
// constantes não tem nenhuma razão legítima para crescer.
func TestTheAddressesImportNothing(t *testing.T) {
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}
	conjunto := token.NewFileSet()
	visitados := 0
	for _, entrada := range arquivos {
		nome := entrada.Name()
		if !strings.HasSuffix(nome, ".go") || strings.HasSuffix(nome, "_test.go") {
			continue
		}
		visitados++
		arquivo, err := parser.ParseFile(conjunto, nome, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for _, imp := range arquivo.Imports {
			caminho := strings.Trim(imp.Path.Value, `"`)
			// A biblioteca PADRÃO passa: o `net/url` escapa o que vai na query, e
			// recusá-lo empurraria o escape para cada chamador — que é como um
			// endereço sai sem escapar em UM lugar e ninguém vê.
			if !strings.HasPrefix(caminho, "t20engine/") {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Este pacote não pode alcançar NADA do projeto. Toda cena o importa: um\n"+
				"alcance aqui é um alcance concedido a todas elas de uma vez, e o guarda de\n"+
				"fronteira de cada uma continua verde, porque só olha os imports dela.",
				nome, caminho)
		}
	}
	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
