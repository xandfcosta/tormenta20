package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// GRANADEIRO (p69): "você pode usar sua Inteligência em vez de Destreza para calcular a CD do teste
// de resistência" dos itens alquímicos. Vale o MAIOR dos dois (decisão do dono, como no Pequeno e
// Rechonchudo), e vale na dose da Injeção alquímica, que libera o próprio preparado (p165). A ficha é
// de nível 8: a CD é 10 + 4 + o atributo.
func TestTheGrenadierUsesIntelligenceForTheAlchemicalDC(t *testing.T) {
	cases := []struct {
		name          string
		grenadier     bool
		intel, dex    int
		catalog, item string
		want, refuse  string
	}{
		{"com o poder e a Inteligência maior", true, 5, 3, "acido", "Ácido",
			"Reflexos CD 19, pela Inteligência do Granadeiro, reduz à metade", "CD 17"},
		{"sem o poder vale a Destreza", false, 5, 3, "acido", "Ácido",
			"Reflexos CD 17 reduz à metade", "Granadeiro"},
		{"com o poder e a Destreza maior", true, 2, 3, "acido", "Ácido",
			"Reflexos CD 17 reduz à metade", "Granadeiro"},
		{"a água benta não é alquímica", true, 5, 3, "agua-benta", "Água benta",
			"Reflexos CD", "Granadeiro"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, goblin := doseAttackOnTurn(t)
			asClass(t, f, "Inventor")
			powers := `["class.inventor.alquimista-iniciado","class.inventor.alquimista-de-batalha"]`
			if c.grenadier {
				powers = `["class.inventor.alquimista-iniciado","class.inventor.alquimista-de-batalha","class.inventor.granadeiro"]`
			}
			withPowers(t, f, powers, "{}")
			if _, err := f.s.db.ExecContext(context.Background(),
				"UPDATE characters SET intelligence = ?, dexterity = ? WHERE id = ?", c.intel, c.dex, f.charID); err != nil {
				t.Fatalf("pôr os atributos: %v", err)
			}
			weapon := weaponWithInjection(t, f)
			loaded(t, f, weapon, c.catalog, c.item)
			attacks(t, f, goblin, "")

			gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
			if !strings.Contains(gm, c.want) {
				t.Errorf("a faixa do mestre não diz %q", c.want)
			}
			if strings.Contains(gm, c.refuse) {
				t.Errorf("a faixa do mestre diz %q", c.refuse)
			}
		})
	}
}
