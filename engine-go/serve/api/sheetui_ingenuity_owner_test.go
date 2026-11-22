package api

import (
	"context"
	"regexp"
	"testing"
)

// A ENGENHOSIDADE RECEBIDA DUAS VEZES custa 1 PM a menos — "caso receba esta habilidade novamente, seu custo é
// reduzido em –1 PM" (kliren, p28) —, e só quem a recebe duas vezes é o kliren inventor. O inventor que não é kliren
// ganhava o corte porque o nome igual o fazia dono da Engenhosidade da raça (ALE-678).
func TestOnlyTheKlirenInventorPaysOneForIngenuity(t *testing.T) {
	// O "custa N PM ao rolar" do diálogo do teste: a mesma expressão aparece em outros lugares com outro número.
	price := regexp.MustCompile(`custa <b data-text="\(0 \+ \(\$expertise_spends\.engenhosidade &gt; 0 \? (\d+) : 0\)\)`)
	for _, c := range []struct {
		name   string
		kliren bool
		want   string
	}{
		{"o inventor que não é kliren", false, "2"},
		{"o kliren inventor", true, "1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newSceneFixture(t)
			asClass(t, f, "Inventor")
			withIntelligence(t, f, 3)
			if c.kliren {
				if _, err := f.s.db.ExecContext(context.Background(),
					"INSERT INTO character_races (characterId, race) VALUES (?, 'kliren')", f.charID); err != nil {
					t.Fatalf("fazer da ficha kliren: %v", err)
				}
			}
			found := price.FindStringSubmatch(sheetTab(t, f, f.charID, "expertises"))
			if found == nil || found[1] != c.want {
				t.Errorf("%s paga %v PM na Engenhosidade; want %s", c.name, found, c.want)
			}
		})
	}
}
