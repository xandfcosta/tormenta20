package engine

import "testing"

// The per-spell PM limit is the rule the cast gate hangs on, and it was WRONG:
// the Nest service this was ported from cited "PDF p171 — ½ nível", but p171
// says no such thing — it points at p224, which reads:
//
//	"o máximo de PM que você pode gastar por uso é igual ao seu nível NA CLASSE
//	 que fornece a habilidade […] Para habilidades de raça, origem ou outras
//	 fontes e poderes gerais, o limite é o seu nível de personagem."
//
// And p171's own worked example: "Um arcanista de 11º nível pode gastar até 11
// PM ao lançar essa magia" (Bola de Fogo). Full level, not half.
func TestSpellPmLimit(t *testing.T) {
	char := func(level int, classes ...CharacterClass) Character {
		return Character{Level: level, Classes: classes}
	}
	arcanista12 := char(12, CharacterClass{ClassName: "Arcanista", Level: 12})

	t.Run("nível NA CLASSE que fornece a magia, cheio", func(t *testing.T) {
		// p171: um arcanista de 11º gasta até 11 PM — o /2 antigo dava 5.
		if got := SpellPmLimit(char(11, CharacterClass{ClassName: "Arcanista", Level: 11}), 0, []string{"Arcanista"}); got != 11 {
			t.Errorf("limite = %d, want 11", got)
		}
		if got := SpellPmLimit(arcanista12, 0, []string{"Arcanista"}); got != 12 {
			t.Errorf("limite = %d, want 12", got)
		}
	})

	// Multiclasse: a magia vem da lista do Arcanista, então vale o nível de
	// Arcanista — não o nível de personagem nem o da classe maior.
	t.Run("multiclasse conta só a classe que fornece", func(t *testing.T) {
		bardoArcanista := char(10, CharacterClass{ClassName: "Guerreiro", Level: 9}, CharacterClass{ClassName: "Arcanista", Level: 1})

		if got := SpellPmLimit(bardoArcanista, 0, []string{"Arcanista"}); got != 1 {
			t.Errorf("limite = %d, want 1", got)
		}
	})

	// Magia concedida por raça/origem/poder não vem de classe nenhuma: o teto é
	// o nível de personagem (p224).
	t.Run("fonte que não é classe usa o nível de personagem", func(t *testing.T) {
		barbaro8 := char(8, CharacterClass{ClassName: "Bárbaro", Level: 8})

		if got := SpellPmLimit(barbaro8, 0, []string{"Arcanista"}); got != 8 {
			t.Errorf("limite = %d, want 8", got)
		}
	})

	t.Run("bônus de item soma ao teto", func(t *testing.T) {
		if got := SpellPmLimit(arcanista12, 2, []string{"Arcanista"}); got != 14 {
			t.Errorf("limite = %d, want 14", got)
		}
	})

	t.Run("nunca abaixo de 1", func(t *testing.T) {
		if got := SpellPmLimit(char(0), 0, nil); got != 1 {
			t.Errorf("limite = %d, want 1", got)
		}
	})
}
