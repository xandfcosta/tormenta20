package convention

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// trackedFiles são os arquivos do ÍNDICE do git que casam os padrões, com o caminho
// a partir desta pasta (`../../e2e/...`), que é o que `os.ReadFile` e as mensagens
// de falha usam.
//
// O ÍNDICE, e não o disco: o disco da máquina tem o que a CI não tem — o
// `e2e/test-results` que o Playwright deixa depois de uma falha, um rascunho fora do
// git —, e um guarda que lê o disco reprova na máquina o que passa na CI. É a regra
// do `CLAUDE.md`: `git add` antes do `go test`.
//
//	trackedFiles(t, "*.go", "*.md") // [../../engine-go/main.go ../../README.md …]
func trackedFiles(t *testing.T, patterns ...string) []string {
	t.Helper()
	root := filepath.Join("..", "..")
	args := append([]string{"-C", root, "ls-files", "-z", "--cached", "--"}, patterns...)
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}
	var files []string
	for _, relative := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if relative != "" {
			files = append(files, filepath.Join(root, relative))
		}
	}
	return files
}
