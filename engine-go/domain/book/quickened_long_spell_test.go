package book

import "testing"

// "Você só pode aplicar este aprimoramento em magias com execução de movimento, padrão ou completa"
// (p131). A execução LONGA o catálogo guarda como "completa", que é o que ela cobra do turno, com o
// texto do livro na nota, e a nota é a execução de verdade.
func TestTheQuickenedSpellRefusesALongExecution(t *testing.T) {
	spellByID := func(id string) Spell {
		for _, spell := range Catalogs().Spells {
			if spell.ID == id {
				return spell
			}
		}
		t.Fatalf("o catálogo não tem a magia %q", id)
		return Spell{}
	}
	if bar := PowerAugmentBar(QuickenedSpellPower, spellByID("bola-de-fogo")); bar != "" {
		t.Fatalf("o controle já estava errado: a Bola de Fogo foi barrada: %q", bar)
	}
	if bar := PowerAugmentBar(QuickenedSpellPower, spellByID("runa-de-protecao")); bar == "" {
		t.Error("a Magia Acelerada passou na Runa de Proteção, que leva 1 hora")
	}
}
