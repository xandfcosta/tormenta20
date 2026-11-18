package table

import (
	"strconv"
	"strings"

	"t20engine/domain/board"
)

// moveModeLandSlug é como a rota diz "a pé": o modo terrestre é vazio no domínio.
const moveModeLandSlug = "terrestre"

// moveModeChoice é uma opção do "Anda por" no cartão do movimento.
type moveModeChoice struct {
	Label  string
	Slug   string
	Active bool
}

var moveModeNames = map[board.MoveModeKind]string{
	board.MoveModeClimb: "Escalada", board.MoveModeSwim: "Natação", board.MoveModeFly: "Voo",
}

// moveModeChoices monta o "Anda por" da peça: a pé e cada outro deslocamento dela, em metros como a
// ficha os diz. Vazio para a peça que só anda — o cartão de hoje fica como está (mockup aprovado).
func moveModeChoices(token board.BoardToken, chosen board.MoveModeKind) []moveModeChoice {
	if len(token.OtherModes) == 0 {
		return nil
	}
	choices := []moveModeChoice{{
		Label: "Terrestre " + squaresInMetres(board.ModeBudget(token, board.MoveModeLand)),
		Slug:  moveModeLandSlug, Active: chosen == board.MoveModeLand,
	}}
	for _, other := range token.OtherModes {
		choices = append(choices, moveModeChoice{
			Label: moveModeNames[other.Kind] + " " + squaresInMetres(other.Squares),
			Slug:  string(other.Kind), Active: chosen == other.Kind,
		})
	}
	return choices
}

// squaresInMetres escreve quadrados em metros como a mesa fala: 6 vira "9m", e 3 vira "4,5m".
func squaresInMetres(squares int) string {
	written := strconv.FormatFloat(float64(squares)*1.5, 'f', -1, 64)
	return strings.Replace(written, ".", ",", 1) + "m"
}

// moveSheet é o diálogo do corpo do cartão do movimento no celular, e o corpo dele no desktop.
const moveSheet = "move-sheet"

// moveSummary é a conta da cabeça do cartão: "4 de 6", ou só "4" sem orçamento.
func moveSummary(m *moveView) string {
	if m.Budget < 0 {
		return strconv.Itoa(m.Cost)
	}
	return strconv.Itoa(m.Cost) + " de " + strconv.Itoa(m.Budget)
}
