package api

import (
	"bytes"
	"os"
	"path/filepath"
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
func TestTodoChaoOferecidoTemComoSerPintado(t *testing.T) {
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

	for _, chao := range tabuleiro.ChoesDoLugar {
		if !strings.Contains(folha, ".chao-"+chao.ID) {
			t.Errorf("o chão %q (%s) é oferecido na tela e o CSS não sabe pintá-lo: falta .chao-%s",
				chao.ID, chao.Rotulo, chao.ID)
		}
		if chao.Rotulo == "" {
			t.Errorf("o chão %q não tem rótulo para o mestre ler", chao.ID)
		}
	}
}

// TestAregraQueESCONDEoDialogoFICAforaDeCAMADA.
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
func TestARegraQueEscondeODialogoFicaForaDeCamada(t *testing.T) {
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

// TestAsFONTESembutidasSAOasMESMASdaSPA.
//
// A cópia é deliberada e o guarda é o que a torna dívida em vez de armadilha:
// `go:embed` não alcança `../frontend/public/fonts`, que está fora do módulo Go,
// então as duas woff2 vivem em dois lugares. Sem este guarda, atualizar a fonte
// num lado só apareceria como um desenho levemente errado no outro — a classe de
// defeito que ninguém liga à causa.
func TestAsFontesEmbutidasSaoAsMesmasDaSPA(t *testing.T) {
	fontes, err := os.ReadDir("piloto/static/fonts")
	if err != nil {
		t.Fatalf("ler as fontes embutidas: %v", err)
	}
	// O CONTROLE: há fonte para comparar. Um diretório vazio faria o laço abaixo
	// não rodar nenhuma vez, e o teste passaria verde afirmando nada.
	if len(fontes) == 0 {
		t.Fatal("nenhuma fonte embutida: a folha pede /fonts/ e o piloto não tem o que servir")
	}

	for _, f := range fontes {
		embutida, err := os.ReadFile(filepath.Join("piloto/static/fonts", f.Name()))
		if err != nil {
			t.Errorf("ler %s embutida: %v", f.Name(), err)
			continue
		}
		daSPA, err := os.ReadFile(filepath.Join("..", "..", "frontend", "public", "fonts", f.Name()))
		if err != nil {
			t.Errorf("%s está embutida no piloto e não existe na SPA: %v", f.Name(), err)
			continue
		}
		if !bytes.Equal(embutida, daSPA) {
			t.Errorf("%s divergiu: %d bytes embutidos contra %d na SPA — a fonte foi atualizada num lugar só",
				f.Name(), len(embutida), len(daSPA))
		}
	}
}
