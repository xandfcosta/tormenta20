package finder

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// Aqui o guarda é quase uma formalidade, e vale dizer por quê: esta é a primeira
// cena que saiu SEM PORTA. Ela não declara `Deps` porque não alcança nada do
// `api` — lê o livro embutido pelo `book`, pontua com o `search`, desenha com o
// `web/ui` e linka pelo `web/routes`.
//
// O que o guarda protege não é o hoje, é o amanhã. A tentação concreta é o
// catálogo: quem for acrescentar uma família de achados vai querer `t20engine/catalog`
// direto, contornando a camada tipada — que é exatamente o que ele pegou na
// forja, no primeiro dia (`items.go` importava o catálogo cru). Nenhum ciclo,
// nenhum erro, só a divisão vazando por baixo.
var permitidos = map[string]bool{
	"t20engine/book":       true, // o catálogo TIPADO, e não o `catalog` cru
	"t20engine/search":     true, // o casamento e a pontuação
	"t20engine/web/ui":     true, // o kit de apresentação
	"t20engine/web/routes": true, // os endereços que ela cita das vizinhas
}

func TestTheFinderDoesNotImportItsHost(t *testing.T) {
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
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.",
				nome, caminho, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
