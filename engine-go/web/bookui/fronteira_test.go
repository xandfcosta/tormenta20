package bookui

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O `bookui` NÃO IMPORTA O HOSPEDEIRO, e este guarda é o mais importante da
// série depois do `book` (ALE-278).
//
// A razão é aritmética, e é a mesma que o `book` documenta: quase toda cena que
// desenha um verbete vai importá-lo — o mestre já importa, o verbete importa, e
// a ficha vai importar quando sair. No dia em que ele alcançar o `api`, todas
// elas alcançam junto, de graça, com o guarda de fronteira de cada uma
// continuando VERDE, porque cada guarda só olha os imports dele.
//
// # Por que este pacote existe, e por que ele não é o `web/ui`
//
// A fatia 4 deixou isto escrito e adiado: o livro e os elos ficaram no `api`
// porque o `trecho` que eles desenham nasce de uma consulta ao catálogo, e
// levá-los para o kit faria o pacote de APRESENTAÇÃO importar catálogo — o
// contrário do que a divisão existe para conseguir.
//
// A decisão adiada venceu na fatia do mestre: ele lê CINCO símbolos daqui, e não
// podia sair enquanto eles morassem no hospedeiro. A saída não era mover para o
// kit nem deixar ficar — era o pacote do meio, que sabe do livro e não sabe de
// HTTP.
var permitidos = map[string]bool{
	"t20engine/book":       true, // o catálogo TIPADO, que é o que ele desenha
	"t20engine/web/ui":     true, // o kit, para o botão e o ícone
	"t20engine/web/routes": true, // os endereços do leitor e do verbete
}

func TestTheBookUIDoesNotImportItsHost(t *testing.T) {
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
