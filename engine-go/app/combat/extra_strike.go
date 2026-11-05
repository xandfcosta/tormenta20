package combat

import (
	"fmt"
	"slices"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// extraStrikeOf devolve o PM do ataque extra de uma arma natural (p25): o primeiro ataque da
// fonte paga, e o segundo das patas já está pago (p137). Se ele abriu, qual o limite e se a
// ficha paga é conferido sob a trava, no `session.Store.ProposeAttack`.
func extraStrikeOf(state *live.SessionRuntimeState, card engine.WeaponCard) (int, error) {
	if card.Natural == nil {
		return 0, fmt.Errorf("%s não é arma natural, e o ataque extra é da arma natural: %w", card.Name, app.ErrRefused)
	}
	if state.Scene != nil && slices.Contains(state.Scene.ExtraStrikesUsed, card.Natural.SourceID) {
		return 0, nil
	}
	return card.Natural.ExtraPM, nil
}
