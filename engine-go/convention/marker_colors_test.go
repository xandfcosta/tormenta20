package convention

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// O guarda da COR do marcador (ALE-264): a lista que o domínio aceita e o CSS
// que a pinta têm de andar juntas. Irmão do `TestEveryOfferedGroundCanBePainted`,
// e ele existe porque o defeito dele JÁ TINHA ACONTECIDO aqui.
//
// A autoridade sempre aceitou `ouro/carmim/azul/verde`; o piloto tinha escrito
// `gold/red/green/blue/violet` no view e no CSS. Nenhuma das cinco casava com
// nenhuma das quatro, então todo marcador caía no dourado — o carmim que o
// mestre escolheu na outra tela chegava aqui dourado, sem erro em lugar nenhum.
//
// Amostragem e não enumeração: o guarda percorre a LISTA, então a quinta cor já
// nasce medida.
// **Ele mudou de casa na ALE-278, e o motivo é o de sempre nesta família.** Ele
// morava na cena da Mesa e lia `piloto/piloto.src.css` — um caminho relativo ao
// `api`. Quando a Mesa virou `web/table`, o arquivo sumiu debaixo dele; aqui a
// falha foi ALTA, porque o controle da folha existia. O guarda do foco, que não
// tinha piso, teria passado verde medindo metade.
func TestEveryMarkerColorCanBePainted(t *testing.T) {
	css, err := os.ReadFile(filepath.Join("..", "api", "piloto", "piloto.src.css"))
	if err != nil {
		t.Fatalf("ler o CSS do piloto: %v", err)
	}
	folha := string(css)

	// O CONTROLE: a folha tem a família que vamos procurar. Sem ele, um arquivo
	// renomeado daria "nenhuma cor encontrada", que se parece com "todas
	// faltando" e passaria verde se a asserção fosse ao contrário.
	if !strings.Contains(folha, "--marcador-") {
		t.Fatalf("o CSS do piloto não tem nenhuma variável --marcador-* — o guarda está lendo o arquivo errado (%d bytes)", len(folha))
	}
	// E o CONTROLE da lista: uma lista vazia faria o laço abaixo não rodar
	// nenhuma vez e o teste passaria afirmando nada.
	if len(tabuleiro.MarkerColors) == 0 {
		t.Fatal("o domínio não oferece cor nenhuma — não há o que medir")
	}

	for _, cor := range tabuleiro.MarkerColors {
		if !strings.Contains(folha, "--marcador-"+cor.ID) {
			t.Errorf("a cor %q (%s) é aceita pelo domínio e o CSS não sabe pintá-la: falta --marcador-%s",
				cor.ID, cor.Rotulo, cor.ID)
		}
		if cor.Rotulo == "" {
			t.Errorf("a cor %q não tem rótulo para o mestre ler", cor.ID)
		}
	}
}
