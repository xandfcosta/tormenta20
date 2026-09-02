package door

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A CENA NÃO IMPORTA O HOSPEDEIRO (ALE-278).
//
// Este é o guarda que faz a PORTA valer alguma coisa. A porta declara em
// `deps.go` o que precisa, e o `api` cumpre.
//
// Importar o `api` daqui o COMPILADOR já recusa — é ciclo, porque o `api`
// importa esta cena de volta para montar rota. O valor deste guarda é o resto,
// e nesta cena o resto era grande: ela alcançava o `bcrypt` para gerar hash de
// senha, o `db` para reconhecer violação de unicidade, e dois sentinelas de erro
// do `api` para saber se um convite tinha sido recusado ou gasto.
//
// As três saíram, e nenhuma por regra de estilo:
//
//   - o BCRYPT porque o custo criptográfico é decisão de segurança do servidor,
//     e a cena estaria carregando a constante dele para fazer trabalho que não é
//     dela (`ResetPassword` faz o caminho inteiro do outro lado);
//   - o `db` e os SENTINELAS porque classificar o erro é do hospedeiro; a cena
//     recebe um MOTIVO e escolhe a frase, que é a parte que é dela.
//
// A lista abaixo é o que sobrou, e ela é curta porque a porta é uma tela de
// formulário: ela não lê catálogo, não computa ficha e não conhece o livro.

var permitidos = map[string]bool{
	"t20engine/account":    true, // o que uma conta aceita: e-mail, senha, a forma do pedido
	"t20engine/web/ui":     true, // o kit de apresentação e a casca
	"t20engine/db/sqlcgen": true, // as linhas do banco, que atravessam a porta
	"t20engine/plataforma": true, // não é domínio nenhum
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
