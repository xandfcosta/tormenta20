package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// BuffDurations são as durações que o bônus da magia pode ter na ficha de quem o recebe, na grafia da coluna
// `scope`: a do verbete primeiro, e depois a que um aprimoramento dá ao pôr a magia na sustentada ou tirá-la dela
// (ver GLOSSARY, **efeito da magia**). O Aplicar da aba Efeitos oferece uma por linha e recusa a que não está aqui.
//
// Tirar da sustentada é ir para a CENA: é o que diz todo aprimoramento de magia com bônus que tira, e quem prende
// isso ao texto é o `TestEveryBuffAugmentThatLeavesTheSustainedGoesToTheScene`. O aprimoramento que troca o efeito
// inteiro não conta (`book.SpellAugment.ReplacesTheBuff`).
//
//	sheet.BuffDurations(velocidade) // ["sustained", "scene"], nil
func BuffDurations(spell book.Spell) ([]string, error) {
	if spell.Buff == nil {
		return nil, fmt.Errorf("a magia %q não deixa efeito para aplicar", spell.ID)
	}
	base, err := engine.EffectScope(spell.Duration, spell.DurationNote, spell.Buff.DefaultScope)
	if err != nil {
		return nil, fmt.Errorf("a magia %q: %w", spell.ID, err)
	}
	durations := []string{base}
	for _, augment := range spell.Augments {
		if augment.Sustains == nil || augment.ReplacesTheBuff {
			continue
		}
		changed := engine.Duration{Kind: engine.DurationScene}
		if *augment.Sustains {
			changed = engine.Duration{Kind: engine.DurationSustained}
		}
		if stored := changed.Stored(); !slices.Contains(durations, stored) {
			durations = append(durations, stored)
		}
	}
	return durations, nil
}
