package sheet

import (
	"slices"
	"strings"
	"testing"

	"t20engine/domain/book"
)

// O APLICAR OFERECE AS DURAÇÕES QUE O BÔNUS PODE TER (ver GLOSSARY, **efeito da magia**): a do verbete e a que o
// aprimoramento dá. A esfera do Campo de Força troca o efeito e não dá os PV temporários (p183).
func TestTheBuffLastsAsTheAugmentSays(t *testing.T) {
	for _, c := range []struct {
		spell string
		want  []string
	}{
		{"velocidade", []string{"sustained", "scene"}},
		{"oracao", []string{"sustained", "scene"}},
		{"camuflagem-ilusoria", []string{"scene", "sustained"}},
		{"campo-de-forca", []string{"scene"}},
		{"armadura-arcana", []string{"scene"}},
	} {
		spell, known := book.SpellByID(c.spell)
		if !known {
			t.Fatalf("a magia %q não existe no livro", c.spell)
		}
		got, err := BuffDurations(spell)
		if err != nil || !slices.Equal(got, c.want) {
			t.Errorf("%s dura %v (%v); want %v", c.spell, got, err, c.want)
		}
	}
}

// TIRAR DA SUSTENTADA É IR PARA A CENA, em toda magia com bônus: é o que o `BuffDurations` lê da marca
// `Sustains: false`, que não diz para onde vai. Um aprimoramento que mude para outra duração quer a dele escrita.
func TestEveryBuffAugmentThatLeavesTheSustainedGoesToTheScene(t *testing.T) {
	measured := 0
	for _, m := range book.Catalogs().Spells {
		spell, _ := book.SpellByID(m.ID)
		if spell.Buff == nil {
			continue
		}
		for i, augment := range spell.Augments {
			if augment.Sustains == nil || *augment.Sustains || augment.ReplacesTheBuff {
				continue
			}
			measured++
			if !strings.Contains(strings.ToLower(augment.Description), "duração para cena") {
				t.Errorf("%s, aprimoramento %d, tira a magia da sustentada para outra duração que não a cena, e o "+
					"Aplicar a ofereceria de cena: %q", spell.ID, i, augment.Description)
			}
		}
	}
	if measured == 0 {
		t.Fatal("nenhuma magia com bônus sai da sustentada: a varredura está olhando o lugar errado")
	}
}
