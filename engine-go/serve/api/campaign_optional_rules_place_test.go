package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// AS REGRAS OPCIONAIS MORAM NA ABA "REGRAS DA MESA", junto das regras da mesa:
// o mestre liga e desliga regra num lugar só (decisão do dono). A aba
// Configurações deixa de tê-las, e o jogador, que lê as regras da mesa, não
// recebe o interruptor — a rota dele o recusaria.
func TestTheOptionalRulesLiveOnTheTableRulesTab(t *testing.T) {
	f := newSceneFixture(t)
	page := func(who int64, tab string) string {
		t.Helper()
		return f.requests(t, who, http.MethodGet, fmt.Sprintf("/campanhas/%d?tab=%s", f.campaignID, tab), "").Body.String()
	}
	if !strings.Contains(page(f.gm, "regras"), `id="rules-panel"`) {
		t.Error("a aba Regras da mesa do mestre não tem as regras opcionais")
	}
	config := page(f.gm, "config")
	// O CONTROLE: a aba Configurações abriu. Sem ele, "não tem o cartão" seria
	// verdade numa página de erro.
	if !strings.Contains(config, "Editar campanha") {
		t.Fatal("a aba Configurações do mestre não abriu")
	}
	if strings.Contains(config, `id="rules-panel"`) {
		t.Error("as regras opcionais continuam na aba Configurações")
	}
	if strings.Contains(page(f.player, "regras"), `id="rules-panel"`) {
		t.Error("o jogador recebeu o interruptor das regras opcionais")
	}
}
