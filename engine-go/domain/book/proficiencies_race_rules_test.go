package book

import (
	"slices"
	"testing"
)

// p28 Vanguardista: "Você recebe proficiência em armas de fogo". A raça entra
// no PADRÃO da ficha como a classe entra (ALE-493).
func TestTheKlirenStartsWithFirearms(t *testing.T) {
	if got := GrantedProficiencies([]string{"Inventor"}, []string{"kliren"}); !slices.Contains(got, "armas-de-fogo") {
		t.Errorf("o kliren inventor nasce com %v, sem armas de fogo", got)
	}
	if got := GrantedProficiencies([]string{"Inventor"}, []string{"humano"}); slices.Contains(got, "armas-de-fogo") {
		t.Errorf("controle: o humano inventor nasce com armas de fogo (%v) — o caso não distingue a raça", got)
	}
}
