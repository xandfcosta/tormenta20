package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func TestNoLayerReadsThePointWithoutAddingTheViewport(t *testing.T) {
	f := newSceneFixture(t)
	if rec := f.requests(t, f.gm, http.MethodPost, f.tableUrl()+"/tabuleiro/abrir",
		`{"new_place":"Taverna do Javali","new_ground":"tavern"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir o tabuleiro deu %d", rec.Code)
	}
	screen := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE vem primeiro: sem ele, "não achei nenhuma leitura crua" é
	// indistinguível de "não achei leitura nenhuma" — e as duas passariam verde.
	read := regexp.MustCompile(`evt\.offset[XY]`).FindAllString(screen, -1)
	if len(read) < 8 {
		t.Fatalf("a cena só tem %d leituras de ponto: o canal não está aberto, e a ausência abaixo não é evidência", len(read))
	}

	// Cada leitura tem de vir somada à vista. A expressão é sempre
	// `(evt.offsetX + $viewport_x)`, então basta olhar o que vem logo depois.
	for _, axis := range []struct{ point, signal string }{
		{"evt.offsetX", "viewport_x"},
		{"evt.offsetY", "viewport_y"},
	} {
		eye := regexp.MustCompile(regexp.QuoteMeta(axis.point) + `(?: \+ \$` + axis.signal + `)?`)
		for _, found := range eye.FindAllString(screen, -1) {
			if !strings.Contains(found, axis.signal) {
				t.Errorf("uma camada lê %q sem somar a janela: com a vista arrastada ela clica no quadrado errado", found)
			}
		}
	}
}
