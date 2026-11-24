package table

import (
	"fmt"
	"strconv"
	"strings"

	"t20engine/domain/live"
)

// A FAIXA DA MAGIA é irmã da do disparo (ver GLOSSARY, **dado da magia**): ela INFORMA o dado, o tipo, a
// resistência e a CD. Quem passou e o que se aplica é do mestre, com os gestos de PV que a fila já tem — ✓ e ✕
// aqui seriam dois controles que não fazem nada. As duas sessões a veem inteira: a magia cai à vista de todos.

// spellCastBand é a magia conjurada do jeito que a mesa lê.
type spellCastBand struct {
	// Head é "Arwen · Bola de Fogo".
	Head  string
	Parts []spellCastLine
	// ByText é a frase do dado que um aprimoramento mudou e que não sai na conjuração, vazia no resto.
	ByText string
	// Save é a resistência com a CD e o alcance: "Reflexos reduz à metade · CD 18 · alcance médio".
	Save string
}

// spellCastLine é uma parcela: o total, a conta ("6d6 (3, 5, 1, 6, 2, 4) de fogo") e a condição do texto.
type spellCastLine struct {
	Total int
	Tally string
	Note  string
}

// spellCastBandOf traduz a magia conjurada do regime na faixa, ou nil quando não há.
func spellCastBandOf(st *live.SessionRuntimeState) *spellCastBand {
	if st == nil || st.LastSpellCast == nil {
		return nil
	}
	c := st.LastSpellCast
	band := &spellCastBand{Head: c.Who + " · " + c.Spell}
	for _, part := range c.Parts {
		band.Parts = append(band.Parts, spellCastLine{Total: part.Total, Tally: spellTally(part), Note: part.Note})
	}
	if c.ByText {
		band.ByText = "um aprimoramento mudou o dado para um que não sai na conjuração: role pelo texto dele"
	}
	save := []string{}
	if c.Save != "" {
		save = append(save, c.Save, fmt.Sprintf("CD %d", c.DC))
	}
	if c.Range != "" {
		save = append(save, "alcance "+c.Range)
	}
	band.Save = strings.Join(save, " · ")
	return band
}

// spellTally é a conta da parcela: os dados que o app rolou, ou "da mesa" quando o total veio da mão, e o tipo.
func spellTally(part live.SpellCastPart) string {
	tally := part.Dice
	if len(part.Rolls) > 0 {
		written := make([]string, 0, len(part.Rolls))
		for _, die := range part.Rolls {
			written = append(written, strconv.Itoa(die))
		}
		tally += " (" + strings.Join(written, ", ") + ")"
	}
	if part.Kind != "" {
		tally += " de " + part.Kind
	}
	if part.ByHand {
		tally += " · da mesa"
	}
	return tally
}
