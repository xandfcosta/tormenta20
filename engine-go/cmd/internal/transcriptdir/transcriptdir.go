// Package transcriptdir acha o diretório do livro transcrito a partir de onde a
// ferramenta foi chamada, para as ferramentas que reescrevem o catálogo
// funcionarem da raiz do repositório, do `engine-go/` ou de uma worktree.
package transcriptdir

import (
	"fmt"
	"os"
	"path/filepath"
)

// Find sobe a partir do diretório atual até achar
// `engine-go/domain/catalog/transcript` (ou `domain/catalog/transcript`).
//
//	dir, err := transcriptdir.Find()
func Find() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; {
		for _, rel := range []string{"engine-go/domain/catalog/transcript", "domain/catalog/transcript"} {
			candidate := filepath.Join(dir, rel)
			if _, err := os.Stat(filepath.Join(candidate, "layout.go")); err == nil {
				return candidate, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("não achei domain/catalog/transcript subindo de %q", cwd)
		}
		dir = parent
	}
}
