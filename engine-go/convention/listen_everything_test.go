package convention

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// NINGUÉM ESCUTA TUDO EM PRODUÇÃO (ALE-639). O `events.EverythingForTests` existe para o detector da
// bancada, que confere se cada gravação que a Mesa mostra publicou o evento dela. Fora de teste, escutar
// tudo é o ouvinte que acorda a cada tecla em qualquer mesa da casa — o que o `Bus.Subscribe` existe para
// não deixar fácil.
func TestNoProductionCodeListensToEverything(t *testing.T) {
	root := filepath.Join("..", "..")
	output, err := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "engine-go/*.go").Output()
	if err != nil {
		t.Fatalf("git ls-files em %s: %v", root, err)
	}
	read := 0
	for _, relative := range strings.Split(strings.TrimRight(string(output), "\x00"), "\x00") {
		if relative == "" || strings.HasSuffix(relative, "_test.go") || strings.HasSuffix(relative, "infra/events/bus.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			t.Fatalf("ler %s: %v", relative, err)
		}
		read++
		if strings.Contains(string(body), "EverythingForTests(") {
			t.Errorf("%s chama o events.EverythingForTests fora de teste: assine o interesse da mesa, da "+
				"campanha ou da ficha que a tela mostra", relative)
		}
	}
	if read < 100 {
		t.Fatalf("o guarda leu %d arquivos de Go; a varredura encolheu", read)
	}
}
