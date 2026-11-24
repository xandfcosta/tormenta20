package character

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// SpellCast é a magia conjurada com o dado dela rolado (ver GLOSSARY, **dado da magia**): as parcelas, ou ByText
// quando um aprimoramento mudou o dado para um que não sai na conjuração, e a resistência com a CD de quem
// conjurou, para a faixa da mesa. Quem passou e o que se aplica é do mestre.
type SpellCast struct {
	Name   string
	Parts  []RolledSpellPart
	ByText bool
	Save   string
	DC     int
	Range  string
}

// RolledSpellPart é uma parcela do dado da magia, rolada no app ou dita pela mesa.
type RolledSpellPart struct {
	book.SpellRoll
	Rolls  []int
	Total  int
	ByHand bool
}

// rollTheCast rola o dado da conjuração, ou confere o total que a mesa rolou na mão para cada parcela (o dado da
// mesa é de primeira classe). Ela recusa o total que os dados não tiram e a mesa que mandou um número de
// parcelas que não é o da conjuração — antes de qualquer cobrança.
func (p Plays) rollTheCast(
	dto sheet.CharacterDTO, onSheet sheet.SpellOnTheSheet, augments []sheet.AugmentPick, fromTheTable []int,
) (SpellCast, error) {
	cast := SpellCast{
		Name: onSheet.Spell.Name, Save: book.SpellResistanceWritten(onSheet.Spell),
		Range: strings.ToLower(book.SpellRangeWritten(onSheet.Spell)),
	}
	if cast.Save != "" {
		computed, err := sheet.Compute(dto)
		if err != nil {
			return SpellCast{}, fmt.Errorf("computar a ficha para a CD de %s: %w", cast.Name, err)
		}
		cast.DC, _ = onSheet.DC(computed)
	}
	roll, err := onSheet.CastRoll(augments)
	if err != nil {
		return SpellCast{}, err
	}
	cast.ByText = roll.ByText
	if fromTheTable != nil && len(fromTheTable) != len(roll.Parts) {
		return SpellCast{}, fmt.Errorf("%s rola %d parcela(s) nesta conjuração, e a mesa mandou %d total(is)",
			cast.Name, len(roll.Parts), len(fromTheTable))
	}
	for i, part := range roll.Parts {
		dice, err := engine.ParseDiceSum(part.Dice)
		if err != nil {
			return SpellCast{}, err
		}
		rolled := RolledSpellPart{SpellRoll: part}
		if fromTheTable != nil {
			lowest, highest := dice.Range()
			if fromTheTable[i] < lowest || fromTheTable[i] > highest {
				return SpellCast{}, fmt.Errorf("%s tira de %d a %d, e a mesa disse %d", part.Dice, lowest, highest, fromTheTable[i])
			}
			rolled.Total, rolled.ByHand = fromTheTable[i], true
		} else if rolled.Rolls, rolled.Total, err = dice.RollWith(p.rollDie); err != nil {
			return SpellCast{}, err
		}
		cast.Parts = append(cast.Parts, rolled)
	}
	return cast, nil
}
