package convention

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoSceneCommandUsesTheDefaultTab — a varredura da convenção (ALE-205).
//
// A convenção que esta issue criou é uma frase: **o comando age na aba de quem
// clicou**, e é ela que dispensou pôr o id do tabuleiro em vinte rotas. O
// compilador obriga a passar o parâmetro; ele não obriga a passar o CERTO — um
// `aAbaPadrao` num arquivo do app compila, roda, e pinta a taverna que a mesa
// está vendo enquanto o mestre olha a cripta. Sem estourar nada.
//
// Por isso ele varre em vez de conferir um caso: uma revisão nomeia um arquivo,
// e a correção é TODO arquivo com a mesma forma. A `aAbaPadrao` é da tela ANTIGA,
// que não tem abas: ela morava em dois arquivos do `serve/api`, e os dois foram
// apagados — um na ALE-277, o outro na ALE-344, quando o deslocamento das peças
// desceu para a cena. O que sobrou dela é a constante em
// `table_live_publish.go`, e lá ela mudou de significado — a Mesa em Datastar TEM
// abas, então "a padrão" virou o quadro que quem fechou uma aba não escolheu.
// **Ele veio para cá na ALE-278**, e a razão é a que o guarda do foco já tinha
// pago: ele varria `piloto_*.go` do PRÓPRIO diretório, o que era a fonte inteira
// enquanto todas as cenas eram um pacote só. Com a Mesa virando `web/table` o
// glob deixou de casar com qualquer coisa — e desta vez o CONTROLE existia, então
// ele falhou ALTO em vez de passar verde sobre zero arquivos.
func TestNoSceneCommandUsesTheDefaultTab(t *testing.T) {
	var files []string
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	if err := filepath.WalkDir(filepath.Dir(root), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		files = append(files, path)
		return nil
	}); err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}
	// CONTROLE: sem ele, um glob que não casa com nada passaria VERDE — o teste
	// diria "nenhum arquivo viola" sobre uma varredura que não visitou ninguém,
	// que é a forma de silêncio que esta casa já pagou caro.
	if len(files) < 200 {
		t.Fatalf("a varredura achou só %d arquivos `.go`: ela está caminhando a árvore errada", len(files))
	}
	visited := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_templ.go") {
			continue
		}
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ler %s: %v", path, err)
		}
		visited++
		if strings.Contains(string(source), "aAbaPadrao") {
			t.Errorf("%s usa aAbaPadrao: o comando da cena age na aba de QUEM CLICOU (c.TabuleiroID), "+
				"e a padrão é da tela antiga — este gesto mexeria na cena que outra pessoa está olhando", path)
		}
	}
	if visited < 200 {
		t.Fatalf("a varredura leu só %d arquivos de produção", visited)
	}
}
