package hub

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// Este é o guarda que faz a PORTA valer alguma coisa. O hub declara em
// `deps.go` o que precisa, e o `api` cumpre.
//
// # O que ele pega
//
// Importar o `api` daqui o COMPILADOR já recusa — ele importa esta cena de volta
// para montar rota. O valor do guarda é o resto: na forja, a primeira execução
// pegou um import de `t20engine/domain/catalog` DIRETO, contornando a camada tipada.
// Nenhum ciclo, nenhum erro — só a divisão vazando por baixo.
//
// # O que a lista permite, e por quê
//
// O hub é a cena mais MAGRA: ele não lê catálogo nem computa ficha, então nem o
// `book` nem o `sheet` entram. A lista curta é o desenho, e não descuido — se
// um dia ela crescer, a pergunta é se a cena está fazendo trabalho de outra.
var permitidos = map[string]bool{
	// O `app/accounts` chega por PARÂMETRO do `New`, e não pela porta: cunhar o
	// convite de conta é caso de uso, e o `app/` está abaixo desta cena
	// (ALE-349).
	"t20engine/app/accounts":     true,
	"t20engine/serve/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/infra/db/sqlcgen": true, // as linhas do banco
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
