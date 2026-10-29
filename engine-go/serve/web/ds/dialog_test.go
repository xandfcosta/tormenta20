package ds

import (
	"bytes"
	"context"
	"regexp"
	"testing"
)

// O TOQUE NO FUNDO FECHA O DIÁLOGO. O `<dialog>` modal não faz isso sozinho, e
// o fundo entrega o clique ao próprio `<dialog>`. O alvo tem de ser o diálogo
// E o ponto tem de estar fora da caixa: o clique de teclado (Enter num botão)
// chega com as coordenadas zeradas, e só o alvo o separa do toque no fundo.
func TestTheDialogClosesOnABackdropTouch(t *testing.T) {
	var out bytes.Buffer
	if err := Dialog("picker", "Escolher", nil).Render(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	click := regexp.MustCompile(`<dialog[^>]*data-on:click="([^"]*)"`).FindStringSubmatch(out.String())
	if click == nil {
		t.Fatalf("o diálogo não escuta o toque no fundo:\n%s", out.String())
	}
	for what, part := range map[string]string{
		"o alvo é o próprio diálogo": "evt.target === el",
		"o ponto fora da caixa":      "getBoundingClientRect()",
		"o fechar":                   "el.close()",
	} {
		if !regexp.MustCompile(regexp.QuoteMeta(part)).MatchString(click[1]) {
			t.Errorf("o gesto do fundo não tem %s: %q", what, click[1])
		}
	}
}
