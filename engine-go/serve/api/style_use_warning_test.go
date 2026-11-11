package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// O USAR DOS ESTILOS AVISA O QUE O LIVRO PEDE DA ARMA, e não trava: quem decide é o mestre, e o uso
// continua valendo. O aviso sai na linha do poder, nos Poderes e nas Ações da mesa, que é onde se usa no
// meio da luta.
//
// Estilo de Duas Armas: "se estiver empunhando duas armas (e pelo menos uma delas for leve)" (p125), e o
// Arma Secundária Grande dispensa a leve (p124). Disparo Rápido: arma "que possa recarregar como ação
// livre" (p125), e só o arco recarrega assim (p146-151).
func TestTheStyleUseWarnsWhatTheWeaponLacks(t *testing.T) {
	const light, bow = "Nenhuma arma empunhada é leve", "Só o arco recarrega como ação livre"
	for _, c := range []struct {
		name, powers, warning string
		weapons               []string
		warns                 bool
	}{
		{"duas de uma mão", `["estilo-de-duas-armas"]`, light, []string{"espada-longa", "espada-longa"}, true},
		{"uma delas leve", `["estilo-de-duas-armas"]`, light, []string{"espada-longa", "adaga"}, false},
		{"com Arma Secundária Grande", `["estilo-de-duas-armas","arma-secundaria-grande"]`, light, []string{"espada-longa", "espada-longa"}, false},
		{"sem arma na mão", `["estilo-de-duas-armas"]`, light, nil, true},
		{"arco curto", `["disparo-rapido"]`, bow, []string{"arco-curto"}, false},
		{"arco longo", `["disparo-rapido"]`, bow, []string{"arco-longo"}, false},
		{"besta leve", `["disparo-rapido"]`, bow, []string{"besta-leve"}, true},
		{"funda", `["disparo-rapido"]`, bow, []string{"funda"}, true},
		{"pistola", `["disparo-rapido"]`, bow, []string{"pistola"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			seedPowerChoices(t, f.s, f.charID, c.powers, "{}")
			for _, weapon := range c.weapons {
				itemSemeia(t, f, f.charID, weapon, weapon, "wielded")
			}
			page := f.requests(t, f.player, http.MethodGet, fmt.Sprintf("/personagens/%d?tab=abilities", f.charID), "").Body.String()
			if !strings.Contains(page, "/poderes/usa/") {
				t.Fatal("o controle falhou: os Poderes não desenharam o Usar")
			}
			if got := strings.Contains(page, c.warning); got != c.warns {
				t.Errorf("os Poderes avisaram = %v, want %v", got, c.warns)
			}
			if len(c.weapons) == 0 {
				return
			}
			actions := acoesDaMesa(t, f)
			if !strings.Contains(actions, "Ação padrão") {
				t.Fatal("o controle falhou: a mesa não desenhou as ações")
			}
			if got := strings.Contains(actions, c.warning); got != c.warns {
				t.Errorf("as Ações avisaram = %v, want %v", got, c.warns)
			}
		})
	}
}
