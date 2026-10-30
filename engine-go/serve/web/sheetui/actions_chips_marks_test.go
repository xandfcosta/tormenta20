package sheetui

import (
	"strings"
	"testing"
)

// O CHIP DAS AÇÕES LEVA AS MARCAS DA FICHA EM QUE É DESENHADO (ALE-473). O
// endereço era montado à mão com `?embutida=1` fixo, e na sessão paralela o
// toque num chip devolvia a ficha de HOJE dentro dela. Agora o chip guarda o
// caminho, e quem escreve o endereço é o `sheetPost` da ficha.
func TestAChipGestureCarriesTheMarksOfTheSheetItIsDrawnIn(t *testing.T) {
	chip := stanceChips(powerRow{Name: "Fúria", Can: true, Stance: &stanceState{Flag: "furia", BasePm: 2}})[0]

	parallel := chipCommand(View{ID: 7, ActiveTab: "expertises", Embedded: true, Tempestade: true}, chip)
	for _, want := range []string{"/personagens/7/poderes/postura/furia/entra", "embutida=1", "tempestade=1"} {
		if !strings.Contains(parallel, want) {
			t.Errorf("o chip da sessão paralela não leva %q: %s", want, parallel)
		}
	}
	if today := chipCommand(View{ID: 7, ActiveTab: "expertises", Embedded: true}, chip); strings.Contains(today, "tempestade=1") {
		t.Errorf("o chip da sessão de hoje levou a marca da paralela: %s", today)
	}
}
