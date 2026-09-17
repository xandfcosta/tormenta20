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
	tela := f.pede(t, f.mestre, http.MethodGet, "/", "").Body.String()

	// O CONTROLE: a página REFERENCIA estáticos. Sem ele, "nenhum endereço cru"
	// seria verdade também sobre uma página que não carregou.
	if !strings.Contains(tela, "/static/") {
		t.Fatal("a página não referencia estático nenhum — o guarda mediria a tela errada")
	}
	if strings.Contains(tela, `"/static/app.css"`) {
		t.Error("a folha entrou sem versão: ela volta a ser rebaixada a cada troca de página")
	}
	for _, pedaco := range strings.Split(tela, "/static/")[1:] {
		fim := strings.IndexAny(pedaco, `"'`)
		if fim < 0 {
			continue
		}
		if !strings.Contains(pedaco[:fim], "?v=") {
			t.Errorf("endereço estático sem versão: /static/%s", pedaco[:fim])
		}
	}
}
