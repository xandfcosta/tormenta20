package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// O GERADOR DA SEED CONTINUA GERANDO, e o que ele gera é o que está commitado.
//
// O gerador já parou de rodar inteiro **sem que nada acusasse**: ele alcança o
// que usa por CAMINHO EM STRING, então o `go build` fica verde e uma varredura
// de órfãs por SÍMBOLO não alcança. Por isso o caso RODA o gerador de verdade
// num arquivo temporário e compara com o `seed.sql` do repositório — o defeito
// não é de compilação nem de asserção, é o programa inteiro não terminar.
//
// Custa ~1s, e vale: o `seed.sql` é como uma máquina nova ganha dado, e um
// gerador quebrado só aparece no dia em que alguém precisa dele.
//
// Ele também prende que o gerador é DETERMINÍSTICO. As datas são constantes e o
// despejo normaliza carimbos justamente para isso; se alguém escrever um
// `time.Now()` no caminho, o arquivo passa a diferir a cada corrida e este caso
// reprova na primeira.
func TestTheSeedGeneratorStillWritesTheCommittedFile(t *testing.T) {
	raiz := moduleRoot(t)
	saida := filepath.Join(t.TempDir(), "seed.sql")

	// Subprocesso e não chamada direta: o `main` escreve arquivo e usa o
	// diretório de trabalho para achar o `.env` e o catálogo. Um `os.Chdir` no
	// teste vazaria para os outros casos do pacote.
	cmd := exec.Command("go", "run", "./cmd/seed", saida)
	cmd.Dir = raiz
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("o gerador não rodou: %v\n%s", err, out)
	}

	gerado, err := os.ReadFile(saida)
	if err != nil {
		t.Fatalf("ler o gerado: %v", err)
	}
	commitado, err := os.ReadFile(filepath.Join(raiz, "seed.sql"))
	if err != nil {
		t.Fatalf("ler o commitado: %v", err)
	}
	if string(gerado) != string(commitado) {
		t.Errorf("o `seed.sql` commitado não é o que o gerador escreve hoje.\n"+
			"Se a mudança foi INTENCIONAL (elenco, regra, crônica), rode\n"+
			"`go run ./cmd/seed` em engine-go/ e commite o resultado.\n"+
			"gerado: %d bytes, commitado: %d bytes", len(gerado), len(commitado))
	}
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cwd: %v", err)
	}
	for range 5 {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("não achei o go.mod subindo a partir do diretório do teste")
	return ""
}
