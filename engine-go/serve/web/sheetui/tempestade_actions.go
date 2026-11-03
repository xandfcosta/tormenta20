package sheetui

import (
	"strings"

	"github.com/a-h/templ"

	"t20engine/serve/web/ds"
)

// actionWaits é a expressão que APAGA um gesto das Ações: o alvo que
// falta, e a vez que não é do jogador quando o grupo gasta a vez. Vazia quando
// nada espera.
//
//	actionWaits(group, roll) // "$turn_target === '' || !$my_turn"
func actionWaits(group actionGroup, needsTarget bool) string {
	var waits []string
	if needsTarget {
		waits = append(waits, "$turn_target === ''")
	}
	if group.OnTurn {
		waits = append(waits, "!$my_turn")
	}
	return strings.Join(waits, " || ")
}

// rollAttrs junta o gesto, a espera e o motivo do botão de rolar.
func rollAttrs(v View, group actionGroup, roll actionRoll, gesture templ.Attributes) templ.Attributes {
	attrs := templ.Attributes{"type": "button"}
	for key, value := range gesture {
		attrs[key] = value
	}
	if waits := actionWaits(group, roll.NeedsTarget); waits != "" {
		attrs["data-attr:disabled"] = waits
	}
	if title := rollTitle(roll); title != "" {
		attrs["title"] = title
	}
	if roll.LabelExpr != "" {
		attrs["data-show"] = roll.LabelExpr + " !== ''"
	}
	return attrs
}

// rollLook é o peso de um gesto numa linha com vários: a manobra tem cinco iguais,
// e o ataque tem o principal e o outro uso.
func rollLook(roll actionRoll) ds.ButtonVariant {
	if roll.Primary {
		return ds.ButtonPrimary
	}
	return ds.ButtonSecondary
}

// rollAfter é o que roda depois do gesto: o golpe gasta o impulso escolhido.
func rollAfter(row actionRow) string {
	if len(row.Boosts) > 0 {
		return rollClearsTheBoost()
	}
	return ""
}

// chipAttrs junta o gesto e a espera do chip. O que o servidor recusa nasce
// desabilitado, com o motivo no `title`.
func chipAttrs(group actionGroup, chip actionChip, gesture templ.Attributes) templ.Attributes {
	attrs := templ.Attributes{}
	for key, value := range gesture {
		attrs[key] = value
	}
	if !chip.Can {
		attrs["disabled"] = true
		attrs["title"] = chip.Why
	} else if waits := actionWaits(group, false); waits != "" {
		attrs["data-attr:disabled"] = waits
	}
	return attrs
}
