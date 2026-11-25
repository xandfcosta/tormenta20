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
	// Bonuses são as parcelas fixas que os poderes de quem conjurou somam ao dado, cada uma com o nome do poder (ver
	// GLOSSARY, **dado da magia**).
	Bonuses []sheet.SpellBonus
	// VitalShare são os PV que outra criatura recupera pela Comunhão Vital, a metade da cura (p58); zero sem ela.
	VitalShare int
}

// RolledSpellPart é uma parcela do dado da magia, rolada no app ou dita pela mesa.
type RolledSpellPart struct {
	book.SpellRoll
	Rolls  []int
	Total  int
	ByHand bool
	// Maximized diz que a parcela não rolou: o Espírito dos Solstícios a pôs no máximo (p62).
	Maximized bool
}

// rollTheCast rola o dado da conjuração, ou confere o total que a mesa rolou na mão para cada parcela (o dado da
// mesa é de primeira classe). Ela recusa o total que os dados não tiram e a mesa que mandou um número de
// parcelas que não é o da conjuração — antes de qualquer cobrança.
func (p Plays) rollTheCast(
	dto sheet.CharacterDTO, onSheet sheet.SpellOnTheSheet, augments []sheet.AugmentPick,
	powerAugments []sheet.PowerAugmentPick, fromTheTable []int,
) (SpellCast, error) {
	cast, powers, err := spellCastOf(dto, onSheet)
	if err != nil {
		return SpellCast{}, err
	}
	roll, err := onSheet.CastRoll(augments)
	if err != nil {
		return SpellCast{}, err
	}
	// OS PODERES QUE SÃO APRIMORAMENTO agem no dado antes de ele rolar: a Magia Sagrada troca o tipo do dano, e é
	// por isso que o frio que virou luz não ganha o Aspecto do Inverno (ver GLOSSARY, **dado da magia**).
	god := ""
	if dto.God != nil {
		god = *dto.God
	}
	powered, err := sheet.PoweredCastOf(powerAugments, book.GodEnergy(god))
	if err != nil {
		return SpellCast{}, err
	}
	roll.Parts = powered.Apply(roll.Parts)
	cast.ByText = roll.ByText
	// A CONJURAÇÃO QUE CRIA CARGAS não rola nada: a faixa diz o que ficou no lugar ("Dardo de lava ×4").
	if roll.Charges > 0 {
		cast.Option, cast.Charges = onSheet.Actions(augments).Options[0].Name, roll.Charges
	}
	if powered.Maximize {
		cast.Parts, err = maximizedParts(roll.Parts)
	} else {
		cast.Parts, err = p.rollTheParts(cast.Name, roll.Parts, fromTheTable)
	}
	if err != nil {
		return SpellCast{}, err
	}
	if powered.VitalShare {
		cast.VitalShare = halfTheHealing(cast.Parts)
	}
	// OS DARDOS DA LANÇA valem "em vez do efeito normal": a conjuração que só os cria não atinge ninguém, e a marca
	// sai de cada dardo (`ActOnSpell`).
	if roll.Charges == 0 {
		cast.Bonuses = powers.CastBonuses(roll.Parts)
		if cast.Lingering, err = lingeringOf(onSheet, augments, cast, powers); err != nil {
			return SpellCast{}, err
		}
	}
	return cast, nil
}

// lingeringOf é a marca que a magia deixa em quem foi atingido, com os aumentos escolhidos ao conjurar, ou nil na
// magia que não deixa nada (ver GLOSSARY, **marca na fila**). A resistência vai junto só na que se repete na vez do
// alvo (a Vontade da Ilusão Lacerante).
func lingeringOf(
	onSheet sheet.SpellOnTheSheet, picks []sheet.AugmentPick, cast SpellCast, powers sheet.SpellRollPowers,
) (*live.QueueMark, error) {
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
	mark.Bonuses = LiveBonuses(powers.LaterBonuses(parts))
	if lingering.RepeatsSave {
		mark.Save, mark.DC = cast.Save, cast.DC
	}
	return mark, nil
}

// spellCastOf é a faixa da magia sem o dado: o nome, a resistência com a CD de quem conjura e o alcance. Ela
// devolve junto os poderes de quem conjura que somam ao dado (`sheet.SpellRollPowers`), da mesma ficha computada.
func spellCastOf(dto sheet.CharacterDTO, onSheet sheet.SpellOnTheSheet) (SpellCast, sheet.SpellRollPowers, error) {
	cast := SpellCast{
		Name: onSheet.Spell.Name, Save: book.SpellResistanceWritten(onSheet.Spell),
		Range: strings.ToLower(book.SpellRangeWritten(onSheet.Spell)),
	}
	computed, err := sheet.Compute(dto)
	if err != nil {
		return SpellCast{}, sheet.SpellRollPowers{}, fmt.Errorf("computar a ficha para %s: %w", cast.Name, err)
	}
	if cast.Save != "" {
		cast.DC, _ = onSheet.DC(computed)
	}
	return cast, sheet.SpellRollPowersOf(dto, computed), nil
}

// LiveBonuses leva as parcelas dos poderes para a faixa e a marca da mesa.
func LiveBonuses(bonuses []sheet.SpellBonus) []live.SpellCastBonus {
	var out []live.SpellCastBonus
	for _, b := range bonuses {
		out = append(out, live.SpellCastBonus{Name: b.Name, Amount: b.Amount, Kind: b.Kind, Note: b.Note})
	}
	return out
}

// maximizedParts põe cada parcela no máximo, sem rolar e sem o total da mão: "uma magia Curar Ferimentos
// aprimorada para curar 5d8+5 PV irá curar automaticamente 45 PV" (p62).
func maximizedParts(parts []book.SpellRoll) ([]RolledSpellPart, error) {
	out := make([]RolledSpellPart, 0, len(parts))
	for _, part := range parts {
		dice, err := engine.ParseDiceSum(part.Dice)
		if err != nil {
			return nil, err
		}
		_, highest := dice.Range()
		out = append(out, RolledSpellPart{SpellRoll: part, Total: highest, Maximized: true})
	}
	return out, nil
}

// halfTheHealing é a metade da cura rolada, que a Comunhão Vital dá a outra criatura (p58), arredondada para baixo.
func halfTheHealing(parts []RolledSpellPart) int {
	healing := 0
	for _, part := range parts {
		if part.Kind == "cura" {
			healing += part.Total
		}
	}
	return healing / 2
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
