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

// Guarda de CASCATA, e ele prende a COLOCAÇÃO porque é ela o defeito.
//
// Dentro de `@layer components` a regra NÃO VALE: o elemento carrega a
// utilitária `flex` do Tailwind, que mora numa camada POSTERIOR, e camada
// posterior ganha de anterior independentemente de especificidade. O efeito é o
// pior possível — numa tela larga aparecem OS DOIS, o painel lateral com a ficha
// e o modal por cima dela.
//
// O que este guarda NÃO faz: ele não resolve cascata, ele lê TEXTO. Cascata de
// verdade só um navegador resolve, e um e2e para uma linha seria caro. O que ele
// pega é a regressão exata e provável — alguém arrastar a regra de volta para
// dentro do `@layer` numa arrumação, achando que camada é organização.
func TestTheRuleThatHidesTheDialogStaysOutOfTheLayer(t *testing.T) {
	sheet, err := os.ReadFile("../web/assets/app.src.css")
	if err != nil {
		t.Fatalf("ler o CSS da casa: %v", err)
	}
	css := string(sheet)

	pos := strings.Index(css, ".table-sheet-in-dialog")
	if pos < 0 {
		t.Fatalf("a regra sumiu da folha — o guarda está lendo o arquivo errado (%d bytes)", len(css))
	}

	// Profundidade de blocos ABERTOS na altura da regra. Um nível é o
	// `@container`, que é legítimo e necessário; dois ou mais significa que há
	// um `@layer` (ou outro bloco) por fora, e lá a regra perde.
	depth := 0
	for _, c := range css[:pos] {
		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
		}
	}
	if depth > 1 {
		t.Errorf("a regra que esconde a ficha em diálogo está %d blocos aninhada, e só o `@container` é esperado: dentro de `@layer` ela perde para o `flex` do Tailwind e a tela larga mostra o painel E o modal", depth)
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
