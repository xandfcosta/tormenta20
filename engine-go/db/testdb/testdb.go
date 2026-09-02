// Package testdb entrega um SQLite migrado por teste, copiando um molde
// construído uma vez (ALE-260, extraído do `api` na ALE-281).
//
// # O problema que ele resolve
//
// Abrir um banco novo por teste faz o `db.Open` rodar todas as migrações nele.
// São milhares de migrações com `fsync` numa corrida, e o custo de um `fsync` é
// o do dispositivo onde o `TMPDIR` cai. Medido nesta máquina:
//
//	                        tmpfs        disco girante
//	migrar do zero .....    7,1 ms          2.102 ms
//	copiar o molde .....      ~0,1 ms           0,076 ms
//	reabrir já migrado .      ~1 ms             1 ms
//
// No prato girante cada migração custava ~49 ms — a rotação do disco, uma por
// `fsync` —, e a suíte do `api/` levava 15m01s com 10 s de CPU: 99% de espera. O
// sintoma mente, porque aparece como "os testes estão lentos", que é a conclusão
// que faz alguém cortar teste em vez de consertar a bancada.
//
// # Por que o molde e não uma variável de ambiente
//
// Exportar `TMPDIR` para um tmpfs funciona — mas só para quem lembrar, e só na
// máquina de quem lembrou. Quando isto foi medido havia uma sessão vizinha
// rodando com o `TMPDIR` no disco girante sem saber que pagava quinze minutos
// por corrida. O molde tira o disco da conta em vez de pedir que alguém escolha
// o disco certo: o que a bancada não deixa acontecer ninguém precisa lembrar de
// evitar.
//
// # Por que ele virou PACOTE
//
// Ele nasceu como `api/bancada_test.go`, e arquivo `_test.go` não exporta nada
// para fora do pacote. Com o `api` se dividindo em um pacote por cena (ALE-278),
// a primeira cena a sair ficaria sem molde, sem banco migrado e sem catálogo — e
// a saída fácil seria ela escrever a própria bancada com um catálogo vazio, que
// é o defeito que o guia do pacote documenta DUAS vezes: catálogo vazio no
// fixture é validação desligada em silêncio, com o guarda verde afirmando o
// contrário do que mede.
//
// # O que ele NÃO faz
//
// Ele não monta servidor nem semeia dado nenhum. Isso é de quem tem o tipo do
// servidor, e pôr aqui obrigaria este pacote a importar o `api` — que importaria
// este de volta nos testes dele, que é um ciclo. O molde é a parte cara e a
// parte compartilhável; o resto é de cada pacote.
package testdb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"t20engine/db"
)

var molde string

// Run monta o molde, roda a suíte do pacote e limpa. Cada pacote de teste que
// usa o `Fresh` declara:
//
//	func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
//
// O molde é construído UMA vez por binário de teste, que é uma vez por pacote.
// Em tmpfs isso é ~7 ms, então dividir o `api` em quinze pacotes custa ~105 ms
// na corrida inteira — a conta foi feita antes de dividir, e não depois.
func Run(m *testing.M) int {
	dir, err := os.MkdirTemp("", "t20-molde-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "criar o molde do banco: %v\n", err)
		return 1
	}
	molde = filepath.Join(dir, "molde.db")
	base, err := db.Open(molde)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrar o molde do banco: %v\n", err)
		return 1
	}
	// Fechar ANTES de qualquer cópia: o `Close` faz o checkpoint do WAL, e um
	// molde copiado com escrita pendente no `-wal` nasceria sem as tabelas que o
	// `assertSchema` exige — o teste falharia dizendo "falta tabela", que manda
	// procurar defeito na migração e não na cópia.
	_ = base.Close()

	codigo := m.Run()
	_ = os.RemoveAll(dir)
	return codigo
}

// Fresh devolve o caminho de um banco migrado e VIRGEM, copiado do molde. O
// arquivo vive no `t.TempDir()`, então o próprio `testing` o apaga.
//
// O `db.Open` do chamador continua sendo o de produção, com o mesmo
// `assertSchema` (ALE-154) — o goose encontra a última versão aplicada e não tem
// o que fazer. É isso que mantém a bancada honesta: o atalho é a CÓPIA, não uma
// segunda forma de abrir banco.
func Fresh(t *testing.T) string {
	t.Helper()
	if molde == "" {
		t.Fatal("o molde não existe: este pacote de teste precisa de\n" +
			"\tfunc TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }\n" +
			"Sem ele cada teste migraria do zero, que é o que a ALE-260 tirou da conta.")
	}
	destino := filepath.Join(t.TempDir(), "test.db")
	if err := copyFile(molde, destino); err != nil {
		t.Fatalf("copiar o molde do banco para %q: %v", destino, err)
	}
	return destino
}

func copyFile(de, para string) error {
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
