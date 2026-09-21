package convention

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// NINGUÉM INVENTA UM CONJUNTO DE CONDICIONAIS VAZIO.
//
// # O mecanismo, que é o que faz disto guarda
//
// Os condicionais são o OPT-IN do jogador: a Fúria em pé, o item esotérico
// vestido que ele declara estar usando. Quem computa a ficha recebe esse
// conjunto por parâmetro — e passar `map[string]bool{}` no lugar dele não
// quebra nada: computa um personagem PLAUSÍVEL, que não é o que está na tela.
//
// Medido na ALE-357, com o defeito na mesma ficha: o crachá do topo dizia Defesa
// 12 e a aba Combate, 17. A aba passa os condicionais de verdade; o crachá vinha
// por um caminho que passava o vazio. Nos oráculos, a Lenda de nível 20 erra por
// DEZ pontos, e o `kharvos-o-guardiao-rubro` via a ficha oferecer um limite de
// PM que o servidor recusava.
//
// # Por que a lista de permitidos é um PACOTE e não uma lista de arquivos
//
// O `domain/engine` é o DONO da conta, e "a ficha base, sem condicional nenhum"
// é um resultado legítimo dele — o oráculo guarda os dois lado a lado (`sheet` e
// `sheetWithConditionals`), e os testes de paridade prendem cada um. Fora dali,
// quem computa está desenhando a ficha de ALGUÉM, e essa pessoa tem condicionais.
func TestNoCallerInventsAnEmptyConditionalSet(t *testing.T) {
	const ownerFile = "domain/engine"

	// As funções que RECEBEM o conjunto. Uma função nova que o receba precisa
	// entrar aqui — é lista de quem se VIGIA, e o custo de esquecer é o guarda
	// ficar verde sobre o caminho novo.
	receiveConditionals := map[string]bool{
		"ComputeSheet": true, "ComputeWeaponCards": true,
		"SpellPmCostFor": true, "SpellPmLimitFor": true,
		"ApplyActiveConditionals": true,
	}

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	set := token.NewFileSet()
	measured, callsSeen := 0, 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_templ.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if strings.HasPrefix(rel, ownerFile) || strings.HasSuffix(rel, "conditional_set_test.go") {
			return nil
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		measured++
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			target, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !receiveConditionals[target.Sel.Name] {
				return true
			}
			callsSeen++
			for _, arg := range call.Args {
				if !isEmptyStringBoolMap(arg) {
					continue
				}
				t.Errorf("%s:%d passa um conjunto de condicionais VAZIO para %s.\n"+
					"Os condicionais são o opt-in do jogador, e o agregado os carrega em\n"+
					"`dto.Conditionals` — passe `sheet.ToStringSet(dto.Conditionals)`. Inventar\n"+
					"o vazio computa um personagem plausível que não é o que está na tela, e\n"+
					"não deixa erro nenhum para trás.",
					rel, set.Position(call.Pos()).Line, target.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR, nas duas pontas: uma varredura que não abriu arquivo e um
	// seletor que deixou de casar com as chamadas se parecem com "nada reprovou".
	if measured < 200 {
		t.Fatalf("o guarda leu só %d arquivos — ele está medindo a árvore errada", measured)
	}
	if callsSeen < 3 {
		t.Fatalf("o guarda viu só %d chamadas que recebem condicionais fora do `%s` — "+
			"os nomes da lista mudaram e ele parou de procurar", callsSeen, ownerFile)
	}
}

// isEmptyStringBoolMap reconhece `map[string]bool{}` escrito no argumento.
//
// É a FORMA e não o valor: uma variável que por acaso esteja vazia em tempo de
// execução é outra coisa — pode ser um personagem que de fato não tem condicional
// ligado, que é legítimo. O que este guarda pega é o literal, que é sempre uma
// decisão de quem escreveu.
func isEmptyStringBoolMap(arg ast.Expr) bool {
	lit, ok := arg.(*ast.CompositeLit)
	if !ok || len(lit.Elts) > 0 {
		return false
	}
	kind, ok := lit.Type.(*ast.MapType)
	if !ok {
		return false
	}
	key, okKey := kind.Key.(*ast.Ident)
	value, okValue := kind.Value.(*ast.Ident)
	return okKey && okValue && key.Name == "string" && value.Name == "bool"
}
