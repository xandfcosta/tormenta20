package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TODA CLASSE APLICADA EXISTE NA FOLHA COMPILADA (ALE-301).
//
// Classe que não está na folha não dá erro: o elemento aparece SEM ESTILO. É a
// armadilha que o `engine-go/CLAUDE.md` registra em duas linhas — a classe que
// não passou pelo scanner e o TOKEN inventado — e ela é a mesma de qualquer
// renome pela metade.
//
// # Por que UM guarda pega os DOIS lados
//
// A folha compilada é derivada da folha-fonte MAIS o que o Tailwind acha varrendo
// o repositório. Então:
//
//   - renomeei na FONTE e esqueci o `.templ` → o nome velho do `.templ` não está
//     mais na folha compilada (a regra da casa sumiu, e o Tailwind não conhece o
//     nome) → vermelho;
//   - renomeei no `.templ` e esqueci a FONTE → o nome novo não é utilitário que o
//     Tailwind saiba emitir, e a fonte não o declara → vermelho.
//
// Não é preciso uma lista do que é "classe da casa": a folha compilada já é o
// denominador, porque o Tailwind emite tudo que ele VÊ. Um guarda que dependesse
// de uma lista de prefixos subcontaria em silêncio, que é o defeito medido na
// varredura dos nomes de arquivo desta mesma issue.
//
// # O que ele NÃO mede, dito de propósito
//
//   - TOKEN COM VARIANTE (`lg:static`, `hover:bg-x`) fica fora: o `:` some do
//     nome da regra emitida, e casar isso pediria reimplementar o Tailwind.
//   - FRAGMENTO que termina em `-` é classe montada em tempo de execução
//     (`"ground-" + v.Chao`). Quem cobra essas é o guarda da família — o
//     `TestEveryOfferedGroundCanBePainted` para o chão, e o do terreno para os
//     pincéis —, e é lá que elas têm o denominador certo.
//   - ARQUIVO DE TESTE fica fora porque ele cita classe MORTA de propósito: o
//     `stage_entry_test.go` afirma que `grimorio-book` NÃO existe, e um guarda
//     que lesse aquilo acusaria o teste que protege a ausência.
var (
	classAttribute = regexp.MustCompile(`class=[{"]([^"}]*"[^"}]*)?`)
	classLiteral   = regexp.MustCompile(`"([^"]*)"`)
	cssRule        = regexp.MustCompile(`\.([a-zA-Z][a-zA-Z0-9_-]*)`)
	plainClassName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
)

func TestEveryAppliedClassExistsInTheStylesheet(t *testing.T) {
	root := filepath.Join("..", "..")
	folha, err := os.ReadFile(filepath.Join(root, "engine-go", "api", "piloto", "static", "piloto.css"))
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	naFolha := map[string]bool{}
	for _, m := range cssRule.FindAllStringSubmatch(string(folha), -1) {
		naFolha[m[1]] = true
	}
	// O DENOMINADOR DA FOLHA: uma folha vazia faria todo token reprovar, e uma
	// folha que o `ReadFile` pegou pela metade faria uma lista de falhas com
	// cara de descoberta.
	if len(naFolha) < 500 {
		t.Fatalf("a folha compilada tem só %d classes — ela é o denominador, e está curta demais", len(naFolha))
	}

	saida, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "*.templ", "*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}

	checked := map[string]string{}
	filesRead := 0
	for _, relative := range strings.Split(strings.TrimRight(string(saida), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_templ.go") ||
			strings.HasSuffix(relative, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		filesRead++
		for _, linha := range strings.Split(string(body), "\n") {
			if strings.HasPrefix(strings.TrimSpace(linha), "//") {
				continue
			}
			for _, attr := range classAttribute.FindAllString(linha, -1) {
				for _, lit := range classLiteral.FindAllStringSubmatch(attr, -1) {
					for _, tok := range strings.Fields(lit[1]) {
						if !plainClassName.MatchString(tok) || strings.HasSuffix(tok, "-") {
							continue
						}
						if _, já := checked[tok]; !já {
							checked[tok] = relative
						}
					}
				}
			}
		}
	}

	if filesRead < 200 || len(checked) < 150 {
		t.Fatalf("a varredura leu %d arquivos e %d classes aplicadas — a raiz é o primeiro suspeito",
			filesRead, len(checked))
	}

	var soltas []string
	for tok, onde := range checked {
		if !naFolha[tok] {
			soltas = append(soltas, tok+" — aplicada em "+onde)
		}
	}
	sort.Strings(soltas)
	if len(soltas) > 0 {
		t.Errorf("classe aplicada que NÃO existe na folha compilada — %d de %d:\n  %s\n"+
			"O elemento aparece SEM ESTILO e nada estoura. Ou a classe foi renomeada só de um "+
			"lado (a folha-fonte e o `.templ` são dois lugares que nada liga), ou ela é nova e "+
			"falta rodar `engine-go/scripts/build-piloto-css.sh`.",
			len(soltas), len(checked), strings.Join(soltas, "\n  "))
	}
	t.Logf("classes aplicadas: %d, todas na folha, de %d arquivos e %d regras", len(checked), filesRead, len(naFolha))
}
