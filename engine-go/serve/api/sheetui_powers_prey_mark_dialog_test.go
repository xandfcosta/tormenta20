package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// O DIÁLOGO DA MARCA escreve o que os acréscimos dão, e a caixa do Inimigo
// aparece só para quem tem o poder — mesmo no nível sem degrau a escolher.
func TestThePreyMarkDialogWritesTheRidersAndTheFavoredBox(t *testing.T) {
	f, id := hunterWith(t, 1, `["class.cacador.espreitar","class.cacador.inimigo-de-criatura"]`)
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
	dialog := preyMarkDialogText(t, page)
	if dialog == "" {
		t.Fatal("o nível 1 com o Inimigo não abre o diálogo da Marca: sem degrau, a caixa ficaria sem lugar")
	}
	if !strings.Contains(dialog, "A presa é do tipo do Inimigo") {
		t.Error("o diálogo não oferece a caixa do Inimigo")
	}
	if !strings.Contains(dialog, "perícias +2 (Espreitar)") {
		t.Errorf("o diálogo não diz o Espreitar dobrado (+2 sem degrau): %q", dialog)
	}

	f2, id2 := hunterWith(t, 5, `["class.cacador.espreitar"]`)
	page = f2.requests(t, f2.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", id2), "").Body.String()
	// O CONTROLE da ausência abaixo: o diálogo existe para quem não tem o
	// Inimigo, e é dentro dele que a caixa faltaria.
	dialog = preyMarkDialogText(t, page)
	if !strings.Contains(dialog, "perícias +1 (Espreitar)") {
		t.Errorf("o diálogo não diz o Espreitar sem degrau: %q", dialog)
	}
	if strings.Contains(page, "A presa é do tipo do Inimigo") {
		t.Error("a caixa do Inimigo aparece para quem não tem o poder")
	}
}

// preyMarkDialogText é o texto do diálogo dos degraus da Marca da Presa, ou ""
// quando a ficha não o desenha.
func preyMarkDialogText(t *testing.T, page string) string {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		t.Fatalf("ler a página: %v", err)
	}
	dialog := findNode(doc, "id", "use-steps-class-cacador-marca-da-presa-1d4")
	if dialog == nil {
		return ""
	}
	return spacedText(dialog)
}
