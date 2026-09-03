package campaign

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O PACOTE É FOLHA, e a lista é de UM (ALE-278).
//
// Irmão gêmeo do `account/fronteira_test.go`, e a semelhança não é estilo: os
// dois pacotes nasceram do mesmo defeito. Uma regra de PRODUTO — o que é um
// e-mail aceitável, o que é um nome de campanha aceitável — morava dentro do
// `api`, era lida por uma tela E por uma rota JSON, e as duas divergiram na
// FRASE, com a rota respondendo em inglês.
//
// A lista curta é a razão de o pacote existir. No dia em que estas funções
// alcançarem banco, catálogo ou HTTP, o próximo lado que precisar da regra não
// vai poder importá-la — e vai escrever uma cópia. A cópia seguinte estará
// errada de outro jeito, e vai compilar.
//
// **O `database/sql` é a tentação NOMEADA aqui**, e ela não é hipotética: a
// versão anterior do `Description` devolvia `sql.NullString`, então a regra de
// produto carregava o tipo do banco. Quem grava é que traduz vazio para NULL.
var permitidos = map[string]bool{
	"t20engine/plataforma": true, // o mapa de erro por campo, e nada mais
}

// recusadosDaPadrao são pacotes da BIBLIOTECA PADRÃO que este pacote não pode
// tocar, e a lista existe por um vermelho.
//
// A prosa acima chamava o `database/sql` de "tentação nomeada", e o guarda não
// o via: ele só olhava `t20engine/*`. Sabotei com `var _ = sql.NullString{}`, o
// build passou E o guarda passou — a docstring afirmava uma garantia que não
// existia, que é exatamente o que o CLAUDE.md quer dizer com **comentário não é
// correção**.
//
// Só `database/sql` por enquanto, e não uma lista de tudo que é infraestrutura:
// esta é a tentação MEDIDA — a versão anterior do `Description` devolvia
// `sql.NullString` de verdade. Lista de perigos imaginados envelhece; lista de
// defeito acontecido, não.
var recusadosDaPadrao = map[string]string{
	"database/sql": "regra de PRODUTO não carrega o tipo do banco: quem grava é que " +
		"traduz vazio para NULL. A versão anterior do `Description` devolvia " +
		"`sql.NullString`, e a extração desfez isso",
}

func TestTheCampaignRulesStayALeaf(t *testing.T) {
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
			if porque, recusado := recusadosDaPadrao[caminho]; recusado {
				t.Errorf("%s importa %q — %s.", nome, caminho, porque)
				continue
			}
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
