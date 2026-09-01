package tabuleiro

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A FRONTEIRA DO TABULEIRO (ALE-254).
//
// Terceiro guarda desta família, e a regra é sempre a mesma: um bounded context
// vale pelo que o compilador IMPEDE. A lista aqui admite o REGIME porque o
// tabuleiro só existe DENTRO de uma sessão ao vivo — as peças vêm da fila, e a
// regra de numerar repetidos é a mesma das duas superfícies para elas não
// numerarem diferente (ALE-192). A direção contrária seria o erro: se o regime
// precisasse do tabuleiro, o ciclo apareceria e diria que a fronteira está no
// lugar errado.
//
// O que este guarda protege de verdade é o dia em que o tabuleiro precisar de
// algo da FICHA. Ele já precisa: o deslocamento da peça sai da ficha computada,
// e hoje isso atravessa o `Server` porque o handler ainda mora em `api/`. Quando
// o contexto `ficha` nascer, a tentação será importá-lo daqui — e a resposta
// certa é uma PORTA declarada neste pacote, como `aovivo.VitaisDaFicha` fez.
// A mensagem de falha diz isso, porque uma lista que cresce em silêncio é uma
// fronteira que deixou de existir sem ninguém notar.

// permitidos é o que o tabuleiro pode importar de dentro do projeto.
var permitidos = map[string]bool{
	// O regime: a fila é de onde as peças vêm, e a numeração de repetidos é
	// compartilhada de propósito.
	"t20engine/aovivo": true,
	// Não é domínio nenhum; a direção é de mão única e o guarda de lá garante.
	"t20engine/plataforma": true,
	// A persistência do mapa é do tabuleiro: `session_boards` é tabela dele.
	"t20engine/db/sqlcgen": true,
	// As regras de alcance e deslocamento do LIVRO já vivem em `engine/board_*`,
	// e é de lá que elas devem vir — reimplementá-las aqui seria a segunda
	// definição que o oráculo existe para impedir.
	"t20engine/engine": true,
	// O VOCABULÁRIO DA MESA é shared kernel, e não um contexto (ALE-279).
	//
	// Ele entra nesta lista sabendo do aviso que está escrito acima — que
	// acrescentar import aqui transforma a porta em enfeite —, e a diferença
	// está no que ele PODE conter. Uma porta não resolve este caso: um
	// barramento TIPADO exige que os dois lados falem do mesmo tipo, e
	// interface declarada aqui só casaria com um tipo daqui, o que devolveria
	// um vocabulário por contexto e o problema inteiro.
	//
	// O que impede a porta dos fundos ("importe `events` para chegar na ficha")
	// não é confiança: é o `TestOVocabularioNaoImportaNinguem`, que recusa
	// QUALQUER import do projeto dentro de `events/`. Enquanto ele for folha,
	// depender dele não cria fronteira errada nenhuma — que é exatamente a
	// justificativa da `plataforma`, por outro caminho.
	"t20engine/events": true,
}

func TestOTabuleiroNaoConheceAFicha(t *testing.T) {
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
			if !strings.HasPrefix(caminho, "t20engine/") || permitidos[caminho] {
				continue
			}
			t.Errorf("%s importa %q — o tabuleiro não conhece esse contexto.\n"+
				"Se ele precisa de algo de lá, DECLARE UMA PORTA aqui e receba quem\n"+
				"a cumpre por parâmetro. Acrescentar o import à lista acima é\n"+
				"apagar a fronteira sem que ninguém perceba.",
				nome, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
