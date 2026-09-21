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

// A APRESENTAÇÃO NÃO ABRE TRANSAÇÃO.
//
// Uma transação é o contorno de um gesto: ou as campanhas mudam de dono e a
// conta some juntas, ou nada acontece. Quem desenha esse contorno está
// decidindo o que o gesto É, e isso é caso de uso — não do lado que sabe o que
// é um `http.ResponseWriter`.
//
// O sintoma de que ela estava no lugar errado não é abstrato. A porta assinava
// a sessão num método que recebia um `ResponseWriter` para poder escrever um
// 500 lá dentro; a administração apagava conta a partir de um `*http.Request`
// que ela só usava pelo `Context()`. Nos dois casos, o segundo transporte teria
// de chamar a própria rota por dentro ou copiar a regra.
//
// O `serve/` chegou a ZERO na ALE-349, e é por isso que este guarda nasce agora:
// a linha de base é vazia, então ele não tem dívida a administrar — ele falha no
// PRIMEIRO que voltar, com o nome do arquivo.
//
// # O que ele NÃO varre
//
// Os `_test.go`. Um teste do `serve/api` arranja estado, e abrir uma transação
// para semear é encanamento de bancada, não desenho de gesto.
func TestNoPresentationLayerOpensATransaction(t *testing.T) {
	// `Begin` entra junto com `BeginTx` porque o `database/sql` tem os dois, e
	// varrer só o nomeado deixaria a porta aberta pelo irmão sem contexto.
	openings := map[string]bool{"BeginTx": true, "Begin": true}

	root, err := filepath.Abs("../serve")
	if err != nil {
		t.Fatalf("achar o `serve/`: %v", err)
	}
	set := token.NewFileSet()
	measured := 0
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
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			target, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !openings[target.Sel.Name] {
				return true
			}
			t.Errorf("%s:%d abre uma transação (`%s`) na camada de APRESENTAÇÃO.\n"+
				"O contorno de um gesto é o caso de uso: mova o corpo para um tipo do\n"+
				"`app/`, que a cena recebe por PARÂMETRO do `New` — e devolva recusa\n"+
				"TIPADA (`app.ErrForbidden`), nunca um número de HTTP.",
				rel, set.Position(call.Pos()).Line, target.Sel.Name)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar o `serve/`: %v", err)
	}

	// O DENOMINADOR: uma lista de reprovados vazia e um diretório não lido se
	// parecem no terminal.
	if measured < 50 {
		t.Fatalf("o guarda leu só %d arquivos em `serve/` — ele está medindo o diretório errado", measured)
	}
}
