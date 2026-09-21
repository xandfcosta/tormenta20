package convention

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// NENHUMA CENA ESCREVE SQL.
//
// # O mecanismo, e por que ele chega agora
//
// Uma cena que grava direto no banco decide o que gravar, e aí a REGRA fica de
// um lado da fronteira e o `INSERT` do outro — a autorização numa função, a
// ordem das escritas noutra, e nenhuma das duas alcançável de fora do HTTP. Foi
// o diagnóstico da ALE-350, e onze famílias desceram para o `app/` por causa
// dele, uma a uma, da ALE-347 à ALE-353.
//
// **Nenhum mecanismo impedia a décima segunda de entrar.** A convenção estava
// escrita e era aplicada exatamente aos arquivos que alguém apontou — e a seção
// "Como uma convenção passa a valer" diz o que falta: o guarda que FORÇA a
// varredura, para a suíte só ficar verde quando o último caso foi tratado
// (ALE-359).
//
// # Ele descobre o que ESCREVE, em vez de enumerar
//
// A lista de queries de escrita sai do `query.sql`, lendo qual verbo cada uma
// usa. Enumerar à mão aqui teria o defeito de sempre: uma query nova nasceria
// fora da varredura, em silêncio. É a mesma forma do
// `TestEveryVitalQueryIsKnownToTheFunnelGuard`.
//
// # O que ele NÃO proíbe
//
// LER. A cena desenha o que o banco tem, e uma leitura não decide nada — o
// `Queries()` continua na porta de quem lê. O que ele proíbe é a cena ser a
// última a saber o que foi gravado.
func TestNoSceneWritesSql(t *testing.T) {
	writers := writingQueries(t)
	if len(writers) < 30 {
		t.Fatalf("o guarda achou só %d queries de escrita no `query.sql` — "+
			"o formato do arquivo mudou e ele parou de ler", len(writers))
	}

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	set := token.NewFileSet()
	measured, readingsSeen := 0, 0
	err = filepath.WalkDir(filepath.Join(root, "serve", "web"), func(
		path string, d fs.DirEntry, err error,
	) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_templ.go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, _ := filepath.Rel(root, path)
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
			if !ok {
				return true
			}
			if !writers[target.Sel.Name] {
				// O denominador: conta as LEITURAS de query para saber que o
				// seletor ainda casa com a forma "receptor ponto nome-da-query".
				if strings.HasPrefix(target.Sel.Name, "List") || strings.HasPrefix(target.Sel.Name, "Get") {
					readingsSeen++
				}
				return true
			}
			t.Errorf("%s:%d chama a query de ESCRITA %s.\n"+
				"Cena não grava: ela desenha. Quem decide o que vai para o banco é um caso\n"+
				"de uso em `app/`, e é lá que a autorização e a ORDEM das escritas moram —\n"+
				"onze famílias desceram por esse motivo entre a ALE-347 e a ALE-353.",
				rel, set.Position(call.Pos()).Line, target.Sel.Name)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("caminhar o `serve/web`: %v", err)
	}

	// O DENOMINADOR, nas duas pontas: uma varredura que não abriu arquivo e um
	// seletor que deixou de casar se parecem com "nada reprovou".
	if measured < 40 {
		t.Fatalf("o guarda leu só %d arquivos de cena — está medindo a árvore errada", measured)
	}
	if readingsSeen < 5 {
		t.Fatalf("o guarda não viu nenhuma LEITURA de query nas cenas (%d) — "+
			"a forma da chamada mudou e ele parou de reconhecer query nenhuma", readingsSeen)
	}
}

// writingQueries devolve, do `query.sql`, o nome de toda query que ESCREVE.
//
// Pelo VERBO e não por uma lista: `INSERT`, `UPDATE` e `DELETE` são o que existe
// de escrita em SQL, e uma query nova com qualquer um deles entra sozinha.
func writingQueries(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "infra", "db", "query.sql"))
	if err != nil {
		t.Fatalf("ler o query.sql: %v", err)
	}
	writers := map[string]bool{}
	for _, block := range strings.Split(string(raw), "-- name: ")[1:] {
		name, _, _ := strings.Cut(block, " ")
		body := strings.ToLower(block)
		if strings.Contains(body, "insert ") || strings.Contains(body, "update ") ||
			strings.Contains(body, "delete ") {
			writers[name] = true
		}
	}
	return writers
}
