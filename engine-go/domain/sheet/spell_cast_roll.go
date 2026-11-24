package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// CastRoll é o dado que esta conjuração rola (ver GLOSSARY, **dado da magia**): as parcelas, ou ByText quando um
// aprimoramento escolhido muda o dado para um que não sai na conjuração — aí quem joga rola pelo texto, e o app
// não rola o dado errado.
type CastRoll struct {
	Parts  []book.SpellRoll
	ByText bool
}

// CastRoll monta o dado da conjuração com os aprimoramentos escolhidos: o que MUDA troca as parcelas da magia, e o
// que AUMENTA soma, por repetição, à parcela do mesmo tipo; o aumento sem tipo fica à parte (os +2d6 da Lança
// Ígnea, p196). A magia que não rola na conjuração devolve nada.
//
// Índice fora de faixa é ignorado de propósito: quem recusa o pedido malformado é o `AugmentPrice`.
//
//	sheet.CasterOf(dto).Spell("bola-de-fogo").CastRoll([]sheet.AugmentPick{{AugmentIndex: 0, Stacks: 2}}) // 10d6 de fogo
func (s SpellOnTheSheet) CastRoll(picks []AugmentPick) (CastRoll, error) {
	parts := slices.Clone(s.Spell.Rolls)
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) {
			continue
		}
		if a := s.Spell.Augments[p.AugmentIndex]; a.ChangesTheRoll {
			if len(a.Rolls) == 0 {
				return CastRoll{ByText: true}, nil
			}
			parts = slices.Clone(a.Rolls)
		}
	}
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) || p.Stacks < 1 {
			continue
		}
		for _, add := range s.Spell.Augments[p.AugmentIndex].Adds {
			var err error
			if parts, err = addToTheRoll(parts, add, p.Stacks); err != nil {
				return CastRoll{}, fmt.Errorf("%s, aprimoramento %d: %w", s.Spell.Name, p.AugmentIndex, err)
			}
		}
	}
	return CastRoll{Parts: parts}, nil
}

// addToTheRoll soma `add` `times` vezes a toda parcela do mesmo tipo; o sem tipo fica à parte, e o tipado sem
// parcela do tipo não soma nada. A nota do aumento vai junto da parcela, com as vezes: "+1d12 em mortos-vivos ×2".
func addToTheRoll(parts []book.SpellRoll, add book.SpellRoll, times int) ([]book.SpellRoll, error) {
	more, err := engine.ParseDiceSum(add.Dice)
	if err != nil {
		return nil, err
	}
	matched := false
	for i, part := range parts {
		if part.Kind != add.Kind || add.Kind == "" {
			continue
		}
		base, err := engine.ParseDiceSum(part.Dice)
		if err != nil {
			return nil, err
		}
		sum, err := base.Plus(more, times)
		if err != nil {
			return nil, err
		}
		parts[i].Dice = sum.String()
		parts[i].Note = joinedNote(part.Note, add.Note, times)
		matched = true
	}
	// O aumento TIPADO sem parcela do tipo não cria uma: ele soma à parcela que outro aprimoramento traz (o +1d8 de
	// cura do Raio Solar só existe com a cura do aprimoramento 2, p203). Só o sem tipo fica à parte.
	if matched || add.Kind != "" {
		return parts, nil
	}
	alone, err := engine.ParseDiceSum(add.Dice)
	if err != nil {
		return nil, err
	}
	alone.Count, alone.Bonus = alone.Count*times, alone.Bonus*times
	return append(parts, book.SpellRoll{Dice: alone.String(), Kind: add.Kind, Note: add.Note}), nil
}

func joinedNote(note, added string, times int) string {
	if added == "" {
		return note
	}
	if times > 1 {
		added = fmt.Sprintf("%s ×%d", added, times)
	}
	if note == "" {
		return added
	}
	return note + "; " + added
}
