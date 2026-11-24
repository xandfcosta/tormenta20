package sheetui

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// AS LINHAS DO DADO DA MAGIA NA FOLHA (ver GLOSSARY, **dado da magia**): uma por parcela, com o campo do total
// rolado na mão, e elas ACOMPANHAM os aprimoramentos ligados na tela, como a prévia do custo. A regra é a do
// `SpellOnTheSheet.CastRoll`, que o servidor usa para conferir a mão: o que muda troca as parcelas (o de índice
// maior ligado vence), e o que aumenta soma à parcela do mesmo tipo; o aumento sem tipo ganha a linha dele, na
// posição seguinte às parcelas. A tela só mostra — quem decide é o servidor.

// spellRollLine é uma linha: Static é o texto antes de o Datastar chegar, Text a expressão que o recalcula, Show
// a expressão que a mostra (vazia: sempre), e BornHidden diz que ela nasce escondida. ByText é a linha do
// aprimoramento que muda o dado para um que não sai na conjuração.
type spellRollLine struct {
	Static     string
	Text       string
	Show       string
	BornHidden bool
	ByText     bool
	Note       string
	Signal     string
	Label      string
}

// spellRollLinesOf monta as linhas de todos os modos da magia: o dado dela, e o de cada aprimoramento que o muda.
func spellRollLinesOf(spell book.Spell) []spellRollLine {
	if len(spell.Rolls) == 0 {
		return nil
	}
	var changers []int
	for i, a := range spell.Augments {
		if a.ChangesTheRoll {
			changers = append(changers, i)
		}
	}
	lines := rollModeLines(spell, spell.Rolls, noneOn(changers), false)
	for k, i := range changers {
		active := augmentSignal(i) + " > 0"
		if higher := noneOn(changers[k+1:]); higher != "" {
			active += " && " + higher
		}
		a := spell.Augments[i]
		if len(a.Rolls) == 0 {
			lines = append(lines, spellRollLine{
				Static: fmt.Sprintf("o aprimoramento %d muda o dado para um que não sai na conjuração: role pelo texto dele", i),
				Show:   active, BornHidden: true, ByText: true,
			})
			continue
		}
		lines = append(lines, rollModeLines(spell, a.Rolls, active, true)...)
	}
	return lines
}

// rollModeLines são as linhas de um modo: as parcelas com os aumentos somados, e a do aumento sem tipo.
func rollModeLines(spell book.Spell, parts []book.SpellRoll, show string, bornHidden bool) []spellRollLine {
	lines := []spellRollLine{}
	for j, part := range parts {
		dice, err := engine.ParseDiceSum(part.Dice)
		if err != nil {
			continue
		}
		count, bonus := strconv.Itoa(dice.Count), strconv.Itoa(dice.Bonus)
		for i, a := range spell.Augments {
			for _, add := range a.Adds {
				more, err := engine.ParseDiceSum(add.Dice)
				if err != nil || add.Kind == "" || add.Kind != part.Kind {
					continue
				}
				count += fmt.Sprintf(" + %d * %s", more.Count, augmentSignal(i))
				bonus += fmt.Sprintf(" + %d * %s", more.Bonus, augmentSignal(i))
			}
		}
		lines = append(lines, rollLine(spell, part, dice.Faces, count, bonus, show, bornHidden, j+1))
	}
	for i, a := range spell.Augments {
		for _, add := range a.Adds {
			more, err := engine.ParseDiceSum(add.Dice)
			if err != nil || add.Kind != "" {
				continue
			}
			on := augmentSignal(i) + " > 0"
			if show != "" {
				on = show + " && " + on
			}
			count := fmt.Sprintf("%d * %s", more.Count, augmentSignal(i))
			bonus := fmt.Sprintf("%d * %s", more.Bonus, augmentSignal(i))
			lines = append(lines, rollLine(spell, add, more.Faces, count, bonus, on, true, len(parts)+1))
		}
	}
	return lines
}

func rollLine(spell book.Spell, part book.SpellRoll, faces int, count, bonus, show string, bornHidden bool, position int) spellRollLine {
	kind := ""
	if part.Kind != "" {
		kind = " de " + part.Kind
	}
	written := part.Dice + kind
	return spellRollLine{
		Static: written,
		Text: fmt.Sprintf("(%s) + 'd%d' + ((%s) > 0 ? '+' + (%s) : '') + '%s'",
			count, faces, bonus, bonus, strings.ReplaceAll(kind, "'", "\\'")),
		Show: show, BornHidden: bornHidden, Note: part.Note,
		Signal: "spell_hand_" + strconv.Itoa(position),
		Label:  fmt.Sprintf("O total da parcela %d que você rolou na mesa para %s", position, spell.Name),
	}
}

// noneOn é "nenhum destes aprimoramentos está ligado", ou vazio sem nenhum.
func noneOn(indexes []int) string {
	if len(indexes) == 0 {
		return ""
	}
	on := make([]string, 0, len(indexes))
	for _, i := range indexes {
		on = append(on, augmentSignal(i)+" > 0")
	}
	return "!(" + strings.Join(on, " || ") + ")"
}
