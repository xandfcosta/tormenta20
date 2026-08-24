package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"t20engine/db"
)

// A BANCADA desta suíte: um banco migrado UMA vez, copiado por teste (ALE-260).
//
// O problema medido: `newTestServer` abria um SQLite novo por teste e `db.Open`
// roda as 9 migrações nele. São ~3.400 migrações com `fsync`, e o custo de um
// `fsync` é o do dispositivo onde o `TMPDIR` cai. Números desta máquina:
//
//	                        tmpfs        disco girante
//	migrar do zero .....    7,1 ms          2.102 ms
//	copiar o molde .....      ~0,1 ms           0,076 ms
//	reabrir já migrado .      ~1 ms             1 ms
//
// No disco girante cada migração custava ~49 ms — a latência de rotação do prato,
// uma por `fsync` —, e a suíte do `api/` levava 15m01s contra 6,8 s da suíte Go
// inteira em tmpfs. Copiar o molde e reabrir custa ~1,1 ms contra 2,1 s: 1900×.
//
// POR QUE O MOLDE E NÃO UMA VARIÁVEL DE AMBIENTE. A issue propunha exportar
// `TMPDIR` para um tmpfs, e isso funciona — mas só para quem lembrar, e só na
// máquina de quem lembrou. Quando isto foi medido havia uma sessão vizinha
// rodando com `TMPDIR` no disco girante sem saber que pagava quinze minutos por
// corrida. O molde tira o disco da conta em vez de pedir que alguém escolha o
// disco certo: o que a bancada não deixa acontecer ninguém precisa lembrar de
// evitar.
//
// O `db.Open` NÃO mudou, e é o que mantém o teste honesto: o banco de teste
// passa pelo mesmo `Open` de produção, com o mesmo `assertSchema` (ALE-154). O
// goose só encontra a versão 9 já aplicada e não tem o que fazer.
var moldeDoBanco string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "t20-molde-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "criar o molde do banco: %v\n", err)
		os.Exit(1)
	}
	moldeDoBanco = filepath.Join(dir, "molde.db")
	base, err := db.Open(moldeDoBanco)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrar o molde do banco: %v\n", err)
		os.Exit(1)
	}
	// Fechar ANTES de qualquer cópia: o `Close` faz o checkpoint do WAL, e um
	// molde copiado com escrita pendente no `-wal` nasceria sem as tabelas que o
	// `assertSchema` exige — o teste falharia dizendo "falta tabela", que manda
	// procurar defeito na migração e não na cópia.
	_ = base.Close()

	codigo := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(codigo)
}

// bancoDeTeste devolve um banco migrado e VIRGEM, copiando o molde. O arquivo
// vive no `t.TempDir()`, então o próprio `testing` o apaga.
func bancoDeTeste(t *testing.T) string {
	t.Helper()
	destino := filepath.Join(t.TempDir(), "test.db")
	if err := copiaArquivo(moldeDoBanco, destino); err != nil {
		t.Fatalf("copiar o molde do banco para %q: %v", destino, err)
	}
	return destino
}

func copiaArquivo(de, para string) error {
	origem, err := os.Open(de)
	if err != nil {
		return fmt.Errorf("abrir %q: %w", de, err)
	}
	defer func() { _ = origem.Close() }()
	destino, err := os.Create(para)
	if err != nil {
		return fmt.Errorf("criar %q: %w", para, err)
	}
	defer func() { _ = destino.Close() }()
	if _, err := io.Copy(destino, origem); err != nil {
		return fmt.Errorf("copiar %q para %q: %w", de, para, err)
	}
	return nil
}
