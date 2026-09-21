package bookui

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O `bookui` NÃO IMPORTA O HOSPEDEIRO, e este guarda é o mais importante da
// série depois do `book`.
//
// A razão é aritmética, e é a mesma que o `book` documenta: quase toda cena que
// desenha um verbete vai importá-lo — o mestre já importa, o verbete importa, e
// a ficha vai importar quando sair. No dia em que ele alcançar o `api`, todas
// elas alcançam junto, de graça, com o guarda de fronteira de cada uma
// continuando VERDE, porque cada guarda só olha os imports dele.
//
// POR QUE ELE NÃO É O `web/ui`: o `trecho` que estes componentes desenham nasce
// de uma consulta ao CATÁLOGO, e levá-los para o kit faria o pacote de
// APRESENTAÇÃO importar catálogo — o contrário do que a divisão existe para
// conseguir. Este é o pacote do meio: sabe do livro e não sabe de HTTP.
var permitidos = map[string]bool{
	"t20engine/domain/book":      true, // o catálogo TIPADO, que é o que ele desenha
	"t20engine/serve/web/ui":     true, // o kit, para o botão e o ícone
	"t20engine/serve/web/routes": true, // os endereços do leitor e do verbete
}

func TestTheBookUIDoesNotImportItsHost(t *testing.T) {
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
			if !strings.HasPrefix(path, "t20engine/") || permitidos[path] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.",
				name, path, path)
		}
	}

	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
