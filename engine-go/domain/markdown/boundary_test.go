package markdown

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// O MARKDOWN NÃO ALCANÇA NADA (ALE-278).
//
// A lista de permitidos é VAZIA, como a do `search` e a do `creature`, e aqui
// isso não é rigor decorativo: este pacote existe porque a Mesa vai sair do
// `api` e não pode levar 266 linhas de parser junto nem deixá-las para trás.
//
// **O que a lista vazia protege é a alternativa que teria acontecido.** A regra
// que o `search` deixou escrita é a mesma: função pura hospedada em pacote
// grande vira CÓPIA na mão de quem não pode importar o pacote — e a cópia do
// `Fold` para o `book` foi escrita errada, chamando a função que só faz
// `ToLower`, o que quebrou o acento sem erro, sem panic e sem log.
//
// Aqui o risco é maior que uma letra: este markdown é uma GRAMÁTICA com
// decisões que custaram caro (a ALE-122 consertou o parágrafo que junta linhas
// soltas, que é o comportamento PADRÃO de um CommonMark). Uma segunda
// implementação não erraria por descuido — ela erraria por ser um parser
// diferente, e o oráculo versionado ao lado só mede ESTA.
var permitidos = map[string]bool{}

func TestTheMarkdownReachesNothing(t *testing.T) {
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
			t.Errorf("%s importa %q — este markdown é uma FUNÇÃO PURA de texto para\n"+
				"árvore, e a lista de permitidos é vazia de propósito.\n"+
				"No dia em que ele alcançar catálogo, banco ou HTTP, o próximo que\n"+
				"precisar dele de um lugar que não pode importá-lo vai escrever uma\n"+
				"SEGUNDA gramática — e o oráculo versionado ao lado só mede esta.",
				nome, caminho)
		}
	}

	// O DENOMINADOR: um diretório não lido e uma lista de reprovados vazia se
	// parecem no terminal.
	if visitados < 2 {
		t.Fatalf("o guarda visitou só %d arquivos `.go` — ele está medindo o "+
			"diretório errado", visitados)
	}
}
