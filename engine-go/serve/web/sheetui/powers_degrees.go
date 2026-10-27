package sheetui

import (
	"encoding/json"
	"fmt"
	"strings"

	"t20engine/app/character"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O PODER INSTANTÂNEO QUE VENDE DEGRAU — hoje a Marca da Presa (p50, ALE-487).
//
// O contador é o MESMO sinal da postura (`stance_degrees`), e por isso todo
// gesto de usar começa zerando-o: um valor que sobrou do diálogo da Fúria
// viraria degrau pedido aqui. O servidor só lê o sinal no poder que vende
// degrau (`usePower`).

// useDegrees é o que a tela precisa para oferecer os degraus de um uso.
type useDegrees struct {
	MaxSteps int
	BasePm   int
	StepPm   int
	// Buys é o que cada contagem de degraus compra, escrito ("+1d8"), de zero
	// ao teto — do mesmo `ScaledGrantModifiers` que o gesto grava.
	Buys []string
	// StepLabel é o que um degrau compra, na voz do catálogo.
	StepLabel string
}

// useDegreesFor resolve os degraus de um instantâneo com escala, ou nil.
func useDegreesFor(dto sheet.CharacterDTO, spec book.Activation) *useDegrees {
	if spec.Kind != "instant" || spec.Scaling == nil {
		return nil
	}
	d := &useDegrees{
		BasePm: spec.Scaling.BasePm, StepPm: spec.Scaling.StepPm, StepLabel: spec.Scaling.StepLabel,
		MaxSteps: character.PowerMaxSteps(dto, spec),
	}
	for steps := 0; steps <= d.MaxSteps; steps++ {
		d.Buys = append(d.Buys, grantWritten(book.ScaledGrantModifiers(spec, steps)))
	}
	return d
}

// grantWritten é o dado ou o número do primeiro modificador da concessão.
func grantWritten(mods []engine.Modifier) string {
	for _, m := range mods {
		if m.Dice != "" {
			return "+" + m.Dice
		}
		if m.Amount != 0 {
			return book.WithSign(m.Amount)
		}
	}
	return ""
}

// grantInEffect é o que a concessão do poder está dando AGORA, escrito, ou ""
// quando o efeito não está em curso.
func grantInEffect(dto sheet.CharacterDTO, powerID string) string {
	for _, e := range dto.ActiveEffects {
		if e.CatalogID != powerID {
			continue
		}
		var mods []engine.Modifier
		if err := json.Unmarshal([]byte(e.Modifiers), &mods); err != nil {
			return "em curso"
		}
		return grantWritten(mods)
	}
	return ""
}

// useDialogID é o valor de `$detail` que abre o contador do poder. Sem pontos:
// o id vira id de elemento.
func useDialogID(powerID string) string {
	return "usa-" + strings.ReplaceAll(powerID, ".", "-")
}

// useOpensCommand abre o contador zerado.
func useOpensCommand(powerID string) string {
	return fmt.Sprintf("$stance_degrees = 0; $detail = '%s'", useDialogID(powerID))
}

// useCounterExpr é o que o contador mostra: o que o degrau compra e o PM.
func useCounterExpr(d useDegrees) string {
	blob, _ := json.Marshal(d.Buys)
	return fmt.Sprintf("%s[$stance_degrees] + ' · ' + (%d + %d * $stance_degrees) + ' PM'",
		blob, d.BasePm, d.StepPm)
}

// useCommand é o toque no "Usar": abre o contador no poder que vende degrau, e
// usa direto no resto — sempre com o contador zerado antes.
func useCommand(v View, row powerRow) string {
	if row.Degrees != nil && row.Degrees.MaxSteps > 0 {
		return useOpensCommand(row.Command)
	}
	return "$stance_degrees = 0; " + sheetPost(v, "/poderes/usa/"+row.Command)
}
