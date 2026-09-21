package character

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// OS GESTOS DE VÁRIAS ESCRITAS PASSAM POR UMA TRANSAÇÃO.
//
// # Por que este guarda lê a FONTE
//
// O que ele protege só acontece quando uma escrita do MEIO falha, e não há como
// fazer a terceira falhar sem instrumentar a produção para o teste. Medido à
// mão, com o terceiro passo do `EnterStance` estourando de propósito: com a
// transação, o PM fica nos 20 semeados e a tabela de posturas fica VAZIA; sem
// ela, o jogador paga 4 PM e a postura fica em pé sem os condicionais dela —
// que é o estado que a tabela `character_stances` existe para tornar
// irreversível, porque sair não devolve PM (ALE-351).
//
// O guarda não prova o rollback. Ele prova que o CONTORNO não sumiu, que é o
// jeito como esta família de defeito volta: alguém simplifica a função, tira o
// invólucro, e nada fica vermelho — os testes que existem exercitam o caminho
// FELIZ, onde transação nenhuma faz diferença.
//
// # E ele não pede transação para todo mundo
//
// A lista é dos gestos que escrevem MAIS DE UMA VEZ. O `UsePower` sem limite
// cobrado escreve uma vez só e sai pelo caminho curto de propósito — uma
// transação que não precisa existir é um bloqueio que ninguém pediu, e o
// comentário dele diz isso.
func TestEveryMultiWriteStanceGestureIsWrappedInATransaction(t *testing.T) {
	require := map[string]bool{"EnterStance": true, "EndStance": true, "UsePower": true}

	set := token.NewFileSet()
	file, err := parser.ParseFile(set, "stance.go", nil, 0)
	if err != nil {
		t.Fatalf("ler o `stance.go`: %v", err)
	}
	seen := map[string]bool{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || !require[fn.Name.Name] {
			continue
		}
		seen[fn.Name.Name] = true
		found := false
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if target, ok := call.Fun.(*ast.SelectorExpr); ok && target.Sel.Name == "inTx" {
				found = true
			}
			return true
		})
		if !found {
			t.Errorf("`%s` escreve mais de uma vez e NÃO passa por `inTx`.\n"+
				"Sem o contorno, uma escrita do meio que falhe deixa o gesto pela metade —\n"+
				"e o caminho feliz, que é o que os testes exercitam, não acusa nada.",
				fn.Name.Name)
		}
	}

	// O DENOMINADOR: um nome que mudou faria o laço não visitar ninguém e o
	// guarda passar dizendo que está tudo certo.
	for name := range require {
		if !seen[name] {
			t.Fatalf("o guarda não achou `%s` no `stance.go` — ele mediria o vazio.\n"+
				"Se o gesto mudou de nome ou de arquivo, a lista acompanha.", name)
		}
	}
	// E o arquivo é o que se pensa que é.
	if source := file.Name.Name; !strings.HasSuffix(source, "character") {
		t.Fatalf("o guarda leu o pacote %q", source)
	}
}
