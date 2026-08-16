package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// O teto da iniciativa existe DUAS vezes: aqui, onde o servidor recusa o 51º
// combatente, e no front, que trunca antes de mandar para relatar o corte na UI
// em vez de morrer num erro de socket no meio do laço. O comentário do front diz
// "espelho do backend" e nada segurava o espelho: se este lado baixasse para 40,
// o cliente continuaria mandando 50 e o mestre veria dez adições sumirem sem
// explicação.
//
// O teste vive do lado do SERVIDOR porque é ele que decide; o front é quem
// espelha.
func TestInitiativeCapMatchesFrontend(t *testing.T) {
	path := filepath.Join("..", "..", "frontend", "src", "features", "gm-tools", "encounter.ts")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	match := regexp.MustCompile(`INITIATIVE_MAX_ENTRIES = (\d+)`).FindSubmatch(raw)
	if match == nil {
		t.Fatal("não achei INITIATIVE_MAX_ENTRIES no front — o espelho mudou de nome e este teste ficou cego")
	}
	front, err := strconv.Atoi(string(match[1]))
	if err != nil {
		t.Fatalf("valor do front não é número: %v", err)
	}
	if front != initiativeMaxEntries {
		t.Errorf("teto do front = %d, servidor = %d — o cliente truncaria no número errado", front, initiativeMaxEntries)
	}
}
