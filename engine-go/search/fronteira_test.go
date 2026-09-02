package search

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A BUSCA NÃO IMPORTA NADA DO PROJETO (ALE-278).
//
// Lista de permitidos VAZIA, como a do `creature`. Ela não é aspiração: o
// arquivo já importava só `strings`, `unicode` e a normalização de acento quando
// morava no `api`, e foi isso que provou que ele saía inteiro.
//
// # Por que este guarda importa mais que a média
//
// Este pacote existe para APAGAR UMA CÓPIA. O `Fold` — que desacentua para
// comparação — morava em dois lugares porque o `book` precisava dele e não podia
// importar o `api`; a segunda cópia foi escrita errada e a classe deixou de
// ligar a perícia que treina, sem erro nenhum.
//
// A cópia só some para sempre enquanto ESTE pacote puder ser importado por
// qualquer um. No dia em que ele alcançar catálogo, banco ou HTTP, o próximo que
// precisar do `Fold` vai copiar de novo — e a próxima cópia vai estar errada de
// outro jeito.
func TestSearchImportsNothing(t *testing.T) {
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
			t.Errorf("%s importa %q — a busca é FOLHA.\n"+
				"Ela existe para qualquer um poder importá-la sem herdar nada; com um\n"+
				"import daqui, quem precisar dela e não puder pagar %q vai COPIAR — e foi\n"+
				"uma cópia dessas que quebrou o desacento em silêncio.",
				nome, caminho, caminho)
		}
	}

	if visitados == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}

// O FOLD DESACENTUA, e este caso existe porque a cópia dele já não desacentuou.
//
// Ele é o irmão do `book.TestTheAddressKeyDropsAccents`: aquele prende o efeito
// no endereço, este prende a função. Duas camadas, e é deliberado — foi
// exatamente aqui que a cópia divergiu do original.
func TestFoldDropsAccentsAndCase(t *testing.T) {
	casos := map[string]string{
		"Atuação": "atuacao",
		"Anão":    "anao",
		"ÉBANO":   "ebano",
		"luta":    "luta",
	}
	for entrada, esperado := range casos {
		if obtido := Fold(entrada); obtido != esperado {
			t.Errorf("Fold(%q) = %q, esperado %q", entrada, obtido, esperado)
		}
	}
}
