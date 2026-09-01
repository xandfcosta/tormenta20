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
// pegou um import de `t20engine/catalog` DIRETO, contornando a camada tipada.
// Nenhum ciclo, nenhum erro — só a divisão vazando por baixo.
//
// # O que a lista permite, e por quê
//
// O hub é a cena mais MAGRA: ele não lê catálogo nem computa ficha, então nem o
// `book` nem o `sheet` entram. A lista curta é o desenho, e não descuido — se
// um dia ela crescer, a pergunta é se a cena está fazendo trabalho de outra.
var permitidos = map[string]bool{
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
