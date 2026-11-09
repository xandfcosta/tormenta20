package api

import (
	"strings"
	"testing"
)

// "Empunhar (2 mãos)" é de quem o livro deixa: a arma de duas mãos e a ADAPTÁVEL (p143). A Versátil é
// bônus em manobra, e ocupar as duas mãos com ela não dá nada.
func TestTheBagOffersTheTwoHandedGripToWhoHasIt(t *testing.T) {
	for _, c := range []struct {
		name, weapon string
		one, two     bool
	}{
		{"adaptável", "espada-bastarda", true, true},
		{"adaptável e ágil", "katana", true, true},
		{"versátil", "tridente", true, false},
		{"de uma mão", "espada-longa", true, false},
		{"de duas mãos", "montante", false, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			f, id := fighterFixture(t)
			itemSemeia(t, f, id, c.weapon, c.weapon, "")
			screen := bagScreen(t, f, id)
			if !strings.Contains(screen, "Empunhar") {
				t.Fatalf("o controle falhou: a mochila não oferece empunhar %s", c.weapon)
			}
			if got := strings.Contains(screen, "Empunhar (1 mão)"); got != c.one {
				t.Errorf("uma mão = %v, want %v", got, c.one)
			}
			if got := strings.Contains(screen, "Empunhar (2 mãos)"); got != c.two {
				t.Errorf("duas mãos = %v, want %v", got, c.two)
			}
		})
	}
}
