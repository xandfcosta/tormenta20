package convention

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// O GESTO NÃO INVENTA O PRÓPRIO CONTEXTO: ele recebe o da requisição.
//
// Um `context.Background()` no meio de um gesto custa duas coisas, e a segunda
// é a que não se vê:
//
//   - CANCELAMENTO deixa de existir. O cliente fecha a aba e o servidor segue
//     lendo e gravando por ele.
//   - A REDE CONTRA TRANSAÇÃO ANINHADA fica desligada. O `session.Units.Do` é
//     reentrante quando o contexto CARREGA a unidade aberta (`session.WithUnit`),
//     e um contexto recém-inventado não carrega nada — então o `Do` de dentro
//     pede a segunda conexão do pool e trava contra a trava de escrita da
//     primeira, que espera este trabalho retornar. É o impasse da ALE-371:
//     `database is locked (5) (SQLITE_BUSY)` depois do `busy_timeout` inteiro.
//
// O TERRENO é `app/` e `serve/`, e não o repositório: em `cmd/` a raiz do
// contexto é legítima — é onde o processo nasce — e o `infra/` é chamado, nunca
// chamador.
//
// # A linha de base está VAZIA
//
// Ela nasceu com duas entradas — a gravação do TABULEIRO, que saía depois em
// goroutine — e a ALE-375 as quitou repetindo no tabuleiro o desenho que a
// ALE-373 deu à fila. O guarda cobrou a quitação: quando os dois sítios sumiram,
// ele reprovou pelas ENTRADAS ÓRFÃS, que é a metade que impede o arquivo de
// virar mentira sozinho.
//
// Com a lista vazia ele não tem dívida a administrar: falha no PRIMEIRO que
// voltar, com o nome do sítio.
func TestNoGestureInventsItsOwnContext(t *testing.T) {
	// VAZIA, e é para continuar. Não acrescente: o conserto é receber o
	// contexto por parâmetro, e não registrar mais uma dívida.
	baseline := map[string]bool{}

	// `TODO` entra junto com `Background`: os dois nascem sem valor nenhum, e
	// varrer só o nomeado deixaria a porta aberta pelo irmão.
	roots := []string{"../app", "../serve"}
	set := token.NewFileSet()
	measured, seen := 0, map[string]bool{}
	for _, dir := range roots {
		root, err := filepath.Abs(dir)
		if err != nil {
			t.Fatalf("achar o %q: %v", dir, err)
		}
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
				if !ok || !invents(call) {
					return true
				}
				// O sítio é o arquivo mais a FUNÇÃO CHAMADA que recebeu o
				// contexto inventado, e não a linha: a linha muda quando alguém
				// mexe no arquivo acima, e aí a linha de base reprova sozinha.
				site := fmt.Sprintf("%s:%s", rel, receiverOfTheEmptyContext(file, call))
				seen[site] = true
				if baseline[site] {
					return true
				}
				t.Errorf("%s:%d inventa um contexto (`%s`) no meio de um gesto.\n"+
					"Receba o da requisição por parâmetro — as cenas já têm `r.Context()`\n"+
					"e os comandos da Mesa, `c.R.Context()`. Sem ele o gesto não cancela,\n"+
					"e o `Units.Do` de dentro abre a SEGUNDA transação em vez de reusar a\n"+
					"aberta: `SQLITE_BUSY` depois do `busy_timeout` inteiro (ALE-371).",
					rel, set.Position(call.Pos()).Line, site)
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("caminhar o %q: %v", dir, err)
		}
	}

	// A linha de base que ENCOLHEU tem de encolher no ARQUIVO também.
	for site := range baseline {
		if !seen[site] {
			t.Errorf("a linha de base ainda lista %q, e o sítio não existe mais.\n"+
				"Apague a entrada: uma dívida quitada que continua escrita faz o\n"+
				"próximo leitor achar que ainda há o que consertar.", site)
		}
	}

	// O DENOMINADOR: uma lista de reprovados vazia e um diretório não lido se
	// parecem no terminal.
	if measured < 100 {
		t.Fatalf("o guarda leu só %d arquivos em `app/` e `serve/` — ele está medindo o terreno errado", measured)
	}
}

// invents diz se a chamada é um `context.Background()` ou um `context.TODO()`.
func invents(call *ast.CallExpr) bool {
	target, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	pkg, ok := target.X.(*ast.Ident)
	if !ok || pkg.Name != "context" {
		return false
	}
	return target.Sel.Name == "Background" || target.Sel.Name == "TODO"
}

// receiverOfTheEmptyContext nomeia a função que RECEBEU o contexto inventado, e
// é ela que diz o que deixou de cancelar. Quando o contexto não é argumento de
// ninguém — um `ctx := context.Background()` —, o nome é o da função que o
// declara, que é a informação que sobra.
func receiverOfTheEmptyContext(file *ast.File, invented *ast.CallExpr) string {
	name := "<solto>"
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			for _, arg := range node.Args {
				if arg != ast.Expr(invented) {
					continue
				}
				if target, ok := node.Fun.(*ast.SelectorExpr); ok {
					name = target.Sel.Name
					return false
				}
				if target, ok := node.Fun.(*ast.Ident); ok {
					name = target.Name
					return false
				}
			}
		case *ast.FuncDecl:
			if node.Body != nil && node.Pos() <= invented.Pos() && invented.Pos() <= node.End() &&
				name == "<solto>" {
				name = node.Name.Name
			}
		}
		return true
	})
	return name
}
