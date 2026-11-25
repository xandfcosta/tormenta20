package character

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/live"
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
	// Kind é o que esta rolagem é, e Moment, Option e Charges o que o tipo diz além da magia: o momento do dado da
	// vez, a opção da ação da magia, ou a ação e as cargas que a conjuração criou (ver `live.SpellCastRoll`).
	Kind    live.SpellBandKind
	Moment  string
	Option  string
	Charges int
	// Lingering é a marca que a magia deixa em quem foi atingido, com os aumentos escolhidos, nil na que não deixa
	// nada (ver GLOSSARY, **marca na fila**). Quem conjurou entra na publicação.
	Lingering *live.QueueMark
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
	cast, err := spellCastOf(dto, onSheet)
	if err != nil {
		return SpellCast{}, err
	}
	roll, err := onSheet.CastRoll(augments)
	if err != nil {
		return SpellCast{}, err
	}
	cast.ByText = roll.ByText
	// A CONJURAÇÃO QUE CRIA CARGAS não rola nada: a faixa diz o que ficou no lugar ("Dardo de lava ×4").
	if roll.Charges > 0 {
		cast.Option, cast.Charges = onSheet.Actions(augments).Options[0].Name, roll.Charges
	}
	if cast.Parts, err = p.rollTheParts(cast.Name, roll.Parts, fromTheTable); err != nil {
		return SpellCast{}, err
	}
	// OS DARDOS DA LANÇA valem "em vez do efeito normal": a conjuração que só os cria não atinge ninguém, e a marca
	// sai de cada dardo (`ActOnSpell`).
	if roll.Charges == 0 {
		if cast.Lingering, err = lingeringOf(onSheet, augments, cast); err != nil {
			return SpellCast{}, err
		}
	}
	return cast, nil
}

// lingeringOf é a marca que a magia deixa em quem foi atingido, com os aumentos escolhidos ao conjurar, ou nil na
// magia que não deixa nada (ver GLOSSARY, **marca na fila**). A resistência vai junto só na que se repete na vez do
// alvo (a Vontade da Ilusão Lacerante).
func lingeringOf(onSheet sheet.SpellOnTheSheet, picks []sheet.AugmentPick, cast SpellCast) (*live.QueueMark, error) {
	lingering := onSheet.Spell.Lingering
	if lingering == nil {
		return nil, nil
	}
	parts, err := onSheet.LingeringRoll(picks)
	if err != nil {
		return nil, err
	}
	mark := &live.QueueMark{
		Spell: cast.Name, At: lingering.At, Counted: lingering.Turns > 0, TurnsLeft: lingering.Turns,
		Burning: lingering.Burning,
	}
	for _, part := range parts {
		mark.Parts = append(mark.Parts, live.QueueMarkPart{Dice: part.Dice, Kind: part.Kind, Note: part.Note})
	}
	if lingering.RepeatsSave {
		mark.Save, mark.DC = cast.Save, cast.DC
	}
	return mark, nil
}

// spellCastOf é a faixa da magia sem o dado: o nome, a resistência com a CD de quem conjura e o alcance.
func spellCastOf(dto sheet.CharacterDTO, onSheet sheet.SpellOnTheSheet) (SpellCast, error) {
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
	return cast, nil
}

// rollTheParts rola cada parcela, ou confere o total que a mesa disse para ela: o total que os dados não tiram é
// recusado, e a mesa que mandou um número de totais que não é o de parcelas também.
func (p Plays) rollTheParts(name string, parts []book.SpellRoll, fromTheTable []int) ([]RolledSpellPart, error) {
	if fromTheTable != nil && len(fromTheTable) != len(parts) {
		return nil, fmt.Errorf("%s rola %d parcela(s) aqui, e a mesa mandou %d total(is)", name, len(parts), len(fromTheTable))
	}
	rolled := make([]RolledSpellPart, 0, len(parts))
	for i, part := range parts {
		dice, err := engine.ParseDiceSum(part.Dice)
		if err != nil {
			return nil, err
		}
		one := RolledSpellPart{SpellRoll: part}
		var told *int
		if fromTheTable != nil {
			told, one.ByHand = &fromTheTable[i], true
		}
		if one.Rolls, one.Total, err = dice.RollOrTake(told, p.rollDie); err != nil {
			return nil, err
		}
		rolled = append(rolled, one)
	}
	return rolled, nil
}
