// Command tsgen escreve os tipos TypeScript da fronteira do motor.
//
// Uso: cd engine-go && go generate ./engine   (ou go run ./cmd/tsgen)
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"t20engine/engine"
)

// out é relativo à RAIZ DO REPOSITÓRIO, não ao cwd: `go generate` roda com o
// diretório do pacote como cwd, então um caminho relativo simples aponta para o
// lugar errado dependendo de onde foi chamado.
const out = "frontend/src/shared/api/engine-types.ts"

func main() {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := filepath.Join(root, filepath.FromSlash(out))
	if err := os.WriteFile(path, []byte(engine.GenerateTypeScript()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "escrever %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("%s escrito\n", path)
}

// repoRoot sobe do cwd até achar o diretório que contém `engine-go` e
// `frontend` lado a lado.
func repoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if isRepoRoot(dir) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("tsgen: não achei a raiz do repositório subindo de %q", dir)
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	for _, marker := range []string{"engine-go", "frontend"} {
		if _, err := os.Stat(filepath.Join(dir, marker)); err != nil {
			return false
		}
	}
	return true
}
