package api

import (
	"net/http"
	"regexp"
	"testing"
)

// A PRÉVIA DO DIÁLOGO DE DEGRAUS diz o preço que se cobra (ALE-647): sob o Alquebrado, a Cura pelas Mãos de 1 PM
// sai por 2, e a prévia o diz — ela fazia a conta crua no navegador, e mostrava 1.
func TestTheUseDialogPreviewSaysThePriceItCharges(t *testing.T) {
	f, id := paladinWithCharisma(t, 9)
	if rec := effect(t, f, id, "condicao/alquebrado"); rec.Code != http.StatusOK {
		t.Fatalf("aplicar o Alquebrado: status %d", rec.Code)
	}
	screen := sheetTab(t, f, id, "abilities")
	preview := regexp.MustCompile(`\[(\d+)(?:,\d+)*\]\[\$stance_degrees\]`).FindStringSubmatch(screen)
	if preview == nil || preview[1] != "2" {
		t.Errorf("a prévia do diálogo da Cura pelas Mãos começa em %v; want 2, o preço com o Alquebrado", preview)
	}
}
