package sheetui

import (
	"strings"
	"testing"
)

// O CHIP DAS AÇÕES LEVA A MARCA DA FICHA EM QUE É DESENHADO (ALE-473). O
// endereço era montado à mão, e o toque num chip devolvia a ficha de página
// inteira dentro da sessão. Agora o chip guarda o caminho, e quem escreve o
// endereço é o `sheetPost` da ficha.
func TestAChipGestureCarriesTheMarkOfTheSheetItIsDrawnIn(t *testing.T) {
	chip := stanceChips(powerRow{Name: "Fúria", Can: true, Stance: &stanceState{Flag: "furia", BasePm: 2}})[0]

	embedded := chipCommand(View{ID: 7, ActiveTab: "expertises", Embedded: true}, chip)
	for _, want := range []string{"/personagens/7/poderes/postura/furia/entra", "embutida=1"} {
		if !strings.Contains(embedded, want) {
			t.Errorf("o chip da sessão não leva %q: %s", want, embedded)
		}
	}
	if loose := chipCommand(View{ID: 7, ActiveTab: "expertises"}, chip); strings.Contains(loose, "embutida=1") {
		t.Errorf("o chip da ficha solta levou a marca da embutida: %s", loose)
	}
}
