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
	// Riders é o que a Marca dá com os acréscimos da ficha, por contagem de
	// degraus — "dano +1d8 · perícias +2 (Espreitar)" —, e Doubled o mesmo com
	// a caixa do Inimigo. Vazios sem acréscimo (p51, ALE-509).
	Riders  [][]riderPart
	Doubled [][]riderPart
	// DoubledBuys é o `Buys` com a caixa marcada, e CanDouble diz se ela
	// aparece: só para quem tem o Inimigo.
	DoubledBuys []string
	CanDouble   bool
}

// riderPart é um pedaço da linha dos acréscimos: "perícias", "+2", "Espreitar".
type riderPart struct {
	What, Value, Source string
}

// opensDialog diz se o "Usar" abre o diálogo: há degrau a escolher, ou a
// caixa do Inimigo a marcar.
func (d useDegrees) opensDialog() bool { return d.MaxSteps > 0 || d.CanDouble }

// useDegreesFor resolve os degraus de um instantâneo com escala, ou nil.
func useDegreesFor(dto sheet.CharacterDTO, spec book.Activation) *useDegrees {
	if spec.Kind != "instant" || spec.Scaling == nil {
		return nil
	}
	d := &useDegrees{
		BasePm: spec.Scaling.BasePm, StepPm: spec.Scaling.StepPm, StepLabel: spec.Scaling.StepLabel,
		MaxSteps: character.PowerMaxSteps(dto, spec),
	}
	riders := character.RidersOf(dto, spec)
	d.CanDouble = book.RidersDouble(spec, riders)
	for steps := 0; steps <= d.MaxSteps; steps++ {
		d.Buys = append(d.Buys, grantWritten(book.ScaledGrantModifiers(spec, steps)))
		doubled := book.GrantModifiersWithRiders(spec, steps, riders, true)
		d.DoubledBuys = append(d.DoubledBuys, grantWritten(doubled))
		if hasRiderModifiers(spec, riders) {
			d.Riders = append(d.Riders, riderParts(spec, book.GrantModifiersWithRiders(spec, steps, riders, false)))
			d.Doubled = append(d.Doubled, riderParts(spec, doubled))
		}
	}
	return d
}

// hasRiderModifiers diz se algum acréscimo soma modificador — o Inimigo
// sozinho só dobra, e a linha repetiria o contador.
func hasRiderModifiers(spec book.Activation, riders []book.Activation) bool {
	for _, r := range riders {
		if r.Rider.On == spec.ID && len(r.Rider.Modifiers) > 0 {
			return true
		}
	}
	return false
}

// riderWhat é o nome que a linha dá a cada alvo que um acréscimo move.
var riderWhat = map[string]string{
	"damage": "dano", "expertiseAll": "perícias", "critRange": "margem de ameaça",
}

// riderParts escreve a concessão com os acréscimos, um pedaço por modificador.
// O do próprio poder não leva fonte: ele é o que o diálogo já nomeia.
func riderParts(spec book.Activation, mods []engine.Modifier) []riderPart {
	parts := make([]riderPart, 0, len(mods))
	for _, m := range mods {
		part := riderPart{What: riderWhat[m.Target.K], Value: book.WithSign(m.Amount)}
		if part.What == "" {
			part.What = m.Note
		}
		if m.Dice != "" {
			part.Value = "+" + m.Dice
		}
		if m.Note != spec.Name {
			part.Source = m.Note
		}
		parts = append(parts, part)
	}
	return parts
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

// useResets zera o que o diálogo de usar escolhe: um degrau ou uma caixa que
// sobrou de outro diálogo viraria pedido aqui.
const useResets = "$stance_degrees = 0; $prey_favored = false; "

// useOpensCommand abre o contador zerado.
func useOpensCommand(powerID string) string {
	return fmt.Sprintf("%s$detail = '%s'", useResets, useDialogID(powerID))
}

// useCounterExpr é o que o contador mostra: o que o degrau compra e o PM. O
// poder sem concessão — o custo variável do Gritar Ordens (ALE-506) — não tem
// o que escrever antes do PM, e a frase do degrau já diz o que ele compra.
func useCounterExpr(d useDegrees) string {
	buys, _ := json.Marshal(d.Buys)
	if d.CanDouble {
		doubled, _ := json.Marshal(d.DoubledBuys)
		buys = []byte(fmt.Sprintf("($prey_favored ? %s : %s)", doubled, buys))
	}
	return fmt.Sprintf("(%s[$stance_degrees] ? %s[$stance_degrees] + ' · ' : '') + (%d + %d * $stance_degrees) + ' PM'",
		buys, buys, d.BasePm, d.StepPm)
}

// riderShows é quando a linha dos acréscimos de `steps` degraus aparece.
func riderShows(steps int, doubled bool) string {
	if doubled {
		return fmt.Sprintf("$stance_degrees == %d && $prey_favored", steps)
	}
	return fmt.Sprintf("$stance_degrees == %d && !$prey_favored", steps)
}
