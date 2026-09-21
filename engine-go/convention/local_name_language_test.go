package convention

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// PARÂMETRO, VARIÁVEL LOCAL E CAMPO TAMBÉM SÃO IDENTIFICADOR (ALE-367).
//
// O `TestNoNewIdentifierIsWrittenInPortuguese` varre só declaração de TOPO, e o
// `CLAUDE.md` manda inglês para variável também. O buraco foi medido quando o
// dono o apontou: ~6.700 nomes em 512 arquivos, de 22.868 — `custo`, `quando`,
// `teve`, `quero` —, todos com a suíte verde. A varredura que os traduziu é a
// ALE-367; este guarda é o que impede a conta de voltar a crescer.
//
// # Por que o vocabulário é MAIOR que o do guarda de topo
//
// A lista de lá é curta de propósito, e o motivo continua valendo lá: acusar
// inglês correto faz o guarda ser desligado. Mas foi ela que deixou `custo` e
// `quando` passarem. Aqui ela é somada às palavras que a ALE-367 de fato
// traduziu — o vocabulário que este código USOU, e não um dicionário —, menos
// as que também são inglês. Fica em `testdata/portuguese_local_words.txt`.
//
// # O que ele NÃO lê
//
// No TypeScript, parâmetro de `function` e propriedade de objeto. Parâmetro de
// `function` pediria um analisador de verdade (o compilador do projeto é o 7,
// sem API JavaScript); propriedade é chave de DOM ou de fio, e muitas vezes
// contrato — o `{falhas, medidos}` dos medidores é chave de objeto documentada.
const (
	localWordsFile    = "testdata/portuguese_local_words.txt"
	localBaselineFile = "testdata/portuguese_locals.txt"
)

// properNouns são nomes próprios: não são tradução pendente, são o nome da coisa.
var properNouns = map[string]bool{"tormenta": true, "tibar": true, "samira": true, "wynlla": true,
	"arton": true, "valkaria": true}

var (
	tsLocalDecl  = regexp.MustCompile(`^\s+(?:const|let|var)\s+([A-Za-z_$][\w$]*)`)
	tsForOf      = regexp.MustCompile(`for\s*\(\s*(?:const|let)\s+([A-Za-z_$][\w$]*)\s+(?:of|in)\b`)
	tsArrowParam = regexp.MustCompile(`\(\s*([A-Za-z_$][\w$]*)\s*(?::[^)]*)?\)\s*=>`)
)

func TestNoLocalNameIsWrittenInPortuguese(t *testing.T) {
	words := map[string]bool{}
	for w := range portugueseWords {
		words[w] = true
	}
	for _, l := range readLines(t, localWordsFile) {
		words[l] = true
	}
	isPortuguese := func(name string) bool {
		spaced := camelBoundary.ReplaceAllString(strings.ReplaceAll(name, "_", " "), "$1 $2")
		for _, seg := range strings.Fields(spaced) {
			s := strings.ToLower(seg)
			if words[s] && !properNouns[s] {
				return true
			}
		}
		return false
	}
	baseline := map[string]bool{}
	for _, l := range readLines(t, localBaselineFile) {
		baseline[l] = true
	}

	findings := map[string]bool{}
	measured, files := 0, 0
	note := func(relative, name string) {
		measured++
		if isPortuguese(name) {
			findings[relative+":"+name] = true
		}
	}
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "test-results", "playwright-report", "dist", "backups", "data", "sqlcgen":
				return fs.SkipDir
			}
			return nil
		}
		relative := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(path, root)), "/")
		switch {
		case strings.HasSuffix(path, ".go"):
			f, perr := parser.ParseFile(fset, path, nil, 0)
			if perr != nil {
				t.Errorf("%s não é Go legível: %v — o guarda FALHA no que não sabe ler", relative, perr)
				return nil
			}
			files++
			// o nome de um parâmetro de componente vem do .templ: é lá que se renomeia
			shown := relative
			if strings.HasSuffix(relative, "_templ.go") {
				shown = strings.TrimSuffix(relative, "_templ.go") + ".templ"
			}
			for _, name := range localGoNames(f) {
				if !strings.HasPrefix(name, "templ_7745c5c3_") {
					note(shown, name)
				}
			}
		case strings.HasSuffix(path, ".ts") && !strings.HasSuffix(path, ".d.ts"):
			body, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			files++
			for _, row := range strings.Split(string(body), "\n") {
				for _, rx := range []*regexp.Regexp{tsLocalDecl, tsForOf, tsArrowParam} {
					for _, m := range rx.FindAllStringSubmatch(row, -1) {
						note(relative, m[1])
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrer: %v", err)
	}
	// O DENOMINADOR: são ~23 mil declarações; um piso folgado denuncia a raiz
	// trocada e o walk que parou cedo, não uma fatia que apagou funções.
	if files < 300 || measured < 15000 {
		t.Fatalf("a varredura leu %d arquivos e %d nomes locais — a raiz é o primeiro suspeito", files, measured)
	}

	var fresh, gone []string
	for k := range findings {
		if !baseline[k] {
			fresh = append(fresh, k)
		}
	}
	for k := range baseline {
		if !findings[k] {
			gone = append(gone, k)
		}
	}
	sort.Strings(fresh)
	sort.Strings(gone)
	if len(fresh) > 0 {
		t.Errorf("nome LOCAL em português — parâmetro, variável ou campo —, e a regra pede inglês "+
			"(CLAUDE.md, \"Idioma\"): %d\n  %s\nRenomeie; nome de .templ se renomeia no .templ. "+
			"Nome próprio vai para `properNouns`, e só nome próprio.",
			len(fresh), strings.Join(fresh, "\n  "))
	}
	if len(gone) > 0 {
		t.Errorf("a linha de base %s cita %d nome(s) que não existem mais:\n  %s\n"+
			"Tire-os: uma catraca que não encolhe deixa de ser catraca.",
			localBaselineFile, len(gone), strings.Join(gone, "\n  "))
	}
	t.Logf("%d nomes locais em português (linha de base), de %d medidos em %d arquivos",
		len(findings), measured, files)
}

// localGoNames são os nomes que o guarda de topo não vê: parâmetro, resultado,
// receptor, variável e constante locais, variável de range e campo de struct.
func localGoNames(f *ast.File) []string {
	var out []string
	fields := func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, fl := range list.List {
			for _, id := range fl.Names {
				out = append(out, id.Name)
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.StructType:
			fields(x.Fields)
		case *ast.FuncType:
			fields(x.Params)
			fields(x.Results)
		case *ast.FuncDecl:
			fields(x.Recv)
		case *ast.AssignStmt:
			if x.Tok == token.DEFINE {
				for _, e := range x.Lhs {
					if id, ok := e.(*ast.Ident); ok && id.Name != "_" {
						out = append(out, id.Name)
					}
				}
			}
		case *ast.DeclStmt:
			if g, ok := x.Decl.(*ast.GenDecl); ok {
				for _, sp := range g.Specs {
					if vs, ok := sp.(*ast.ValueSpec); ok {
						for _, id := range vs.Names {
							out = append(out, id.Name)
						}
					}
				}
			}
		case *ast.RangeStmt:
			for _, e := range []ast.Expr{x.Key, x.Value} {
				if id, ok := e.(*ast.Ident); ok && id.Name != "_" {
					out = append(out, id.Name)
				}
			}
		}
		return true
	})
	return out
}

// readLines lê um arquivo de testdata, sem linhas vazias e sem comentário `#`.
func readLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	var out []string
	for _, l := range strings.Split(string(raw), "\n") {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}
