package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// O DIÁLOGO DA MARCA escreve o que os acréscimos dão, e a caixa do Inimigo
// aparece só para quem tem o poder — mesmo no nível sem degrau a escolher.
func TestThePreyMarkDialogWritesTheRidersAndTheFavoredBox(t *testing.T) {
	f, id := hunterWith(t, 1, `["class.cacador.espreitar","class.cacador.inimigo-de-criatura"]`)
	page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", id), "").Body.String()
	if !strings.Contains(page, "usa-class-cacador-marca-da-presa-1d4") {
		t.Fatal("o nível 1 com o Inimigo não abre o diálogo da Marca: sem degrau, a caixa ficaria sem lugar")
	}
	if !strings.Contains(page, "A presa é do tipo do Inimigo") {
		t.Error("o diálogo não oferece a caixa do Inimigo")
	}
	if !strings.Contains(page, ">+2</span> (Espreitar)") {
		t.Error("o diálogo não diz o Espreitar dobrado (+2 sem degrau)")
	}

	f2, id2 := hunterWith(t, 5, `["class.cacador.espreitar"]`)
	page = f2.requests(t, f2.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", id2), "").Body.String()
	if !strings.Contains(page, ">+1</span> (Espreitar)") {
		t.Error("o diálogo não diz o Espreitar sem degrau")
	}
	if strings.Contains(page, "A presa é do tipo do Inimigo") {
		t.Error("a caixa do Inimigo aparece para quem não tem o poder")
	}
}
