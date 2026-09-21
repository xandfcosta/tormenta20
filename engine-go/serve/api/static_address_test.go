package api

import (
	"net/http"
	"strings"
	"testing"
)

// A regressão silenciosa: caminho cru continua funcionando, e é servido SEM
// cache. A página que escrever um à mão volta a piscar, sozinha, e ninguém liga
// uma coisa à outra — o sintoma aparece em UMA tela e a causa está noutro
// arquivo.
//
// Ele mora aqui e não no `serve/web/assets` porque precisa de uma PÁGINA
// renderizada: quem monta os endereços é a casca, e o pacote dos estáticos não
// conhece cena nenhuma.
func TestEveryStaticAddressOnThePageIsVersioned(t *testing.T) {
	f := newSceneFixture(t)
	screen := f.pede(t, f.gm, http.MethodGet, "/", "").Body.String()

	// O CONTROLE: a página REFERENCIA estáticos. Sem ele, "nenhum endereço cru"
	// seria verdade também sobre uma página que não carregou.
	if !strings.Contains(screen, "/static/") {
		t.Fatal("a página não referencia estático nenhum — o guarda mediria a tela errada")
	}
	if strings.Contains(screen, `"/static/app.css"`) {
		t.Error("a folha entrou sem versão: ela volta a ser rebaixada a cada troca de página")
	}
	for _, chunk := range strings.Split(screen, "/static/")[1:] {
		end := strings.IndexAny(chunk, `"'`)
		if end < 0 {
			continue
		}
		if !strings.Contains(chunk[:end], "?v=") {
			t.Errorf("endereço estático sem versão: /static/%s", chunk[:end])
		}
	}
}
