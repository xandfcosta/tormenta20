// Package testdb entrega um SQLite migrado por teste, copiando um molde
// construído uma vez.
//
// # O problema que ele resolve
//
// Abrir um banco novo por teste faz o `db.Open` rodar todas as migrações nele, e
// o custo de cada `fsync` é o do dispositivo onde o `TMPDIR` cai. Em disco
// girante isso são ~49 ms POR MIGRAÇÃO, e a suíte inteira vira 99% de espera —
// um sintoma que mente, porque aparece como "os testes estão lentos" e faz
// alguém cortar teste em vez de consertar a bancada.
//
// # Por que o molde e não uma variável de ambiente
//
// Exportar `TMPDIR` para um tmpfs funciona — mas só para quem lembrar, e só na
// máquina de quem lembrou. O molde tira o disco da conta em vez de pedir que
// alguém escolha o disco certo.
//
// # Por que ele é PACOTE e não um `_test.go`
//
// Arquivo `_test.go` não exporta nada para fora do pacote, e cada cena nova
// ficaria sem molde, sem banco migrado e sem catálogo — com a saída fácil sendo
// escrever a própria bancada com um catálogo VAZIO, que é validação desligada em
// silêncio, com o guarda verde afirmando o contrário do que mede.
//
// # O que ele NÃO faz
//
// Ele não monta servidor nem semeia dado nenhum. Isso é de quem tem o tipo do
// servidor, e pôr aqui obrigaria este pacote a importar o `api` — que importaria
// este de volta nos testes dele, que é um ciclo.
package testdb

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"t20engine/infra/db"
)

var molde string

// Run monta o molde, roda a suíte do pacote e limpa. Cada pacote de teste que
// usa o `Fresh` declara:
//
//	func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }
//
// O molde é construído UMA vez por binário de teste, que é uma vez por pacote:
// em tmpfs isso é alguns milissegundos, então dividir um pacote grande em muitos
// menores não muda a conta.
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
// `assertSchema` — o goose encontra a última versão aplicada e não tem o que
// fazer. É isso que mantém a bancada honesta: o atalho é a CÓPIA, não uma
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

// copyFile copia o molde e CONFERE que ele chegou inteiro.
//
// Uma cópia truncada se comporta EXATAMENTE como um banco sem as tabelas, e o
// teste que a receber reprova dizendo `no such table` — que manda quem investiga
// caçar migração e não cópia. Daí as três defesas, cada uma pegando o que a
// anterior deixa passar:
//
//  1. O ERRO DO `Close` volta. Num `defer` que descarta, perde-se justamente a
//     falha que uma escrita com buffer só reporta no fechamento.
//  2. O `Sync` força os bytes ao disco ANTES de alguém abrir o arquivo. Sem ele,
//     `io.Copy` devolver nil só diz que os bytes saíram do processo.
//  3. O TAMANHO é conferido, e esta é a que não depende de o sistema de arquivos
//     reportar coisa alguma.
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
	if _, err := io.Copy(destino, origem); err != nil {
		_ = destino.Close()
		return fmt.Errorf("copiar %q para %q: %w", de, para, err)
	}
	if err := destino.Sync(); err != nil {
		_ = destino.Close()
		return fmt.Errorf("gravar %q no disco: %w", para, err)
	}
	// O `Close` FECHA e o erro dele VOLTA — não é higiene: num sistema de
	// arquivos com buffer, é aqui que a falha de escrita aparece.
	if err := destino.Close(); err != nil {
		return fmt.Errorf("fechar %q: %w", para, err)
	}
	return conferAcopia(de, para)
}

// conferAcopia recusa uma cópia que não tem o tamanho da origem.
//
// A mensagem carrega OS DOIS tamanhos porque quem a lê precisa saber se faltou
// um byte ou o arquivo inteiro — e porque ela é o que aparece no lugar de um
// `no such table` três camadas adiante.
func conferAcopia(de, para string) error {
	origem, err := os.Stat(de)
	if err != nil {
		return fmt.Errorf("medir o molde %q: %w", de, err)
	}
	copia, err := os.Stat(para)
	if err != nil {
		return fmt.Errorf("medir a cópia %q: %w", para, err)
	}
	if copia.Size() != origem.Size() {
		return fmt.Errorf(
			"a cópia do molde saiu incompleta: %q tem %d bytes e o molde %q tem %d. "+
				"Um SQLite truncado se comporta como um banco SEM AS TABELAS, e o teste que "+
				"o receber reprova dizendo `no such table` — que manda procurar defeito na "+
				"migração (ALE-268)",
			para, copia.Size(), de, origem.Size())
	}
	return nil
}
