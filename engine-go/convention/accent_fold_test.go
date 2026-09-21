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

// A DOBRA DE ACENTO TEM UM LUGAR SÓ: o `domain/search.Fold`.
//
// Ela já teve dois. Uma tabela de nove pares vivia no `serve/web/sheetui`, com
// doze chamadores, enquanto o buscador do livro dobrava por NFD — e o argumento
// escrito na tabela era *"o alfabeto que a mesa digita é conhecido e cabe em
// nove pares"*.
//
// O argumento era verdadeiro e insuficiente. O que a mesa digita inclui o nome
// que um JOGADOR inventa para um ofício ou um item, e ali cabe qualquer acento:
// medido, as duas dobras concordavam em 29 de 34 casos e discordavam em `ñ`,
// `è`, `ï`, `å` e `ö` — todos os cinco a favor da que ficou (ALE-352).
//
// # O modo de falha é SILENCIOSO, e é por isso que isto é guarda
//
// Uma segunda tabela não quebra nada: ela acha quase tudo. O que ela produz é
// uma busca que funciona em nove das dez abas e falha na décima, com a mesma
// palavra digitada — e ninguém liga uma coisa à outra.
//
// # O que ele procura
//
// Um `strings.NewReplacer` cujos argumentos são letras acentuadas. É a forma que
// a tabela tinha, e é a forma que a próxima teria: ninguém escreve uma dobra de
// acento sem listar os pares.
func TestNoSecondAccentFolderIsWritten(t *testing.T) {
	const ondeADobraMora = "domain/search"

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	set := token.NewFileSet()
	measured := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_templ.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		// O guarda cita as letras para poder proibi-las, e o dono da dobra pode
		// listá-las à vontade.
		if strings.HasPrefix(rel, ondeADobraMora) || strings.HasSuffix(rel, "accent_fold_test.go") {
			return nil
		}
		file, err := parser.ParseFile(set, path, nil, 0)
		if err != nil {
			return err
		}
		measured++
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || !isNewReplacer(call) || !hasAccentedArgument(call) {
				return true
			}
			t.Errorf("%s:%d monta uma SEGUNDA dobra de acento.\n"+
				"Use o `search.Fold` do `%s`: uma tabela a mais acha quase tudo, e o que\n"+
				"ela produz é a mesma palavra achando numa aba e não achando na outra.",
				rel, set.Position(call.Pos()).Line, ondeADobraMora)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}

	// O DENOMINADOR: uma lista de reprovados vazia e uma varredura que não abriu
	// arquivo nenhum se parecem no terminal.
	if measured < 200 {
		t.Fatalf("o guarda leu só %d arquivos — ele está medindo a árvore errada", measured)
	}
}

func isNewReplacer(call *ast.CallExpr) bool {
	target, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || target.Sel.Name != "NewReplacer" {
		return false
	}
	pkg, ok := target.X.(*ast.Ident)
	return ok && pkg.Name == "strings"
}

// hasAccentedArgument separa a dobra de acento de um `NewReplacer` legítimo —
// escapar HTML, trocar barra por traço. O que denuncia é a LETRA acentuada.
func hasAccentedArgument(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		for _, r := range lit.Value {
			if r > 127 && r < 0x2000 {
				return true
			}
		}
	}
	return false
}
