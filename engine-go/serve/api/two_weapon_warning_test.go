package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// "Se estiver empunhando duas armas (e pelo menos uma delas for leve)" (p125), e o
// Arma Secundária Grande dispensa a leve (p124). A tela avisa e não trava: quem
// decide é o mestre, e o interruptor continua ligando.
func TestTheTwoWeaponSwitchWarnsWithoutALightWeapon(t *testing.T) {
	const warning = "Nenhuma arma empunhada é leve"
	for _, c := range []struct {
		name, powers string
		weapons      []string
		warns        bool
	}{
		{"duas de uma mão", `["estilo-de-duas-armas"]`, []string{"espada-longa", "espada-longa"}, true},
		{"uma delas leve", `["estilo-de-duas-armas"]`, []string{"espada-longa", "adaga"}, false},
		{"com Arma Secundária Grande", `["estilo-de-duas-armas","arma-secundaria-grande"]`, []string{"espada-longa", "espada-longa"}, false},
		{"sem arma na mão", `["estilo-de-duas-armas"]`, nil, true},
		{"outro interruptor não avisa", `["ataque-poderoso"]`, []string{"espada-longa"}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			seedPowerChoices(t, f.s, f.charID, c.powers, "{}")
			for _, weapon := range c.weapons {
				itemSemeia(t, f, f.charID, weapon, weapon, "wielded")
			}
			page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=conditionals", f.charID), "").Body.String()
			if !strings.Contains(page, "Situação · você liga quando vale") {
				t.Fatal("o controle falhou: a aba Efeitos não desenhou os interruptores")
			}
			if got := strings.Contains(page, warning); got != c.warns {
				t.Errorf("avisou = %v, want %v", got, c.warns)
			}
			if len(c.weapons) == 0 {
				return
			}
			// As Ações da mesa são onde o interruptor se liga no meio da luta.
			chips := acoesDaMesa(t, f)
			if !strings.Contains(chips, "Ação padrão") {
				t.Fatal("o controle falhou: a mesa não desenhou as ações")
			}
			if got := strings.Contains(chips, "sem arma leve"); got != c.warns {
				t.Errorf("o chip avisou = %v, want %v", got, c.warns)
			}
		})
	}
}
