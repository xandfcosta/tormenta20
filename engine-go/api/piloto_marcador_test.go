package api

import (
	"os"
	"strings"
	"testing"

	"t20engine/tabuleiro"
)

// O guarda da COR do marcador (ALE-264): a lista que o domínio aceita e o CSS
// que a pinta têm de andar juntas. Irmão do `TestTodoChaoOferecidoTemComoSerPintado`,
// e ele existe porque o defeito dele JÁ TINHA ACONTECIDO aqui.
//
// A autoridade sempre aceitou `ouro/carmim/azul/verde`; o piloto tinha escrito
// `gold/red/green/blue/violet` no view e no CSS. Nenhuma das cinco casava com
// nenhuma das quatro, então todo marcador caía no dourado — o carmim que o
// mestre escolheu na outra tela chegava aqui dourado, sem erro em lugar nenhum.
//
// Amostragem e não enumeração: o guarda percorre a LISTA, então a quinta cor já
// nasce medida.
func TestTodaCorDeMarcadorTemComoSerPintada(t *testing.T) {
	css, err := os.ReadFile("piloto/piloto.src.css")
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
	if len(tabuleiro.CoresDeMarcador) == 0 {
		t.Fatal("o domínio não oferece cor nenhuma — não há o que medir")
	}

	for _, cor := range tabuleiro.CoresDeMarcador {
		if !strings.Contains(folha, "--marcador-"+cor.ID) {
			t.Errorf("a cor %q (%s) é aceita pelo domínio e o CSS não sabe pintá-la: falta --marcador-%s",
				cor.ID, cor.Rotulo, cor.ID)
		}
		if cor.Rotulo == "" {
			t.Errorf("a cor %q não tem rótulo para o mestre ler", cor.ID)
		}
	}
}

// TestCorDeMarcadorDesconhecidaCaiNoPadrao — o caminho do dado torto.
//
// A cor vem do banco, logo é dado de CLIENTE: ela entra num `style`, e string
// livre daqui seria injeção de CSS no estado da mesa. Este guarda prende o
// fallback e, de quebra, prende que ele aponta para uma variável que EXISTE —
// era exatamente aí que o defeito antigo morava.
func TestCorDeMarcadorDesconhecidaCaiNoPadrao(t *testing.T) {
	padrao := corDeMarcador(tabuleiro.CorPadraoDeMarcador())

	for _, torta := range []string{"gold", "red", "'; background: url(x)", ""} {
		if got := corDeMarcador(torta); got != padrao {
			t.Errorf("a cor %q devolveu %q em vez do padrão %q", torta, got, padrao)
		}
	}
	// O CONTROLE: uma cor BOA não cai no padrão por acidente, senão o teste
	// acima seria verdade sobre uma função que devolve sempre a mesma coisa.
	outra := ""
	for _, c := range tabuleiro.CoresDeMarcador {
		if c.ID != tabuleiro.CorPadraoDeMarcador() {
			outra = c.ID
			break
		}
	}
	if outra == "" {
		t.Fatal("só há uma cor — o controle não tem como distinguir nada")
	}
	if corDeMarcador(outra) == padrao {
		t.Errorf("a cor %q devolveu o padrão: a função não distingue cor nenhuma", outra)
	}
}
