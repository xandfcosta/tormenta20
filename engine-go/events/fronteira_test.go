package events

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O VOCABULÁRIO É FOLHA, e este guarda é o que faz isso valer (ALE-279).
//
// Ele é o irmão dos `fronteira_test.go` do `aovivo`, do `tabuleiro` e da
// `plataforma`, e existe por uma razão específica: `events` teve de ENTRAR na
// lista de permitidos dos outros dois, e a lista de lá vem com um aviso —
// *acrescentar o import à lista transforma a porta em enfeite*.
//
// O que impede isso de acontecer aqui não é a boa intenção de quem escreveu: é
// este teste. Enquanto `events` não importar NADA do projeto, depender dele não
// pode criar fronteira errada nenhuma, porque não há para onde a dependência
// vazar. No dia em que alguém importar a ficha aqui para "enriquecer o evento",
// os dois contextos passam a alcançar a ficha por tabela, e o guarda de lá
// continua verde — ele só olha os imports DELE.
//
// É a mesma forma da `plataforma`, por outro caminho: lá a lista de permitidos
// é vazia porque ela não é domínio nenhum; aqui ela é vazia porque o vocabulário
// é de todos e por isso não pode ser de ninguém.
//
// # O que PODE entrar aqui
//
// Tipo de evento, e o barramento. Nada que precise ler banco, catálogo ou regra:
// um evento carrega os identificadores do que aconteceu, e quem quiser saber
// mais vai ao store — que é o contrato escrito no `Publish`.
func TestVocabularyImportsNothing(t *testing.T) {
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
				continue
			}
			t.Errorf("%s importa %q — o vocabulário da mesa é FOLHA.\n"+
				"Ele está na lista de permitidos do `aovivo` e do `tabuleiro`\n"+
				"justamente porque não alcança nada; com um import daqui, os dois\n"+
				"contextos passam a alcançar %q de graça, e o guarda de lá não vê.",
				nome, caminho, caminho)
		}
	}

	// Sem isto, apagar o pacote deixaria o guarda VERDE — ausência lida como
	// aprovação, que é a família que o `CLAUDE.md` da raiz descreve.
	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}
