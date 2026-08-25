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
