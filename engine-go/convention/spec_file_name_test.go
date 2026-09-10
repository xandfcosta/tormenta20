package convention

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// NOME DE ARQUIVO DE SPEC É INGLÊS, E A DESCRIÇÃO DE DENTRO É PORTUGUÊS (ALE-301).
//
// O par é o que confunde, e por isso ele é decisão registrada no `CLAUDE.md` em
// vez de convenção implícita: o mesmo arquivo leva as duas línguas —
// `piloto-board.spec.ts` contendo `test('arrastar a peça propõe a parada')`. O
// arquivo é identificador, a descrição é frase que uma pessoa lê no relatório.
//
// Este guarda cobra SÓ a metade do arquivo. A descrição não tem guarda e não
// deve ter: "está em português" não é mecanizável sem um dicionário, e um
// medidor que chutasse ali reprovaria a frase certa — foi o que aconteceu na
// ALE-300, quando contar partículas ambíguas acusou trezentos nomes de teste em
// inglês correto.
//
// FICA DE FORA o que o glossário chama de nome PRÓPRIO. `piloto` é o pacote do
// sistema de desenho e `grimorio` é a identidade visual ("Grimório de Arton") —
// os dois são o nome da coisa, não tradução pendente, como `tormenta` e `tibar`.
var properNounsInFileNames = map[string]bool{"piloto": true, "grimorio": true}

func TestNoSpecFileIsNamedInPortuguese(t *testing.T) {
	// A RAIZ é o `e2e/`, e o recorte é honesto: o `engine-go` ainda tem nome de
	// arquivo em português (`pericias.json`, `paginas-do-livro.py`), e
	// varrê-los é outra fatia da ALE-301. Um guarda que mede o que a fatia
	// arrumou é melhor que um guarda desligado por medir o que ela não arrumou.
	raiz := filepath.Join("..", "..", "e2e", "tests")
	medidos := 0
	err := filepath.WalkDir(raiz, func(caminho string, entrada fs.DirEntry, err error) error {
		if err != nil || entrada.IsDir() {
			return err
		}
		nome := entrada.Name()
		if !strings.HasSuffix(nome, ".ts") {
			return nil
		}
		medidos++
		base := strings.SplitN(nome, ".", 2)[0]
		for _, seg := range strings.Split(base, "-") {
			seg = strings.ToLower(seg)
			if properNounsInFileNames[seg] || !portugueseWords[seg] {
				continue
			}
			t.Errorf("o arquivo %s tem %q no nome, que é português.\n"+
				"Nome de arquivo é IDENTIFICADOR e sai em inglês (CLAUDE.md, \"Idioma\"). "+
				"A descrição de dentro do `test('…')` é que fica em português.",
				filepath.Join("e2e/tests", strings.TrimPrefix(caminho, raiz+string(filepath.Separator))), seg)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("varrer %s: %v", raiz, err)
	}
	// O DENOMINADOR: uma lista vazia e uma raiz errada se parecem no terminal.
	if medidos < 25 {
		t.Fatalf("a varredura leu %d arquivos em %s — a raiz é o primeiro suspeito", medidos, raiz)
	}
}
