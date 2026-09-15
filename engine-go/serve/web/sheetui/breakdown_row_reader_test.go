package sheetui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TODO CAMPO DE `breakdownRow` ESCRITO TEM QUEM O LEIA.
//
// A família é a do `TestEverySignalDeclaredByValueHasAReader`: alguma coisa é
// declarada com cuidado, documentada em três linhas de comentário, e NINGUÉM a
// lê — então ela não estoura, ela faz o desenho "não fazer nada".
//
// O caso que originou o guarda foi o `Indented`. Ele existe para a linha que
// EXPLICA a de cima em vez de somar ao lado dela — as contribuições de item sob
// o "Outros" das Perícias —, era escrito em `expertises.go` e não aparecia em
// `.templ` nenhum. O efeito na tela é uma decomposição que NÃO FECHA:
//
//	½ nível +5 · Atributo (DES) +0 · Treino +0 · Outros −7
//	Penalidade de armadura −7          ← esta EXPLICA o "Outros", não soma
//	TOTAL −2
//
// Quem lê soma −9 e vê −2. É exatamente a família que o `CLAUDE.md` descreve em
// "toda decomposição afirma a soma antes de afirmar as parcelas", acontecida na
// tela que existe PARA mostrar a decomposição.
//
// # Por que este guarda, e não um teste do valor
//
// Porque o valor estava CERTO. `5 + 0 + 0 − 7 = −2` fecha no dado, e qualquer
// asserção sobre `ExpertiseBreakdown` passaria verde sobre o defeito: ele era
// 100% de apresentação. O que é mecanizável aqui é a PONTA SOLTA — campo escrito
// sem leitor —, e a aparência (o recuo lê como subordinação?) se confere
// OLHANDO, que foi como este defeito apareceu.
//
// # O que ele NÃO pega
//
// Um campo lido por um `.templ` que o desenha sem efeito visual nenhum. O guarda
// prende a LIGAÇÃO, não o desenho — de propósito: prender aparência em teste é a
// família inteira de armadilhas do `CLAUDE.md`.
func TestEveryBreakdownRowFieldHasAReader(t *testing.T) {
	written, sites := breakdownRowFieldsWritten(t)
	read := selectorsReadInPackage(t)

	// O DENOMINADOR. Sem ele, um `breakdownRow` renomeado deixa `written` vazio e
	// o guarda passa verde sem ter olhado nada — que é a diferença entre "nada
	// reprovou" e "não mediu".
	if len(written) < 4 || sites < 20 {
		t.Fatalf("a varredura achou %d campos escritos em %d literais de `breakdownRow` — "+
			"esperava ao menos 4 campos em 20 sítios (eram 23 no dia). "+
			"O tipo foi renomeado ou o parser parou de achá-lo",
			len(written), sites)
	}

	var loose []string
	for _, field := range written {
		if !read[field] {
			loose = append(loose, field)
		}
	}
	sort.Strings(loose)

	if len(loose) > 0 {
		t.Errorf("%d campo(s) de `breakdownRow` são ESCRITOS e nenhum `.templ` os lê: %s\n"+
			"Um campo sem leitor não estoura — ele faz o desenho não acontecer.\n"+
			"Ou desenhe o campo em `combat.templ`, ou apague-o de quem o escreve.\n"+
			"(medidos: %d campos em %d literais)",
			len(loose), strings.Join(loose, ", "), len(written), sites)
	}
}

// breakdownRowFieldsWritten colhe os campos nomeados em todo literal
// `breakdownRow{…}` do pacote, e quantos literais foram vistos.
func breakdownRowFieldsWritten(t *testing.T) ([]string, int) {
	t.Helper()
	seen := map[string]bool{}
	sites := 0
	for _, path := range packageSources(t, ".go") {
		fileSet := token.NewFileSet()
		tree, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		ast.Inspect(tree, func(n ast.Node) bool {
			literal, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			// O TIPO ELIDIDO é metade do terreno, e a primeira versão deste guarda
			// não o via: dentro de `[]breakdownRow{ {…}, {…} }` o Go permite omitir
			// o nome do tipo, então o `Type` do literal de dentro é NIL. Medido — o
			// denominador acusou 10 literais onde um `grep` de `breakdownRow{`
			// achava 18, e os que faltavam eram todos desta forma, o
			// `expertiseBreakdownRows` entre eles. Guarda que varre um nível não
			// varre.
			if isBreakdownRowSlice(literal.Type) {
				for _, element := range literal.Elts {
					if inner, ok := element.(*ast.CompositeLit); ok && inner.Type == nil {
						sites++
						collectFieldNames(inner, seen)
					}
				}
				return true
			}
			if name, ok := literal.Type.(*ast.Ident); ok && name.Name == "breakdownRow" {
				sites++
				collectFieldNames(literal, seen)
			}
			return true
		})
	}
	fields := make([]string, 0, len(seen))
	for field := range seen {
		fields = append(fields, field)
	}
	sort.Strings(fields)
	return fields, sites
}

// isBreakdownRowSlice diz se o literal é uma FATIA de `breakdownRow`, cujos
// elementos podem omitir o nome do tipo.
func isBreakdownRowSlice(declared ast.Expr) bool {
	slice, ok := declared.(*ast.ArrayType)
	if !ok {
		return false
	}
	element, ok := slice.Elt.(*ast.Ident)
	return ok && element.Name == "breakdownRow"
}

// collectFieldNames anota os campos NOMEADOS de um literal.
func collectFieldNames(literal *ast.CompositeLit, seen map[string]bool) {
	for _, element := range literal.Elts {
		pair, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if key, ok := pair.Key.(*ast.Ident); ok {
			seen[key.Name] = true
		}
	}
}

// fieldRead casa a LEITURA de um campo (`.Campo`), que é a forma que a escrita
// (`Campo:`) nunca tem — é isso que separa um lado do outro sem AST.
var fieldRead = regexp.MustCompile(`\.([A-Z][A-Za-z0-9_]*)\b`)

// selectorsReadInPackage colhe todo `.Campo` lido nos `.templ` e nos `.go`
// escritos à mão do pacote.
func selectorsReadInPackage(t *testing.T) map[string]bool {
	t.Helper()
	read := map[string]bool{}
	for _, path := range append(packageSources(t, ".templ"), packageSources(t, ".go")...) {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		for _, m := range fieldRead.FindAllStringSubmatch(string(body), -1) {
			read[m[1]] = true
		}
	}
	return read
}

// packageSources lista as fontes do diretório, fora o gerado e os testes.
func packageSources(t *testing.T, extension string) []string {
	t.Helper()
	all, err := filepath.Glob("*" + extension)
	if err != nil {
		t.Fatalf("listar *%s: %v", extension, err)
	}
	var sources []string
	for _, path := range all {
		if strings.HasSuffix(path, "_templ.go") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		sources = append(sources, path)
	}
	if len(sources) == 0 {
		t.Fatalf("nenhuma fonte *%s no pacote — o guarda ficou cego", extension)
	}
	return sources
}
