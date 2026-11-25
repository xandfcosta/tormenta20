package live

import "testing"

// A FAIXA DA MAGIA DIZ DE QUE TIPO É A ROLAGEM (ver GLOSSARY, **rolagem informada**), e a cabeça sai do tipo: a
// mesma frase de antes para a conjuração, o dado da vez, a ação da magia, a marca na fila e a condição Em Chamas,
// que deixa de vestir os campos de magia.
func TestTheSpellBandHeadComesFromTheKindOfRoll(t *testing.T) {
	for _, c := range []struct {
		why  string
		roll SpellCastRoll
		want string
	}{
		{"a conjuração", SpellCastRoll{Who: "Arwen", Spell: "Bola de Fogo"}, "Arwen · Bola de Fogo"},
		{"a conjuração que cria cargas", SpellCastRoll{Who: "Arwen", Spell: "Lança Ígnea de Aleph", Option: "Dardo de lava", Charges: 4},
			"Arwen · Lança Ígnea de Aleph · Dardo de lava ×4"},
		{"o dado da vez", SpellCastRoll{Kind: RollOfTheTurn, Who: "Arwen", Spell: "Enxame de Pestes", Moment: "fim"},
			"Arwen · Enxame de Pestes · fim do turno"},
		{"a ação da magia", SpellCastRoll{Kind: RollOfTheAction, Who: "Arwen", Spell: "Mão Poderosa de Talude", Option: "Esmagar"},
			"Arwen · Mão Poderosa de Talude · Esmagar"},
		{"a marca na fila", SpellCastRoll{Kind: RollOfTheMark, Who: "Arwen", Spell: "Flecha Ácida", Moment: "inicio", Target: "Ogro"},
			"Arwen · Flecha Ácida · início do turno de Ogro"},
		{"a condição Em Chamas", SpellCastRoll{Kind: RollOfTheCondition, Condition: "Em Chamas", Moment: "inicio", Target: "Ogro"},
			"Ogro · Em Chamas · início do turno"},
	} {
		if got := c.roll.Head(); got != c.want {
			t.Errorf("%s: a cabeça diz %q; want %q", c.why, got, c.want)
		}
	}
}
