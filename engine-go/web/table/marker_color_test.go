package table

import (
	"testing"

	"t20engine/tabuleiro"
)

// O FALLBACK da cor de marcador fica NA CENA, e a varredura da folha foi para
// o `convention/` (ALE-278): esta é a regra de uma função, aquela é a
// convenção do repositório inteiro — e foi a que quebrou ao mudar de casa,
// porque lia um caminho relativo ao `api`.

// TestAnUnknownMarkerColorFallsBackToTheDefault — o caminho do dado torto.
//
// A cor vem do banco, logo é dado de CLIENTE: ela entra num `style`, e string
// livre daqui seria injeção de CSS no estado da mesa. Este guarda prende o
// fallback e, de quebra, prende que ele aponta para uma variável que EXISTE —
// era exatamente aí que o defeito antigo morava.
func TestAnUnknownMarkerColorFallsBackToTheDefault(t *testing.T) {
	padrao := markerColor(tabuleiro.DefaultMarkerColor())

	for _, torta := range []string{"gold", "red", "'; background: url(x)", ""} {
		if got := markerColor(torta); got != padrao {
			t.Errorf("a cor %q devolveu %q em vez do padrão %q", torta, got, padrao)
		}
	}
	// O CONTROLE: uma cor BOA não cai no padrão por acidente, senão o teste
	// acima seria verdade sobre uma função que devolve sempre a mesma coisa.
	outra := ""
	for _, c := range tabuleiro.MarkerColors {
		if c.ID != tabuleiro.DefaultMarkerColor() {
			outra = c.ID
			break
		}
	}
	if outra == "" {
		t.Fatal("só há uma cor — o controle não tem como distinguir nada")
	}
	if markerColor(outra) == padrao {
		t.Errorf("a cor %q devolveu o padrão: a função não distingue cor nenhuma", outra)
	}
}
