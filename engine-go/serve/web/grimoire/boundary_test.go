package grimoire

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// O grimório declara em `deps.go` o que precisa, e o `*api.Server` cumpre.
//
// # O que ele pega
//
// Importar o `api` daqui o COMPILADOR já recusa — ele importa esta cena de volta
// para montar rota. O valor do guarda é o resto: na forja, a primeira execução
// pegou um import de `t20engine/domain/catalog` DIRETO, contornando a camada tipada.
// Nenhum ciclo, nenhum erro — só a divisão vazando por baixo.
//
// # A lista mais curta do projeto, e isso é o desenho
//
// Só o kit. Uma folha de especificação desenha PEÇAS: ela não lê banco, não
// computa regra e não sabe o que é um personagem. No dia em que esta lista
// crescer, a pergunta não é qual import permitir — é o que está sendo desenhado
// aqui que não é peça.
var permitidos = map[string]bool{
	"t20engine/serve/web/ui": true,
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
			if !strings.HasPrefix(path, "t20engine/") || permitidos[path] {
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
