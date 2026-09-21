package forge

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO.
//
// Este é o guarda que faz a PORTA valer alguma coisa. A forja declara em
// `deps.go` o que precisa, e o `api` cumpre.
//
// # O que ele pega, medido e não suposto
//
// Importar o `api` daqui o COMPILADOR já recusa, porque o `api` importa esta
// cena de volta para montar rota — sabotado de propósito, a mensagem é "import
// cycle not allowed" e não este guarda. O valor dele é o resto, e ele o provou
// na primeira execução: `items.go` importava `t20engine/domain/catalog` DIRETO,
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
	// O `app/character` é o NASCIMENTO, e ele está ABAIXO desta cena: não há
	// ciclo para desviar, então não há interface. Três entradas da porta saíram
	// com ele (ALE-347), e a porta ficou com as quatro do núcleo.
	"t20engine/app/character":    true,
	"t20engine/domain/sheet":     true, // a forma e a construção da ficha
	"t20engine/domain/book":      true, // o catálogo tipado: raça, classe, item
	"t20engine/domain/engine":    true, // as regras do livro
	"t20engine/serve/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/infra/db/sqlcgen": true, // as linhas do banco
	"t20engine/infra/db/dbvalue": true,
	"t20engine/infra/wire":       true,
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
				"Acrescentar o import à lista transforma a porta em enfeite — e se %q for\n"+
				"o `api`, é ciclo, porque ele importa esta cena para montar rota.",
				name, path, path)
		}
	}

	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
