package table

import (
	"fmt"
	"strings"

	"t20engine/domain/live"
)

// A FAIXA DA MAGIA (ver GLOSSARY, **dado da magia**): o dado, o tipo, a resistência e a CD, uma linha por
// parcela. Quem passou e o que se aplica é do mestre, com os gestos de PV que a fila já tem. As duas sessões a
// veem inteira: a magia cai à vista de todos.

// spellCastBandOf traduz a magia conjurada do regime na faixa, ou nil quando não há.
func spellCastBandOf(st *live.SessionRuntimeState) *rollBand {
	if st == nil || st.LastSpellCast == nil {
		return nil
	}
	c := st.LastSpellCast
	band := &rollBand{What: c.Who + " · " + c.Spell}
	// O DADO DA VEZ diz o momento na cabeça (ver GLOSSARY, **dado da vez**): a mesma faixa, e quem lê sabe que não
	// é uma conjuração nova.
	if c.When != "" {
		band.What += " · " + c.When
	}
	for _, part := range c.Parts {
		band.Lines = append(band.Lines, rollLine{
			Total: part.Total, Tally: writtenRoll(part.Dice, part.Rolls, part.Kind, part.ByHand), Note: part.Note,
		})
	}
	if c.ByText {
		band.Help = append(band.Help, "um aprimoramento mudou o dado para um que não sai na conjuração: role pelo texto dele")
	}
	// A resistência com a CD e o alcance: "Reflexos reduz à metade · CD 18 · alcance médio".
	save := []string{}
	if c.Save != "" {
		save = append(save, c.Save, fmt.Sprintf("CD %d", c.DC))
	}
	if c.Range != "" {
		save = append(save, "alcance "+c.Range)
	}
	if len(save) > 0 {
		band.Help = append(band.Help, strings.Join(save, " · "))
	}
	return band
}
