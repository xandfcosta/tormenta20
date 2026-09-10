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
// pegou um import de `t20engine/catalog` DIRETO, contornando a camada tipada.
// Nenhum ciclo, nenhum erro — só a divisão vazando por baixo.
//
// # A lista mais curta do projeto, e isso é o desenho
//
// Só o kit. Uma folha de especificação desenha PEÇAS: ela não lê banco, não
// computa regra e não sabe o que é um personagem. No dia em que esta lista
// crescer, a pergunta não é qual import permitir — é o que está sendo desenhado
// aqui que não é peça.
var permitidos = map[string]bool{
	"t20engine/web/ui": true,
}

func TestTheSceneDoesNotImportItsHost(t *testing.T) {
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
			if !strings.HasPrefix(caminho, "t20engine/") || permitidos[caminho] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Se a cena precisa de algo de lá, DECLARE na `Deps` e receba de quem monta.\n"+
				"Acrescentar o import à lista transforma a porta em enfeite.",
				nome, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
