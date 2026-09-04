package api

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

func TestNoLayerReadsThePointWithoutAddingTheViewport(t *testing.T) {
	f := novoPiloto(t)
	if rec := f.pede(t, f.mestre, http.MethodPost, f.tableUrl()+"/tabuleiro/abrir",
		`{"novolugar":"Taverna do Javali","novochao":"taverna"}`); rec.Code != http.StatusOK {
		t.Fatalf("abrir o tabuleiro deu %d", rec.Code)
	}
	tela := f.pede(t, f.mestre, http.MethodGet, f.tableUrl(), "").Body.String()

	// O CONTROLE vem primeiro: sem ele, "não achei nenhuma leitura crua" é
	// indistinguível de "não achei leitura nenhuma" — e as duas passariam verde.
	lidas := regexp.MustCompile(`evt\.offset[XY]`).FindAllString(tela, -1)
	if len(lidas) < 8 {
		t.Fatalf("a cena só tem %d leituras de ponto: o canal não está aberto, e a ausência abaixo não é evidência", len(lidas))
	}

	// Cada leitura tem de vir somada à vista. A expressão é sempre
	// `(evt.offsetX + $vistax)`, então basta olhar o que vem logo depois.
	for _, eixo := range []struct{ ponto, sinal string }{
		{"evt.offsetX", "vistax"},
		{"evt.offsetY", "vistay"},
	} {
		olho := regexp.MustCompile(regexp.QuoteMeta(eixo.ponto) + `(?: \+ \$` + eixo.sinal + `)?`)
		for _, achado := range olho.FindAllString(tela, -1) {
			if !strings.Contains(achado, eixo.sinal) {
				t.Errorf("uma camada lê %q sem somar a janela: com a vista arrastada ela clica no quadrado errado", achado)
			}
		}
	}
}
