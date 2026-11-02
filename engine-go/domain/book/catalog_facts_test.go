package book

import (
	"regexp"
	"strings"
	"testing"
)

// Todo verbete dos catálogos aponta a página do livro: sem ela, o cartão do mestre não tem o que
// abrir, e a falta não aparece em lugar nenhum.
func TestEveryCatalogEntryHasABookPage(t *testing.T) {
	measured := 0
	for _, p := range FlattenedPowers() {
		measured++
		if p.BookPage == 0 {
			t.Errorf("o poder %s (%s) não tem página", p.ID, p.Source)
		}
	}
	for _, i := range Catalogs().Items {
		measured++
		if i.BookPage == 0 {
			t.Errorf("o item %s não tem página", i.ID)
		}
	}
	if measured < 500 {
		t.Fatalf("só %d verbetes medidos: o catálogo não carregou", measured)
	}
}

// A resistência diz O QUE se testa, como o livro escreve (Relâmpago, p203).
func TestTheSpellSaysWhatResists(t *testing.T) {
	for id, want := range map[string]string{
		"relampago":    "Reflexos reduz à metade",
		"bola-de-fogo": "Reflexos reduz à metade",
		"luz":          "Vontade anula", "armadura-arcana": "",
	} {
		spell, found := SpellByID(id)
		if !found {
			t.Fatalf("a magia %s não está no catálogo", id)
		}
		if got := SpellResistanceWritten(spell); got != want {
			t.Errorf("%s: a resistência saiu %q, e o livro diz %q", id, got, want)
		}
	}
	for _, spell := range Catalogs().Spells {
		if spell.Resistance != "" && SpellResistanceWritten(spell) == spell.Resistance {
			t.Errorf("%s: a resistência %q não diz o que se testa", spell.ID, spell.Resistance)
		}
	}
}

// O pré-requisito sai por extenso, com o nome de quem é pedido, e nunca o id.
func TestThePowerSaysWhatItRequires(t *testing.T) {
	for id, want := range map[string]string{
		"acuidade-com-arma":      "Des 1",
		"class.bucaneiro.aparar": "Esgrimista",
	} {
		if got := PowerRequirements(id); got != want {
			t.Errorf("%s: o pré-requisito saiu %q, e devia ser %q", id, got, want)
		}
	}
	for id := range ClassPowers() {
		if written := PowerRequirements(id); strings.Contains(written, "class.") {
			t.Errorf("%s: o pré-requisito cita um id: %q", id, written)
		}
	}
}

// Toda magia diz sobre o que ela age, nas palavras do livro: o Alvo, a Área ou o Efeito. A forma
// vale para as 198; o valor se prende só onde há armadilha.
func TestEverySpellSaysWhatItActsOn(t *testing.T) {
	shape := regexp.MustCompile(`^(Alvo|Alvos|Área|Efeito|Alvo ou Área): \S`)
	spells := Catalogs().Spells
	for _, spell := range spells {
		if !shape.MatchString(spell.Targeting) {
			t.Errorf("%s (p%d): %q não é um alvo do livro", spell.ID, spell.BookPage, spell.Targeting)
		}
	}
	if len(spells) != 198 {
		t.Fatalf("o catálogo tem %d magias, e o livro tem 198", len(spells))
	}
	for id, want := range map[string]string{
		"relampago":      "Área: linha de 30m",
		"dissipar-magia": "Alvo ou Área: 1 criatura ou 1 objeto mágico ou esfera com 3m de raio",
		"sono":           "Alvo: 1 humanoide",
	} {
		if spell, _ := SpellByID(id); spell.Targeting != want {
			t.Errorf("%s: saiu %q, e o livro diz %q", id, spell.Targeting, want)
		}
	}
}
