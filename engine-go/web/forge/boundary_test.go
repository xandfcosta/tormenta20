package forge

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// Este é o guarda que faz a PORTA valer alguma coisa. A forja declara em
// `deps.go` o que precisa, e o `api` cumpre.
//
// # O que ele pega, medido e não suposto
//
// Importar o `api` daqui o COMPILADOR já recusa, porque o `api` importa esta
// cena de volta para montar rota — sabotado de propósito, a mensagem é "import
// cycle not allowed" e não este guarda. O valor dele é o resto, e ele o provou
// na primeira execução: `items.go` importava `t20engine/catalog` DIRETO,
// contornando a camada tipada que existe justamente para isso. Nenhum ciclo,
// nenhum erro — só a divisão vazando por baixo.
//
// A regra que aquele achado deixou: o destino de uma função é a DEPENDÊNCIA
// dela. O índice de itens de origem lia catálogo, então era do livro, mesmo que
// só a forja o usasse.
//
// # O que a lista permite, e por quê
//
// As três camadas que saíram antes desta cena, mais o kit e a infraestrutura. É
// exatamente o conjunto que a extração tornou possível: sem `sheet`, `book` e
// `web/ui` fora do `api`, esta cena não teria conseguido sair.
var permitidos = map[string]bool{
	"t20engine/sheet":      true, // a forma e a construção da ficha
	"t20engine/book":       true, // o catálogo tipado: raça, classe, item
	"t20engine/engine":     true, // as regras do livro
	"t20engine/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/db/sqlcgen": true, // as linhas do banco
	"t20engine/plataforma": true, // não é domínio nenhum
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
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.",
				nome, caminho, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
