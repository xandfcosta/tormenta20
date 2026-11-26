package book

import "testing"

// A NATUREZA DE UM PERSONAGEM NA MESA é a raça e o TIPO dela (ALE-650): o osteon é morto-vivo, o golem é
// construto (p27, p29). Sem o tipo, uma Anticriatura "contra morto-vivo" num osteon jogador sai como
// "desconhecido", e a imunidade do tipo (p228) não o alcança.
func TestACharacterNatureCarriesTheRaceType(t *testing.T) {
	for race, want := range map[string]string{"osteon": "morto-vivo", "Osteon": "morto-vivo", "golem": "construto", "humano": "humanoide"} {
		if got := CharacterNature(race, 0); got.Kind != want || got.Race != race || got.Intelligence == nil {
			t.Errorf("a natureza de %q é %+v; want o tipo %q e a raça", race, got, want)
		}
	}
	if got := TargetOfTheChosenKind("morto-vivo", CharacterNature("osteon", 0)); got != TargetIsTheChosenKind {
		t.Errorf("a Anticriatura contra morto-vivo lê o osteon jogador como %q", got)
	}
	if got := TargetOfTheChosenKind("morto-vivo", CharacterNature("humano", 0)); got != TargetIsNotTheChosenKind {
		t.Errorf("a Anticriatura contra morto-vivo lê o humano jogador como %q", got)
	}
}
