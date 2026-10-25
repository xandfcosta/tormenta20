package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
)

// paidBoost é o impulso que o golpe leva: o poder e o PM que a confirmação
// cobra.
type paidBoost struct {
	powerID string
	pm      int
}

// weaponFor é a carta com que o golpe rola: a empunhada no índice pedido, já
// com o impulso de PM quando houver um.
//
// O PM é conferido AQUI, contra a ficha, e cobrado só na confirmação (decisão
// do dono na ALE-487): um provisório que o mestre cancela não custa nada, e um
// que a ficha não pode pagar não chega a existir.
func weaponFor(striker Combatant, req Request) (engine.WeaponCard, paidBoost, error) {
	cards := striker.Weapons
	boost := paidBoost{}
	if req.Boost.PowerID != "" {
		if striker.Boost == nil {
			return engine.WeaponCard{}, boost, fmt.Errorf("%s não paga PM no golpe: %w", striker.Label, app.ErrRefused)
		}
		boosted, pm, err := striker.Boost(req.Boost)
		if err != nil {
			return engine.WeaponCard{}, boost, fmt.Errorf("%w: %w", err, app.ErrRefused)
		}
		if pm > striker.ManaAvailable {
			return engine.WeaponCard{}, boost, fmt.Errorf(
				"o impulso custa %d PM e %s tem %d: %w", pm, striker.Label, striker.ManaAvailable, app.ErrRefused)
		}
		cards, boost = boosted, paidBoost{powerID: req.Boost.PowerID, pm: pm}
	}
	if req.Weapon < 0 || req.Weapon >= len(cards) {
		return engine.WeaponCard{}, boost, fmt.Errorf(
			"%s empunha %d arma(s) e o pedido veio na %d: %w",
			striker.Label, len(cards), req.Weapon, app.ErrRefused)
	}
	return cards[req.Weapon], boost, nil
}
