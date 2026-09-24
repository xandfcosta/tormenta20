package convention

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// O ORÁCULO DE PARIDADE ESTÁ CONGELADO, E MEXER NELE É ATO DELIBERADO.
//
// # O que este guarda protege, e por que prosa não bastava
//
// O `parity/` são 18 fichas ponta a ponta, e ele é a rede de regressão da
// ALE-378 — a reescrita do motor de regras em ECS. O problema é que ele é
// gerado A PARTIR do motor que está sendo substituído: todo teste de paridade
// vermelho tem um botão verde a um `go run ./cmd/genoracle` de distância.
//
// O cabeçalho do gerador já pedia que não — "o diff de um oráculo é revisado
// contra o LIVRO, nunca aceito porque o teste ficou verde". Aquilo é prosa, e
// prosa é aplicada aos arquivos que alguém apontou. O que faz a regra valer é o
// guarda FORÇAR a parada, que é a tese da seção "Como uma convenção passa a
// valer" do guia.
//
// # Por que hash e não uma revisão de diff
//
// Um diff de 1,3 MB de JSON não se revisa lendo: a mudança que importa é um
// número no meio de uma ficha, e ela tem a mesma cara de uma reordenação de
// chave. O hash não pede revisão — ele pede DECISÃO, e a decisão fica no
// commit que levanta a linha de base.
//
// # O que fazer quando ele reprovar
//
// Se a regra do livro mudou de verdade: regenere, confira o diff CONTRA O
// LIVRO com a página citada, atualize `testdata/oracle_freeze.txt` e diga no
// corpo do commit contra o que conferiu. Se você está no meio da ALE-378 e o
// oráculo mudou sem você pedir, o motor novo discorda do velho — que é
// exatamente o que a rede existe para dizer.
func TestNoOracleFixtureChangesWithoutRaisingTheBaseline(t *testing.T) {
	dir := filepath.Join("..", "parity")
	base := oracleFreezeBaseline(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ler o %s: %v", dir, err)
	}

	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := entry.Name()
		seen[name] = true

		got := digestOf(t, filepath.Join(dir, name))
		want, known := base[name]
		if !known {
			t.Errorf("parity/%s é uma fixture NOVA e não está na linha de base.\n"+
				"Acrescente a linha `%s %s` em convention/testdata/oracle_freeze.txt, "+
				"e diga no commit por que a ficha nova entrou.", name, name, got)
			continue
		}
		if got != want {
			t.Errorf("parity/%s MUDOU.\n  linha de base: %s\n  no disco:      %s\n"+
				"Se a regra do livro mudou, confira o diff contra a PÁGINA e levante a base em "+
				"convention/testdata/oracle_freeze.txt. Se você está na ALE-378, isto é o motor "+
				"novo discordando do velho — conserte o motor, não a base.", name, want, got)
		}
	}

	for name := range base {
		if !seen[name] {
			t.Errorf("parity/%s está na linha de base e NÃO existe mais no disco.\n"+
				"Tire a linha de convention/testdata/oracle_freeze.txt — uma base que "+
				"afirma um arquivo morto é a que ninguém relê.", name)
		}
	}

	// O DENOMINADOR. Sem ele, "nada reprovou" e "o diretório sumiu" são a mesma
	// cor no terminal — que é a armadilha que o guia cataloga.
	if len(seen) == 0 {
		t.Fatalf("nenhuma fixture medida em %s: o guarda não está vendo o oráculo", dir)
	}
	t.Logf("fixtures do oráculo conferidas: %d", len(seen))
}

func digestOf(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ler %s: %v", path, err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func oracleFreezeBaseline(t *testing.T) map[string]string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "oracle_freeze.txt"))
	if err != nil {
		t.Fatalf("ler a linha de base: %v", err)
	}
	base := map[string]string{}
	for _, row := range strings.Split(string(raw), "\n") {
		row = strings.TrimSpace(row)
		if row == "" || strings.HasPrefix(row, "#") {
			continue
		}
		name, digest, ok := strings.Cut(row, " ")
		if !ok {
			t.Fatalf("linha de base malformada: %q — o formato é `<arquivo> <sha256>`", row)
		}
		base[name] = strings.TrimSpace(digest)
	}
	if len(base) == 0 {
		t.Fatal("a linha de base do oráculo está VAZIA: o guarda passaria sobre nada")
	}
	return base
}
