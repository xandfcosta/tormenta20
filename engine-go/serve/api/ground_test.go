package api

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"t20engine/domain/board"
)

// O guarda do CHÃO do lugar: a lista que a tela OFERECE e o CSS que a PINTA têm
// de andar juntas.
//
// Uma cópia à mão do conjunto no templ é como nasce a opção escolhível que o
// navegador desenha em branco. O defeito não estoura: ele pinta o chão errado,
// em silêncio, que é a marca desta família.
//
// Amostragem e não enumeração: o guarda percorre a LISTA, então o chão que
// alguém acrescentar amanhã já nasce medido — não há uma entrada por caso aqui
// para alguém esquecer de escrever.
func TestEveryOfferedGroundCanBePainted(t *testing.T) {
	css, err := os.ReadFile("../web/assets/app.src.css")
	if err != nil {
		t.Fatalf("ler o CSS da casa: %v", err)
	}
	sheet := string(css)

	// O CONTROLE: a folha tem a família que vamos procurar. Sem ele, um caminho
	// errado ou um arquivo renomeado daria "nenhum chão encontrado" — que se
	// parece com "todos faltando" e passaria verde se a asserção fosse ao
	// contrário.
	if !strings.Contains(sheet, ".ground-") {
		t.Fatalf("o CSS da casa não tem nenhuma classe .ground-* — o guarda está lendo o arquivo errado (%d bytes)", len(sheet))
	}

	for _, floor := range board.PlaceGrounds {
		if !strings.Contains(sheet, ".ground-"+floor.ID) {
			t.Errorf("o chão %q (%s) é oferecido na tela e o CSS não sabe pintá-lo: falta .ground-%s",
				floor.ID, floor.Label, floor.ID)
		}
		if floor.Label == "" {
			t.Errorf("o chão %q não tem rótulo para o mestre ler", floor.ID)
		}
	}
}

// A FOLHA E A PASTA DE FONTES DIZEM A MESMA COISA, NAS DUAS DIREÇÕES.
//
// A folha pede `/fonts/…` por caminho absoluto, e um pedido sem arquivo não dá
// erro em lugar nenhum: o navegador cai na face do sistema e a tela inteira
// muda de largura — o defeito que a ALE-362 mediu, com a bancada e a CI
// medindo telas diferentes.
//
// A direção contrária é igualmente muda: um `.woff2` embutido que a folha não
// pede é peso morto dentro do binário, e ninguém o encontra procurando.
//
// Aqui morava uma contagem à mão (`len(fontes) != 2`), e ela reprovou na
// primeira fatia que acrescentou uma face. Número escrito à mão sobre família
// que cresce envelhece sozinho; o denominador agora é a própria folha.
func TestTheStylesheetFontsExist(t *testing.T) {
	sheet, err := os.ReadFile("../web/assets/static/app.css")
	if err != nil {
		t.Fatalf("ler a folha compilada: %v", err)
	}
	requested := map[string]bool{}
	for _, found := range fontURL.FindAllStringSubmatch(string(sheet), -1) {
		requested[found[1]] = true
	}
	if len(requested) == 0 {
		t.Fatalf("a folha compilada não pede fonte nenhuma — o guarda está lendo o arquivo errado (%d bytes)", len(sheet))
	}

	const folder = "../web/assets/static/fonts"
	for file := range requested {
		info, err := os.Stat(filepath.Join(folder, file))
		if err != nil {
			t.Errorf("a folha pede /fonts/%s e o arquivo não está embutido: o navegador cai na face do sistema, e a tela muda de largura sem ninguém ligar uma coisa à outra", file)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("/fonts/%s está vazia: o navegador ignora a fonte e cai na do sistema", file)
		}
	}

	embedded, err := os.ReadDir(folder)
	if err != nil {
		t.Fatalf("ler as fontes embutidas: %v", err)
	}
	for _, f := range embedded {
		if !strings.HasSuffix(f.Name(), ".woff2") || requested[f.Name()] {
			continue
		}
		t.Errorf("%s está embutida e a folha não a pede: peso morto no binário", f.Name())
	}
	t.Logf("%d fontes pedidas pela folha, %d arquivos na pasta", len(requested), len(embedded))
}

var fontURL = regexp.MustCompile(`url\(["\']?/fonts/([^"\')]+)`)
