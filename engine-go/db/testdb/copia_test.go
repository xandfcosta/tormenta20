package testdb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A CÓPIA DO MOLDE FALHA ALTO, ONDE ELA ACONTECE (ALE-268).
//
// # O sintoma apontava para o lugar errado
//
// O CI reprovou uma vez com `no such table: session_boards` acompanhado de
// `disk I/O error (1802)`. "No such table" é a frase de um banco NÃO MIGRADO, e
// a migração está lá e é aplicada — quem procurar por migração não vai achar
// nada. Medido em 2026-09-08: uma ocorrência em 100 corridas de CI, contra a
// primeira estimativa de 1 em 13.
//
// # O que a cópia escondia
//
// Duas coisas, e as duas são silenciosas:
//
//   - o erro do `Close` era DESCARTADO (`defer func() { _ = destino.Close() }()`),
//     e é no `Close` que uma escrita com buffer reporta falha;
//   - não havia `Sync`, então `io.Copy` devolver nil não significa byte nenhum
//     no disco.
//
// O resultado de uma cópia parcial é um arquivo SQLite truncado, que se comporta
// EXATAMENTE como um banco sem a tabela. O `PRAGMA synchronous=OFF` do banco de
// teste — decisão certa para velocidade — remove justamente a barreira que
// transformaria isso num erro mais honesto.
//
// Este caso prende a terceira defesa, que é a que não depende do sistema de
// arquivos reportar coisa alguma: conferir o TAMANHO do que foi copiado. Uma
// cópia byte a byte de um arquivo tem o tamanho dele, e nada mais precisa ser
// verdade para isso valer.
func TestTheBenchRefusesATruncatedCopy(t *testing.T) {
	dir := t.TempDir()
	origem := filepath.Join(dir, "molde.db")
	if err := os.WriteFile(origem, make([]byte, 4096), 0o600); err != nil {
		t.Fatalf("montar o molde de mentira: %v", err)
	}
	destino := filepath.Join(dir, "truncado.db")
	if err := os.WriteFile(destino, make([]byte, 1024), 0o600); err != nil {
		t.Fatalf("montar a cópia truncada: %v", err)
	}

	err := conferAcopia(origem, destino)
	if err == nil {
		t.Fatal("a cópia truncada passou: o teste que a receber vai reprovar dizendo " +
			"`no such table`, e quem investigar vai procurar defeito na migração")
	}
	// A MENSAGEM carrega os dois tamanhos, que é o que separa "a cópia falhou" de
	// "a cópia falhou e olha o quanto": quem lê precisa saber se faltou um byte
	// ou faltou o arquivo inteiro.
	for _, pedaco := range []string{"1024", "4096"} {
		if !strings.Contains(err.Error(), pedaco) {
			t.Errorf("a mensagem não diz os tamanhos (falta %q): %v", pedaco, err)
		}
	}

	// O CONTROLE POSITIVO: uma cópia inteira passa. Sem ele, uma verificação que
	// recusa TUDO passaria neste arquivo — e é o jeito mais fácil de errar.
	inteira := filepath.Join(dir, "inteira.db")
	if err := os.WriteFile(inteira, make([]byte, 4096), 0o600); err != nil {
		t.Fatalf("montar a cópia inteira: %v", err)
	}
	if err := conferAcopia(origem, inteira); err != nil {
		t.Errorf("a cópia inteira foi recusada: %v", err)
	}
}

// E O CAMINHO DE VERDADE continua entregando um banco que abre — sem isto, as
// asserções acima provariam a verificação e nada sobre o `Fresh`.
func TestTheBenchStillHandsOutAWorkingDatabase(t *testing.T) {
	caminho := Fresh(t)
	info, err := os.Stat(caminho)
	if err != nil {
		t.Fatalf("o banco copiado não existe: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("o banco copiado está vazio")
	}
	molde, err := os.Stat(molde)
	if err != nil {
		t.Fatalf("o molde sumiu: %v", err)
	}
	if info.Size() != molde.Size() {
		t.Errorf("a cópia tem %d bytes e o molde tem %d", info.Size(), molde.Size())
	}
}
