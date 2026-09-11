package board

import "testing"

// O CHAO QUE A FOLHA NAO SABE PINTAR NAO E GRAVADO (ALE-301).
//
// O caminho de criar lugar gravava o `terrain` que chegasse do formulário, e o
// de abrir cena filtrava — duas portas para o mesmo campo, uma com guarda e
// outra sem. Enquanto os ids eram os do formulário da casa isso nunca apareceu;
// com eles em inglês, um cliente velho grava `pedra` e o quadrado desenha sem
// textura, porque a classe virou `ground-stone`.
func TestAnUnknownGroundFallsBackToTheDefault(t *testing.T) {
	// O CONTROLE: um chão que EXISTE atravessa. Sem ele, uma função que
	// devolvesse o padrão sempre passaria no caso de baixo.
	if got := KnownGround("crypt"); got != "crypt" {
		t.Errorf("o chão conhecido não atravessou: %q", got)
	}
	for _, asked := range []string{"pedra", "cripta", "", "vulcão"} {
		if got := KnownGround(asked); got != DefaultGround() {
			t.Errorf("o chão %q foi aceito como %q, e a folha não sabe pintá-lo", asked, got)
		}
	}
}
