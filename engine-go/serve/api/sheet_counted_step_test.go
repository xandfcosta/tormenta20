package api

import (
	"testing"

	"t20engine/domain/book"
)

// O PASSO DA ESCOLHA CONTADA, na lista do Escolher, diz numa palavra o que se escolhe — como os outros
// passos dizem "benefícios" e "poderes" —, e a descrição do poder fica no bloco, onde a pessoa escolhe.
// Na lista, a descrição inteira empurrava os outros passos para baixo.
func TestTheCountedStepSaysWhatIsChosenAndTheBlockSaysThePower(t *testing.T) {
	cases := []struct {
		name, step, field, power, meta string
		character                      func(t *testing.T) (sceneFixture, int64)
	}{
		{"as perícias do Especialista", "Ladino · Especialista", "Especialista", "class.ladino.especialista", "perícias",
			func(t *testing.T) (sceneFixture, int64) { return specialist(t, 2) }},
		{"os terrenos do Explorador", "Caçador · Explorador", "Explorador", "class.cacador.explorador", "terrenos",
			func(t *testing.T) (sceneFixture, int64) { return hunter(t, 7) }},
		{"as escolas do bardo", "Bardo · Magias (1° círculo)", "Magias (1° círculo)", "class.bardo.magias-1-circulo", "escolas",
			func(t *testing.T) (sceneFixture, int64) { return schoolCaster(t, "Bardo") }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, id := c.character(t)
			screen := powerScreen(t, f, id)
			if step := choiceStepTitled(t, screen, c.step); step.Meta != c.meta {
				t.Errorf("o passo %q diz %q na lista, quer %q", c.step, step.Meta, c.meta)
			}
			field, found := choiceFieldIn(t, screen, c.step, c.field)
			if !found {
				t.Fatalf("o passo %q não tem o bloco %q", c.step, c.field)
			}
			if want := book.ClassPowers()[c.power].Description; want == "" || field.Detail != want {
				t.Errorf("o bloco %q diz %q embaixo do cabeçalho, quer a descrição do poder %q", c.field, field.Detail, want)
			}
		})
	}
}
