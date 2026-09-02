package api

import (
	"os"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// O guarda do CHÃO do lugar (ALE-264): a lista que a tela OFERECE e o CSS que a
// PINTA têm de andar juntas.
//
// A lista nasceu porque o mesmo conjunto já existia duas vezes — `.chao-*` aqui
// e `TERRAIN_LABEL` na SPA — e uma terceira cópia à mão no templ é como nasce a
// opção escolhível que o navegador desenha em branco. O defeito não estoura:
// ele pinta o chão errado, em silêncio, que é a marca desta família.
//
// Amostragem e não enumeração: o guarda percorre a LISTA, então o chão que
// alguém acrescentar amanhã já nasce medido — não há uma entrada por caso aqui
// para alguém esquecer de escrever.
func TestEveryOfferedGroundCanBePainted(t *testing.T) {
	css, err := os.ReadFile("piloto/piloto.src.css")
	if err != nil {
		t.Fatalf("ler o CSS do piloto: %v", err)
	}
	folha := string(css)

	// O CONTROLE: a folha tem a família que vamos procurar. Sem ele, um caminho
	// errado ou um arquivo renomeado daria "nenhum chão encontrado" — que se
	// parece com "todos faltando" e passaria verde se a asserção fosse ao
	// contrário.
	if !strings.Contains(folha, ".chao-") {
		t.Fatalf("o CSS do piloto não tem nenhuma classe .chao-* — o guarda está lendo o arquivo errado (%d bytes)", len(folha))
	}

	for _, chao := range tabuleiro.PlaceGrounds {
		if !strings.Contains(folha, ".chao-"+chao.ID) {
			t.Errorf("o chão %q (%s) é oferecido na tela e o CSS não sabe pintá-lo: falta .chao-%s",
				chao.ID, chao.Rotulo, chao.ID)
		}
		if chao.Rotulo == "" {
			t.Errorf("o chão %q não tem rótulo para o mestre ler", chao.ID)
		}
	}
}

// TestTheRuleThatHidesTheDialogStaysOutOfTheLayer.
//
// Guarda de CASCATA, e ele prende a COLOCAÇÃO porque é ela o defeito.
//
// A regra viveu dentro de `@layer components` e nunca valeu: o elemento carrega
// a utilitária `flex` do Tailwind, que mora numa camada POSTERIOR, e camada
// posterior ganha de anterior independentemente de especificidade. O efeito era
// o pior possível — numa tela larga apareciam OS DOIS, o painel lateral com a
// ficha e o modal por cima dela. O dono viu e perguntou por que havia diálogo se
// a ficha já abre ao lado; o comentário do código afirmava que não havia.
//
// Medido antes do conserto: contêiner de 1276×566, as duas condições da consulta
// casando, e `display: flex`. A MESMA regra injetada sem camada devolveu
// `display: none` — o experimento que fecha a causa.
//
// O que este guarda NÃO faz: ele não resolve cascata, ele lê TEXTO. Cascata de
// verdade só um navegador resolve, e um e2e para uma linha seria caro. O que ele
// pega é a regressão exata e provável — alguém arrastar a regra de volta para
// dentro do `@layer` numa arrumação, achando que camada é organização.
func TestTheRuleThatHidesTheDialogStaysOutOfTheLayer(t *testing.T) {
	folha, err := os.ReadFile("piloto/piloto.src.css")
	if err != nil {
		t.Fatalf("ler o CSS do piloto: %v", err)
	}
	css := string(folha)

	pos := strings.Index(css, ".mesa-ficha-em-dialogo")
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

// TestTheStylesheetFontsExist.
//
// Aqui morava o `TestAsFontesEmbutidasSaoAsMesmasDaSPA`, que comparava as woff2
// embutidas com as da SPA byte a byte: `go:embed` não alcançava
// `../frontend/public/fonts`, então a fonte vivia em dois lugares e o guarda era
// o que tornava a cópia dívida em vez de armadilha.
//
// Com a SPA apagada (ALE-272, fatia 10c) não há segundo lado: estas SÃO as
// fontes. O que sobra para prender é a presença — a folha pede `/fonts/…` por
// caminho absoluto, e sem arquivo a Cinzel cai para uma serifada do sistema em
// toda tela, que é um defeito de aparência que ninguém liga à causa.
func TestTheStylesheetFontsExist(t *testing.T) {
	fontes, err := os.ReadDir("piloto/static/fonts")
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
