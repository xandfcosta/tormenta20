package api

import (
	"os"
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
	css, err := os.ReadFile("assets/app.src.css")
	if err != nil {
		t.Fatalf("ler o CSS da casa: %v", err)
	}
	folha := string(css)

	// O CONTROLE: a folha tem a família que vamos procurar. Sem ele, um caminho
	// errado ou um arquivo renomeado daria "nenhum chão encontrado" — que se
	// parece com "todos faltando" e passaria verde se a asserção fosse ao
	// contrário.
	if !strings.Contains(folha, ".ground-") {
		t.Fatalf("o CSS da casa não tem nenhuma classe .ground-* — o guarda está lendo o arquivo errado (%d bytes)", len(folha))
	}

	for _, chao := range board.PlaceGrounds {
		if !strings.Contains(folha, ".ground-"+chao.ID) {
			t.Errorf("o chão %q (%s) é oferecido na tela e o CSS não sabe pintá-lo: falta .ground-%s",
				chao.ID, chao.Rotulo, chao.ID)
		}
		if chao.Rotulo == "" {
			t.Errorf("o chão %q não tem rótulo para o mestre ler", chao.ID)
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
	folha, err := os.ReadFile("assets/app.src.css")
	if err != nil {
		t.Fatalf("ler o CSS da casa: %v", err)
	}
	css := string(folha)

	pos := strings.Index(css, ".table-sheet-in-dialog")
	if pos < 0 {
		t.Fatalf("a regra sumiu da folha — o guarda está lendo o arquivo errado (%d bytes)", len(css))
	}

	// Profundidade de blocos ABERTOS na altura da regra. Um nível é o
	// `@container`, que é legítimo e necessário; dois ou mais significa que há
	// um `@layer` (ou outro bloco) por fora, e lá a regra perde.
	profundidade := 0
	for _, c := range css[:pos] {
		if c == '{' {
			profundidade++
		} else if c == '}' {
			profundidade--
		}
	}
	if profundidade > 1 {
		t.Errorf("a regra que esconde a ficha em diálogo está %d blocos aninhada, e só o `@container` é esperado: dentro de `@layer` ela perde para o `flex` do Tailwind e a tela larga mostra o painel E o modal", profundidade)
	}
}

// Estas SÃO as fontes: não há segundo lado para comparar. O que se prende é a
// PRESENÇA — a folha pede `/fonts/…` por caminho absoluto, e sem arquivo a
// Cinzel cai para uma serifada do sistema em toda tela, que é um defeito de
// aparência que ninguém liga à causa.
func TestTheStylesheetFontsExist(t *testing.T) {
	fontes, err := os.ReadDir("assets/static/fonts")
	if err != nil {
		t.Fatalf("ler as fontes embutidas: %v", err)
	}
	if len(fontes) != 2 {
		t.Fatalf("%d fontes embutidas, e a folha declara duas (latin e latin-ext)", len(fontes))
	}
	for _, f := range fontes {
		info, err := f.Info()
		if err != nil || info.Size() == 0 {
			t.Errorf("%s está vazia: o navegador ignora a fonte e cai na do sistema", f.Name())
		}
	}
}
