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
// ficha paga é conferido sob a trava, no `session.Store.ProposeAttack`. O adicional de um estilo
// (p125) não cobra PM, e o de um poder cobra o dele (ALE-651).
func extraStrikeOf(state *live.SessionRuntimeState, card engine.WeaponCard, power string) (int, error) {
	if power != "" {
		paid, found := paidExtraOf(card, power)
		if !found {
			return 0, fmt.Errorf("%s não faz o ataque pago %q: %w", card.Name, power, app.ErrRefused)
		}
		return paid.PM, nil
	}
	if card.Style != nil {
		return 0, nil
	}
	if card.Natural == nil {
		return 0, fmt.Errorf("%s não é arma natural, e o ataque extra é da arma natural: %w", card.Name, app.ErrRefused)
	}
	if state.Scene != nil && slices.Contains(state.Scene.ExtraStrikesUsed, card.Natural.SourceID) {
		return 0, nil
	}
	return card.Natural.ExtraPM, nil
}

// opensTheStyle grava no golpe de agredir o estilo que a resposta do mestre vai abrir (p125).
func opensTheStyle(pending *live.PendingAttack, card engine.WeaponCard) {
	if card.Style != nil {
		pending.OpensStyle, pending.StyleItem, pending.FullAction = card.Style.SourceID, card.ItemID, card.Style.FullAction
	}
}

// paidExtraOf é o ataque pago desse poder entre os que a carta alcança.
func paidExtraOf(card engine.WeaponCard, power string) (engine.PaidExtraStrike, bool) {
	for _, paid := range card.PaidExtras {
		if paid.PowerID == power {
			return paid, true
		}
	}
	return engine.PaidExtraStrike{}, false
}

// impulsiveWeapon é a carta do Golpe Imprudente (p77): ele só existe no Golpe Relâmpago, de quem tem o poder,
// e dá um dado a mais do mesmo tipo. A guarda aberta pousa no ✓ do mestre, como a da investida.
func impulsiveWeapon(striker Combatant, req Request, weapon engine.WeaponCard) (engine.WeaponCard, error) {
	if !req.Impulsive {
		return weapon, nil
	}
	if !striker.StrikesImpulsively {
		return weapon, fmt.Errorf("%s não tem o Golpe Imprudente (p77): %w", striker.Label, app.ErrRefused)
	}
	if paid, found := paidExtraOf(weapon, req.ExtraPower); !found || !paid.Unarmed {
		return weapon, fmt.Errorf("o Golpe Imprudente é atacar de forma impulsiva no Golpe Relâmpago (p77): %w", app.ErrRefused)
	}
	return engine.ImpulsiveStrikeCard(weapon)
}
