package board

import (
	"fmt"
	"strings"

	"t20engine/domain/live"
)

// MoveModeKind é o deslocamento com que uma peça anda (p228-229). Vazio é a pé, o terrestre.
type MoveModeKind string

const (
	MoveModeLand  MoveModeKind = ""
	MoveModeClimb MoveModeKind = "escalada"
	MoveModeSwim  MoveModeKind = "natacao"
	MoveModeFly   MoveModeKind = "voo"
)

// MoveMode é um deslocamento além do terrestre, em QUADRADOS, como o `SpeedSquares`.
type MoveMode struct {
	Kind    MoveModeKind `json:"kind"`
	Squares int          `json:"squares"`
}

// ModeBudget é o orçamento da peça no modo: o terrestre, ou o do modo que ela tem.
func ModeBudget(token BoardToken, mode MoveModeKind) int {
	for _, other := range token.OtherModes {
		if other.Kind == mode && mode != MoveModeLand {
			return other.Squares
		}
	}
	return speedOf(token)
}

// PendingMode é o modo do movimento proposto para a peça, ou a pé quando não há proposta dela: a
// prévia do arrasto e o alcance medem com ele, para concordarem com o que o servidor mede.
func PendingMode(b *BoardState, tokenID string) MoveModeKind {
	if b != nil && b.Pending != nil && b.Pending.TokenID == tokenID {
		return b.Pending.Mode
	}
	return MoveModeLand
}

// ChangeMoveMode troca o modo do movimento proposto e o mede de novo, pelas mesmas paradas.
//
// O SISTEMA NÃO SABE o que é água ou parede: o modo só troca o orçamento e, no voo, o terreno
// difícil. Quem diz se nadar vale ali é o mestre, que confirma — e por isso quem troca é quem
// pode propor, e não há recusa por "não há água".
func ChangeMoveMode(b *BoardState, st *live.SessionRuntimeState, tokenID string, mode MoveModeKind, by Mover) error {
	pending := b.Pending
	if pending == nil || pending.TokenID != tokenID {
		return fmt.Errorf("não há movimento proposto para a peça %q", tokenID)
	}
	if by.Role != "gm" && pending.ByUserID != by.UserID {
		return fmt.Errorf("o movimento de %q foi proposto por outra pessoa", tokenID)
	}
	token := FindToken(b, tokenID)
	if token == nil {
		return fmt.Errorf("peça %q não está no tabuleiro", tokenID)
	}
	if mode != MoveModeLand && !hasMode(*token, mode) {
		return fmt.Errorf("a peça %q não tem o deslocamento %q; tem: %s", token.Label, mode, modeNames(*token))
	}
	stops := pending.Stops
	if len(stops) == 0 {
		stops = pending.Path
	}
	pending.Mode = mode
	return ProposeMoveWithStops(b, st, tokenID, stops, by)
}

func hasMode(token BoardToken, mode MoveModeKind) bool {
	for _, other := range token.OtherModes {
		if other.Kind == mode {
			return true
		}
	}
	return false
}

// modeNames diz os modos da peça, para a recusa nomear o que ela pode escolher.
func modeNames(token BoardToken) string {
	names := []string{"terrestre"}
	for _, other := range token.OtherModes {
		names = append(names, string(other.Kind))
	}
	return strings.Join(names, ", ")
}
