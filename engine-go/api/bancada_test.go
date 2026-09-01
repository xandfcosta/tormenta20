package api

import (
	"os"
	"testing"

	"t20engine/db/testdb"
)

// A BANCADA desta suíte mora em `db/testdb` desde a ALE-281.
//
// Ela era este arquivo inteiro — o molde migrado uma vez e copiado por teste
// (ALE-260) —, e mudou de casa por uma razão de forma e não de conteúdo: arquivo
// `_test.go` não exporta nada para fora do pacote, e o `api` está se dividindo em
// um pacote por cena (ALE-278). A primeira cena a sair encontraria a bancada
// inalcançável e escreveria a própria, que é como um fixture nasce com catálogo
// vazio e desliga validação em silêncio.
//
// O que ficou aqui é o que é DESTE pacote: o `TestMain` de uma linha e o atalho
// `bancoDeTeste`, que os quarenta e seis arquivos de teste chamam.

func TestMain(m *testing.M) { os.Exit(testdb.Run(m)) }

// bancoDeTeste devolve um banco migrado e VIRGEM.
func bancoDeTeste(t *testing.T) string {
	t.Helper()
	return testdb.Fresh(t)
}
