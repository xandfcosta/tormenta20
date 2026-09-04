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
// `aAbaPadrao` num arquivo do piloto compila, roda, e pinta a taverna que a mesa
// está vendo enquanto o mestre olha a cripta. Sem estourar nada.
//
// Por isso ele varre em vez de conferir um caso: uma revisão nomeia um arquivo,
// e a correção é TODO arquivo com a mesma forma. A `aAbaPadrao` é da tela ANTIGA,
// que não tem abas: ela morava em `board_commands.go`, apagado na ALE-277, e em
// `board_rules.go`, que ficou. O que sobrou dela é a constante em
// `live_publish.go`, e lá ela mudou de significado — a Mesa em Datastar TEM
// abas, então "a padrão" virou o quadro que quem fechou uma aba não escolheu.
// **Ele veio para cá na ALE-278**, e a razão é a que o guarda do foco já tinha
// pago: ele varria `piloto_*.go` do PRÓPRIO diretório, o que era a fonte inteira
// enquanto todas as cenas eram um pacote só. Com a Mesa virando `web/table` o
// glob deixou de casar com qualquer coisa — e desta vez o CONTROLE existia, então
// ele falhou ALTO em vez de passar verde sobre zero arquivos.
func TestNoSceneCommandUsesTheDefaultTab(t *testing.T) {
	var arquivos []string
	raiz, err := os.Getwd()
	if err != nil {
		t.Fatalf("achar a raiz: %v", err)
	}
	if err := filepath.WalkDir(filepath.Dir(raiz), func(caminho string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(caminho, ".go") {
			return err
		}
		arquivos = append(arquivos, caminho)
		return nil
	}); err != nil {
		t.Fatalf("caminhar a árvore: %v", err)
	}
	// CONTROLE: sem ele, um glob que não casa com nada passaria VERDE — o teste
	// diria "nenhum arquivo viola" sobre uma varredura que não visitou ninguém,
	// que é a forma de silêncio que esta casa já pagou caro.
	if len(arquivos) < 200 {
		t.Fatalf("a varredura achou só %d arquivos `.go`: ela está caminhando a árvore errada", len(arquivos))
	}
	visitados := 0
	for _, caminho := range arquivos {
		if strings.HasSuffix(caminho, "_test.go") || strings.HasSuffix(caminho, "_templ.go") {
			continue
		}
		fonte, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatalf("ler %s: %v", caminho, err)
		}
		visitados++
		if strings.Contains(string(fonte), "aAbaPadrao") {
			t.Errorf("%s usa aAbaPadrao: o comando do piloto age na aba de QUEM CLICOU (c.TabuleiroID), "+
				"e a padrão é da tela antiga — este gesto mexeria na cena que outra pessoa está olhando", caminho)
		}
	}
	if visitados < 200 {
		t.Fatalf("a varredura leu só %d arquivos de produção", visitados)
	}
}
