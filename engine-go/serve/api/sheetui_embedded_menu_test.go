package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// DENTRO DA SESSÃO, O ⋯ DA FICHA TEM QUEM O ABRA.
//
// O cabeçalho da ficha é trocado pelo da sessão, e o único botão do ⋯ morava nele. O
// diálogo continuava desenhado, e nada o abria: a 390px sumiam o degrau de nível, quem
// é o herói e o modo de cor. O ⋯ entra no fim da fileira de seções (celular) e no fim
// das abas (desktop), decisão do dono.
func TestTheEmbeddedSheetCanOpenItsMenu(t *testing.T) {
	f := newSceneFixture(t)
	opens := `commandfor="sheet-menu"`

	// O CONTROLE: a ficha solta abre o ⋯ pelo cabeçalho, uma vez.
	alone := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d", f.charID), "").Body.String()
	if n := strings.Count(alone, opens); n != 1 {
		t.Fatalf("a ficha solta tem %d botões do ⋯, e o cabeçalho tem um", n)
	}

	embedded := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?embutida=1", f.charID), "").Body.String()
	if !strings.Contains(embedded, `id="sheet-menu"`) {
		t.Fatal("a ficha embutida não desenhou o diálogo do ⋯ — o caso mediria outra coisa")
	}
	_, nav, ok := strings.Cut(embedded, `class="ds-section-nav"`)
	if !ok {
		t.Fatal("a ficha embutida não tem a fileira de seções")
	}
	nav, _, _ = strings.Cut(nav, "</nav>")
	if !strings.Contains(nav, opens) {
		t.Error("a fileira de seções da ficha embutida não abre o ⋯: no celular, nada o abre")
	}
	// O RÓTULO diz o que abre, e não colide com o "Mais" da ficha solta, que abre as
	// seções que não cabem na barra (decisão do dono).
	if _, item, _ := strings.Cut(nav, opens); !strings.Contains(item, "Herói") {
		t.Error("o ⋯ da fileira não se chama \"Herói\"")
	}
	_, tabs, _ := strings.Cut(embedded, `class="ds-sheet-tabs"`)
	tabs, _, _ = strings.Cut(tabs, "</div>")
	if !strings.Contains(tabs, opens) {
		t.Error("as abas da ficha embutida não abrem o ⋯: no desktop, nada o abre")
	}
}
