package search

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// A BUSCA NÃO IMPORTA NADA DO PROJETO.
//
// Lista de permitidos VAZIA, e ela não é aspiração: este pacote importa só
// `strings`, `unicode` e a normalização de acento.
//
// Ele existe para APAGAR UMA CÓPIA. O `Fold` — que desacentua para
// comparação — morava em dois lugares porque o `book` precisava dele e não podia
// importar o `api`; a segunda cópia foi escrita errada e a classe deixou de
// ligar a perícia que treina, sem erro nenhum.
//
// A cópia só some para sempre enquanto ESTE pacote puder ser importado por
// qualquer um. No dia em que ele alcançar catálogo, banco ou HTTP, o próximo que
// precisar do `Fold` vai copiar de novo — e a próxima cópia vai estar errada de
// outro jeito.
func TestSearchImportsNothing(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ler o pacote: %v", err)
	}

	set := token.NewFileSet()
	visited := 0
	for _, entry := range files {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		visited++
		file, err := parser.ParseFile(set, name, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("ler %s: %v", name, err)
		}
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			if !strings.HasPrefix(path, "t20engine/") {
				continue
			}
			t.Errorf("%s importa %q — a busca é FOLHA.\n"+
				"Ela existe para qualquer um poder importá-la sem herdar nada; com um\n"+
				"import daqui, quem precisar dela e não puder pagar %q vai COPIAR — e foi\n"+
				"uma cópia dessas que quebrou o desacento em silêncio.",
				name, path, path)
		}
	}

	if visited == 0 {
		t.Fatal("nenhum arquivo .go visitado — o guarda ficou cego")
	}
}

// O FOLD DESACENTUA, e este caso existe porque a cópia dele já não desacentuou.
//
// Ele é o irmão do `book.TestTheAddressKeyDropsAccents`: aquele prende o efeito
// no endereço, este prende a função. Duas camadas, e é deliberado — foi
// exatamente aqui que a cópia divergiu do original.
func TestFoldDropsAccentsAndCase(t *testing.T) {
	cases := map[string]string{
		"Atuação": "atuacao",
		"Anão":    "anao",
		"ÉBANO":   "ebano",
		"luta":    "luta",
	}
	for entry, want := range cases {
		if obtained := Fold(entry); obtained != want {
			t.Errorf("Fold(%q) = %q, esperado %q", entry, obtained, want)
		}
	}
}
