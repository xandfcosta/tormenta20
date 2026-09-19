package door

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO.
//
// Este é o guarda que faz a PORTA valer alguma coisa. A porta declara em
// `deps.go` o que precisa, e o `api` cumpre.
//
// Importar o `api` daqui o COMPILADOR já recusa — é ciclo, porque o `api`
// importa esta cena de volta para montar rota. O valor deste guarda é o RESTO, e
// o que ele guarda hoje é o BCRYPT: hashear senha não é trabalho de quem desenha
// o formulário, e o caminho inteiro da redefinição acontece do outro lado da
// chamada (`accounts.Resets.Apply`).
//
// A lista é curta porque a porta é uma tela de formulário: ela não lê catálogo,
// não computa ficha e não conhece o livro.

var permitidos = map[string]bool{
	// O `app/accounts` NÃO é concessão, é a porta encolhendo de nove métodos
	// para dois (ALE-349): ele está ABAIXO desta cena, então não há ciclo para
	// desviar e não há interface a declarar. Os casos de uso chegam por
	// PARÂMETRO do `New`, como nas campanhas e na Mesa — e com as recusas
	// exportadas de lá, o vocabulário que esta cena declarava só para atravessar
	// a fronteira deixou de existir.
	"t20engine/app/accounts":     true,
	"t20engine/domain/account":   true, // o que uma conta aceita: e-mail, senha, a forma do pedido
	"t20engine/serve/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/infra/db/sqlcgen": true, // as linhas do banco, que atravessam a porta
	"t20engine/infra/wire":       true,
}

func TestTheDoorDoesNotImportItsHost(t *testing.T) {
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
