package book

import (
	"slices"
	"testing"
)

// O TIPO DE CRIATURA das raças (ALE-469), com o vocabulário do bestiário.
// Sete raças jogáveis declaram tipo próprio no livro ("Você é uma criatura do
// tipo…", p24-31); as outras são humanoides. A Memória Póstuma (p29) escolhe
// entre as humanoides (ALE-504).
func TestEveryRaceHasACreatureType(t *testing.T) {
	known := []string{"humanoide", "monstro", "construto", "espirito", "morto-vivo"}
	exceptions := map[string]string{
		"lefou": "monstro", "golem": "construto", "medusa": "monstro", "osteon": "morto-vivo",
		"silfide": "espirito", "suraggel": "espirito", "trog": "monstro",
	}
	measured := 0
	for key, race := range RaceTraitsByKey() {
		if key != race.ID {
			continue // o índice também é por nome
		}
		measured++
		if !slices.Contains(known, race.Type) {
			t.Errorf("%s tem tipo %q, e os tipos do bestiário são %v", race.ID, race.Type, known)
		}
		if want, special := exceptions[race.ID]; special && race.Type != want {
			t.Errorf("%s é %s no livro (p24-31), e o catálogo diz %q", race.ID, want, race.Type)
		}
	}
	if measured < 17 {
		t.Fatalf("mediu %d raças, e o livro tem 17", measured)
	}
}
