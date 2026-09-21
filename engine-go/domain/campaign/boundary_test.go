package campaign

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O PACOTE É FOLHA, e a lista é de UM.
//
// Irmão gêmeo do `account/boundary_test.go`, e a semelhança não é estilo: os
// dois pacotes nasceram do mesmo defeito. Uma regra de PRODUTO — o que é um
// e-mail aceitável, o que é um nome de campanha aceitável — dentro do `api`, é
// lida por uma tela E por uma rota JSON, e as duas divergem na FRASE.
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
	"t20engine/infra/wire": true,
	// O `engine` é a exceção que a lista curta suporta: a pergunta "esta regra
	// opcional existe?" é de domínio para domínio, o `engine` é folha, e não há
	// ciclo nem HTTP no caminho. A alternativa era o chamador passar a lista de
	// regras conhecidas — o que faria cada tela carregar um dado que ela não usa
	// para nada além de repassar.
	"t20engine/domain/engine": true,
}

// recusadosDaPadrao são pacotes da BIBLIOTECA PADRÃO que este pacote não pode
// tocar, e a lista existe por um vermelho.
//
// Um guarda que só olhasse `t20engine/*` deixa passar `var _ = sql.NullString{}`
// — sabotado assim, o build passa E o guarda passa, com a docstring afirmando
// uma garantia que não existe. **Comentário não é correção.**
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
			if reason, refused := recusadosDaPadrao[path]; refused {
				t.Errorf("%s importa %q — %s.", name, path, reason)
				continue
			}
			if !strings.HasPrefix(path, "t20engine/") || permitidos[path] {
				continue
			}
			t.Errorf("%s importa %q.\n"+
				"Este pacote é FOLHA de propósito: ele é lido pela cena da porta e pela API\n"+
				"JSON, e um import a mais aqui é o que faz o próximo lado escrever uma cópia\n"+
				"da regra em vez de importá-la. Já aconteceu uma vez.",
				name, path)
		}
	}

	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
