package api

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// ALQUIMISTA DE BATALHA (p68): "quando usa um preparado alquímico ou poção que cause dano, você soma
// sua Inteligência na rolagem de dano".
//
// No app o preparado se usa carregado na arma pela Injeção alquímica, e a dose liberada CONTA como
// usá-lo (decisão do dono). A Água benta causa dano mas é equipamento de aventura (p155), e não
// preparado (p160): ela não ganha a Inteligência. A ficha é de nível 8 com Inteligência 4.
func TestTheBattleAlchemistAddsIntelligenceToThePreparationDose(t *testing.T) {
	cases := []struct {
		catalog, name string
		want, refuse  string
	}{
		{"acido", "Ácido", "2d4+4 de ácido", "2d4 de ácido"},
		{"bomba", "Bomba", "6d6+4 de impacto", "6d6 de impacto"},
		{"fogo-alquimico", "Fogo alquímico", "1d6+4 de fogo", "1d6 de fogo"},
		{"agua-benta", "Água benta", "2d10 de luz", "2d10+4"},
	}
	for _, c := range cases {
		t.Run(c.catalog, func(t *testing.T) {
			f, goblin := doseAttackOnTurn(t)
			battleAlchemist(t, f)
			weapon := weaponWithInjection(t, f)
			loaded(t, f, weapon, c.catalog, c.name)
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

// battleAlchemist faz da ficha da bancada um INVENTOR com o Alquimista de Batalha e o pré-requisito
// dele, e Inteligência 4: o poder de classe só vale para quem tem a classe.
func battleAlchemist(t *testing.T, f sceneFixture) {
	t.Helper()
	ctx := context.Background()
	if _, err := f.s.db.ExecContext(ctx,
		"UPDATE character_level_choices SET ref = 'Inventor' WHERE characterId = ? AND kind = 'class'", f.charID); err != nil {
		t.Fatalf("fazer da ficha um inventor: %v", err)
	}
	withPowers(t, f, `["class.inventor.alquimista-iniciado","class.inventor.alquimista-de-batalha"]`, "{}")
	if _, err := f.s.db.ExecContext(ctx, "UPDATE characters SET intelligence = 4 WHERE id = ?", f.charID); err != nil {
		t.Fatalf("pôr a Inteligência: %v", err)
	}
}

// Sem o poder, o dano do preparado não soma atributo nenhum: é o controle do caso acima.
func TestWithoutTheBattleAlchemistThePreparationDoseKeepsItsDice(t *testing.T) {
	f, goblin := doseAttackOnTurn(t)
	if _, err := f.s.db.ExecContext(context.Background(),
		"UPDATE characters SET intelligence = 4 WHERE id = ?", f.charID); err != nil {
		t.Fatalf("pôr a Inteligência: %v", err)
	}
	weapon := weaponWithInjection(t, f)
	loaded(t, f, weapon, "acido", "Ácido")
	attacks(t, f, goblin, "")

	gm := f.requests(t, f.gm, http.MethodGet, f.tableUrl(), "").Body.String()
	if !strings.Contains(gm, "2d4 de ácido") || strings.Contains(gm, "2d4+4") {
		t.Error("sem o Alquimista de Batalha a dose de Ácido devia dizer só 2d4 de ácido")
	}
}
