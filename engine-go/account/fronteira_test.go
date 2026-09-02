package account

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O PACOTE É FOLHA, e aqui isso importa mais do que na média (ALE-278).
//
// Irmão dos `fronteira_test.go` do `search`, do `creature`, do `sheet`, do
// `events`, do `tabuleiro` e da `plataforma`. A lista é de UM: só o
// `plataforma`, de onde vem o mapa de erro por campo.
//
// A razão de ser tão curta é a razão de o pacote existir. Estas funções são
// lidas pela cena da porta E pela API JSON; no dia em que uma delas alcançar
// banco, catálogo ou HTTP, o próximo lado que precisar da regra não vai poder
// importá-la — e vai escrever uma cópia. **Aqui a cópia já existiu**: o `api`
// carregava as mesmas três validações duas vezes, e as duas famílias divergiram
// no texto da recusa, com a rota JSON respondendo em inglês a uma tela que
// responde em português.
//
// É a mesma lição que o `search` documenta com o `Fold`, e a única diferença é
// que lá a cópia estava errada na CONTA e aqui na FRASE. As duas compilam.
var permitidos = map[string]bool{
	"t20engine/plataforma": true, // o mapa de erro por campo, e nada mais
}

func TestTheAccountRulesStayALeaf(t *testing.T) {
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
				"Este pacote é FOLHA de propósito: ele é lido pela cena da porta e pela API\n"+
				"JSON, e um import a mais aqui é o que faz o próximo lado escrever uma cópia\n"+
				"da regra em vez de importá-la. Já aconteceu uma vez.",
				nome, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
