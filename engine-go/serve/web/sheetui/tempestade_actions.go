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
//	actionWaits(group, true, "") // "$turn_target === '' || $attack_open || !$my_turn"
func actionWaits(group actionGroup, needsTarget bool, also string) string {
	var waits []string
	if needsTarget {
		// O que pede alvo é golpe, e o próximo só abre depois da resposta do mestre.
		waits = append(waits, "$turn_target === ''", "$attack_open")
	}
	if also != "" {
		waits = append(waits, also)
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
	if waits := actionWaits(group, roll.NeedsTarget, roll.AlsoWaits); waits != "" {
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
	after := ""
	if len(row.Boosts) > 0 {
		after = rollClearsTheBoost()
	}
	if len(row.Doses) > 0 || row.ClearsDose {
		after += rollClearsTheDose()
	}
	return after
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
	} else if waits := actionWaits(group, false, ""); waits != "" {
		attrs["data-attr:disabled"] = waits
	}
	return attrs
}

// rollSheetFoot é o pé da folha do Rolar.
func rollSheetFoot(v View, group actionGroup, sheet rollSheet) templ.Component {
	return rollSheetFootBody(v, group, sheet)
}

// dialogOpens é o que o gesto zera ao abrir o diálogo dele: o `Before` da linha quando ela diz — a folha do
// Golpe Relâmpago desliga o Golpe Imprudente —, e o contador de degraus quando não.
func dialogOpens(roll actionRoll) string {
	if roll.Before != "" {
		return roll.Before
	}
	return useResets
}

// rowPicksATarget diz que a linha tem um gesto que pede alvo: é ela que o painel das ações no tabuleiro mostra
// (ALE-660), e as outras ficam na aba Ações.
func rowPicksATarget(row actionRow) bool {
	for _, roll := range row.Rolls {
		if roll.NeedsTarget {
			return true
		}
	}
	return false
}
