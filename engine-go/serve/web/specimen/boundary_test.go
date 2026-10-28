package specimen

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FOLHA VIVA NÃO IMPORTA O HOSPEDEIRO, nem nada além das peças (ALE-479).
//
// Ela é a irmã da folha do grimório, e a lista é curta pelo mesmo motivo: uma
// folha de peças desenha PEÇAS. O `ds` é o kit que ela mostra, o `ui` é a
// casca, e o `routes` é o endereço do seletor de modo, que o hospedeiro atende.
// No dia em que a lista crescer, a pergunta não é qual import permitir — é o
// que está sendo desenhado aqui que não é peça.
var allowed = map[string]bool{
	"t20engine/serve/web/ds":     true,
	"t20engine/serve/web/ui":     true,
	"t20engine/serve/web/routes": true,
}

func TestTheSceneDoesNotImportItsHost(t *testing.T) {
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
			if !strings.HasPrefix(path, "t20engine/") || allowed[path] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite.",
				name, path)
		}
	}

	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
