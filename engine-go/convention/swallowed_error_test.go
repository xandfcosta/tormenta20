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

// O CASO DE USO NÃO ENGOLE ERRO COM `_ =`.
//
// O `app/` é onde um gesto decide o que acontece, e ali um erro descartado não
// é uma linha de log a menos — é o gesto acontecendo pela metade sem ninguém
// saber. Medido na ALE-372: o fim de uma habilidade sustentada não paga
// escrevia na ficha com
//
//	_ = st.turnEffects.EndSustained(context.Background(), …)
//
// e esse `_ =` estava escondendo um `database is locked (5) (SQLITE_BUSY)`. O
// efeito continuava ligado na ficha, a tela dizia que a sustentada tinha
// acabado, e **nada aparecia em lugar nenhum**. O caso que reprovava dizia "a
// sustentada não paga continua na ficha" — o SINTOMA, três camadas longe da
// causa.
//
// A razão que sustentava esses `_ =` era boa e foi derrubada pelo dono na
// ALE-373: "uma mesa travada no turno de alguém é pior que uma manutenção não
// cobrada". O preço dela era pior — a vez passava com o efeito ligado, e a mesa
// seguia jogando sobre uma incerteza. Hoje a escrita que falha RECUSA o clique.
//
// # O terreno é o `app/`, e só ele
//
// No `serve/` o `_ = sse.PatchElements(…)` é outra conversa: ali o erro é o
// cliente ter ido embora no meio da resposta, e não há gesto a desfazer nem a
// quem contar. Trazer o `serve/` para este guarda misturaria duas perguntas e
// obrigaria a uma lista de exceções do tamanho do diretório.
//
// # A ÚNICA forma dispensada, e por quê
//
// `defer func() { _ = tx.Rollback() }()` é o idioma que garante que toda saída
// por erro — inclusive um `panic` — feche a transação. Ele é NO-OP depois do
// commit, e o erro dele não tem leitor possível: se o commit passou não há o
// que desfazer, e se não passou o erro que importa já está subindo.
//
// # A forma que o `_ =` NÃO mostra, e que foi a que escapou
//
// `p.mirrorToTracker(ctx, …)` numa linha sozinha descarta o erro **sem escrever
// nada**: em Go, chamar e ignorar o retorno é sintaxe válida e silenciosa. Uma
// varredura por `_ =` passa por cima — e foi assim que o espelho dos vitais no
// descanso ficou sem leitor de erro enquanto os três `_ =` irmãos eram
// consertados (ALE-372).
//
// Para saber se uma chamada SOLTA devolve erro sem carregar o verificador de
// tipos inteiro, o guarda colhe do próprio `app/` os nomes declarados que
// devolvem `error` — o que inclui os métodos das PORTAS, porque as interfaces
// que os declaram moram aqui.
//
// **A colisão de nome é a armadilha, e ela é sistemática e não eventual.** O
// `app/boards.Store` tem um `RemoveToken` que devolve erro, e o `domain/board`
// tem um `RemoveToken` PURO que não devolve nada — e o store chama o puro. Pela
// primeira versão, cinco chamadas de regra pura apareceram como erro engolido:
// `RemoveToken`, `ClearSquare`, `RemoveMarker`, `PaintTerrain`, `StartScene`.
// **Essa é a forma normal deste repositório**, em que o caso de uso embrulha a
// regra e herda o nome dela.
//
// O que separa as duas é o RECEPTOR: `board.RemoveToken(…)` é qualificado por
// PACOTE, `bs.RemoveToken(…)` não. O guarda lê os imports de cada arquivo e
// pula o que é chamada de pacote — o que sobra é método em receptor e função
// do próprio pacote, que é exatamente o gesto.
func TestNoUseCaseSwallowsAnError(t *testing.T) {
	root, err := filepath.Abs("../app")
	if err != nil {
		t.Fatalf("achar o `app/`: %v", err)
	}
	set := token.NewFileSet()
	returnsError := namesThatReturnAnError(t, set, root)
	measured, swallows := 0, 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		measured++
		rel, _ := filepath.Rel(filepath.Join(root, ".."), path)
		packages := importedPackagesOf(file)
		ast.Inspect(file, func(n ast.Node) bool {
			// A CHAMADA SOLTA: o erro cai no chão sem uma linha dizendo isso.
			if loose, ok := n.(*ast.ExprStmt); ok {
				call, ok := loose.X.(*ast.CallExpr)
				if ok && returnsError[calledName(call)] && !isPackageCall(call, packages) {
					swallows++
					t.Errorf("%s:%d chama `%s`, que devolve erro, e não olha o retorno.\n"+
						"Esta é a forma que um `grep` por `_ =` não acha: em Go, ignorar o\n"+
						"retorno é sintaxe válida e silenciosa. Trate o erro, ou deixe-o SUBIR\n"+
						"para quem clicou (ALE-372).",
						rel, set.Position(loose.Pos()).Line, calledName(call))
				}
				return true
			}
			assign, ok := n.(*ast.AssignStmt)
			if !ok || !everyTargetIsBlank(assign) {
				return true
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok || isTheRollbackIdiom(call) {
				return true
			}
			swallows++
			t.Errorf("%s:%d joga fora o resultado de `%s` com `_ =`.\n"+
				"No `app/` um erro descartado é o gesto acontecendo pela metade: trate-o,\n"+
				"ou deixe-o SUBIR para quem clicou. Se o gesto tem mesmo de seguir apesar\n"+
				"da falha, isso é uma decisão — escreva-a no comentário e diga o que foi\n"+
				"perdido, com o NOME da coisa que não foi gravada (ALE-372).",
				rel, set.Position(assign.Pos()).Line, calledName(call))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar o `app/`: %v", err)
	}

	// O DENOMINADOR: uma lista de reprovados vazia e um diretório não lido se
	// parecem no terminal.
	if measured < 30 {
		t.Fatalf("o guarda leu só %d arquivos em `app/` — ele está medindo o terreno errado", measured)
	}
	if swallows > 0 {
		t.Logf("%d descartes em %d arquivos de `app/`", swallows, measured)
	}
}

// namesThatReturnAnError colhe, do próprio `app/`, os nomes de função e de
// método que devolvem `error` — inclusive os declarados em INTERFACE, que é
// como as portas dos casos de uso aparecem.
//
// É o que substitui o verificador de tipos para a pergunta "esta chamada solta
// devolve erro?". Ela responde só sobre o que o `app/` declara, e é justamente
// aí que mora o gesto.
func namesThatReturnAnError(t *testing.T, set *token.FileSet, root string) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.FuncDecl:
				if hasAnErrorResult(node.Type) {
					names[node.Name.Name] = true
				}
			case *ast.InterfaceType:
				for _, method := range node.Methods.List {
					signature, ok := method.Type.(*ast.FuncType)
					if !ok || !hasAnErrorResult(signature) {
						continue
					}
					for _, name := range method.Names {
						names[name.Name] = true
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("colher os nomes que devolvem erro: %v", err)
	}
	return names
}

// importedPackagesOf colhe os identificadores de pacote que ESTE arquivo
// importa — é por eles que se reconhece a chamada de regra pura.
func importedPackagesOf(file *ast.File) map[string]bool {
	packages := map[string]bool{}
	for _, imported := range file.Imports {
		if imported.Name != nil {
			packages[imported.Name.Name] = true
			continue
		}
		path := strings.Trim(imported.Path.Value, `"`)
		packages[path[strings.LastIndex(path, "/")+1:]] = true
	}
	return packages
}

// isPackageCall diz se a chamada é `pacote.Func(…)` e não método num receptor.
func isPackageCall(call *ast.CallExpr, packages map[string]bool) bool {
	target, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	receiver, ok := target.X.(*ast.Ident)
	return ok && packages[receiver.Name]
}

// hasAnErrorResult diz se a assinatura tem `error` entre os retornos.
func hasAnErrorResult(signature *ast.FuncType) bool {
	if signature.Results == nil {
		return false
	}
	for _, result := range signature.Results.List {
		if name, ok := result.Type.(*ast.Ident); ok && name.Name == "error" {
			return true
		}
	}
	return false
}

// everyTargetIsBlank diz se a atribuição joga TUDO fora — `_ =` e `_, _ =`.
//
// Tudo e não "algum": `hp, _ := …` descarta um valor que o chamador não quer, e
// isso é diferente de descartar o resultado inteiro de uma chamada.
func everyTargetIsBlank(assign *ast.AssignStmt) bool {
	if len(assign.Rhs) != 1 {
		return false
	}
	for _, target := range assign.Lhs {
		name, ok := target.(*ast.Ident)
		if !ok || name.Name != "_" {
			return false
		}
	}
	return len(assign.Lhs) > 0
}

// isTheRollbackIdiom reconhece o `_ = tx.Rollback()` do `defer`.
func isTheRollbackIdiom(call *ast.CallExpr) bool {
	target, ok := call.Fun.(*ast.SelectorExpr)
	return ok && target.Sel.Name == "Rollback"
}

// calledName nomeia a chamada para a mensagem de falha dizer O QUE foi
// descartado — "joga fora o resultado" sem sujeito manda procurar na linha.
func calledName(call *ast.CallExpr) string {
	switch target := call.Fun.(type) {
	case *ast.SelectorExpr:
		return target.Sel.Name
	case *ast.Ident:
		return target.Name
	}
	return "a chamada"
}
