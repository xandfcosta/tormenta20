package plataforma

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FRONTEIRA DA PLATAFORMA, e o único jeito de ela valer alguma coisa (ALE-254).
//
// A issue do trabalho diz a regra e ela merece ser repetida aqui: **um bounded
// context vale pelo que o compilador IMPEDE.** Se `plataforma` puder alcançar o
// domínio, a pasta é decoração — o pacote continuaria sendo o saco onde tudo
// cabe, só que com nome novo.
//
// O compilador já impede um ciclo: se `plataforma` importasse `api`, e `api`
// importa `plataforma`, o build quebra. Mas ele NÃO impede o caminho que
// realmente acontece na prática — alguém importar `engine`, `catalog` ou
// `db/sqlcgen` daqui, achando que "é só um tipinho". Isso compila, não é ciclo,
// e é exatamente como o `api/` chegou a 20 mil linhas.
//
// Este guarda fecha esse caminho. Ele é barato porque só lê imports, e é
// deliberadamente uma LISTA DO QUE PODE em vez de uma lista do que não pode:
// pacote novo do domínio nasce barrado sem ninguém precisar lembrar de o
// acrescentar, que é a diferença entre amostragem e enumeração.
//
// A linha do glossário diz o critério em uma frase: se um conceito do jogo
// entrar aqui, a fronteira está errada. Este teste é essa frase, executável.

// permitidos é o que a plataforma pode importar: biblioteca padrão e nada mais.
// A entrada vazia representa a stdlib inteira — ela não tem barra no caminho.
var permitidos = map[string]bool{}

func TestThePlatformDoesNotReachTheDomain(t *testing.T) {
	arquivos, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	conjunto := token.NewFileSet()
	visitados := 0
	for _, entrada := range arquivos {
		nome := entrada.Name()
		if !strings.HasSuffix(nome, ".go") {
			continue
		}
		visitados++
		arquivo, err := parser.ParseFile(conjunto, nome, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", nome, err)
		}
		for _, imp := range arquivo.Imports {
			caminho := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(caminho, "t20engine/") {
				continue // stdlib: sempre pode
			}
			if permitidos[caminho] {
				continue
			}
			t.Errorf("%s importa %q — a plataforma não é domínio nenhum.\n"+
				"Se este import é mesmo necessário, a fronteira está no lugar errado:\n"+
				"mova o CÓDIGO para o contexto dono do conceito, não o import para a lista.",
				nome, caminho)
		}
	}

	// Sem isto, apagar o pacote inteiro deixaria o guarda VERDE — ausência lida
	// como aprovação, que é a família de defeito que o `CLAUDE.md` da raiz
	// descreve. Um guarda que não visita nada não mede nada.
	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
