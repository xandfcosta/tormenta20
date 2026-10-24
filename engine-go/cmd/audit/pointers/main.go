// Command pointers confere se os arquivos CITADOS nos comentários dos testes
// ainda existem.
//
// Um teste que sai deixa escrito para onde a garantia foi, e um destino que não
// existe é pior que nenhuma nota: manda a próxima pessoa procurar um arquivo que
// não está lá (ALE-187). Varrer acha o que lembrar não acha.
//
// LIMITAÇÃO: a varredura não distingue "o destino sumiu" de "o comentário cita um
// arquivo que FOI SUBSTITUÍDO e diz isso" — o nome antigo é história, não
// endereço. A saída são candidatos, não veredito.
//
//	go run ./cmd/audit/pointers
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// cited é o nome entre crases. `\w` é a classe UNICODE do Python, que o RE2 não
// tem por padrão.
var cited = regexp.MustCompile("`([\\p{L}\\p{N}_./-]+\\.(?:ts|tsx|go))`")

func git(root string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, _ := cmd.Output()
	return string(out)
}

// exists pergunta ao git por qualquer arquivo rastreado com o mesmo nome-base:
// o ponteiro costuma citar só o nome, sem o caminho.
func exists(root, name string, memo map[string]bool) bool {
	base := filepath.Base(name)
	if found, ok := memo[base]; ok {
		return found
	}
	found := strings.TrimSpace(git(root, "ls-files", "*"+base)) != ""
	memo[base] = found
	return found
}

type broken struct{ file, name string }

func main() {
	root := strings.TrimSpace(git(".", "rev-parse", "--show-toplevel"))
	files := strings.Fields(git(root, "ls-files", "*.test.ts", "*.test.tsx", "*_test.go", "*.spec.ts"))

	memo := map[string]bool{}
	var candidates []broken
	alive := 0
	for _, rel := range files {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			// Só COMENTÁRIO: um import ou uma string de código não é ponteiro.
			if !strings.Contains(line, "//") && !strings.Contains(line, "*") {
				continue
			}
			for _, m := range cited.FindAllStringSubmatch(line, -1) {
				if exists(root, m[1], memo) {
					alive++
				} else {
					candidates = append(candidates, broken{rel, m[1]})
				}
			}
		}
	}

	fmt.Printf("ponteiros conferidos: %d · vivos: %d · candidatos: %d\n", alive+len(candidates), alive, len(candidates))
	for _, c := range candidates {
		fmt.Printf("  %s  →  %s\n", c.file, c.name)
	}
	if len(candidates) > 0 {
		fmt.Println("\nConfira cada um: citar um arquivo que foi SUBSTITUÍDO, dizendo isso,")
		fmt.Println("é história e não endereço quebrado.")
	}
}
