package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
)

// paidBoost é o impulso que o golpe leva: o poder e o PM que a proposta paga.
type paidBoost struct {
	powerID string
	pm      int
}

// weaponFor é a carta com que o golpe rola: a empunhada no índice pedido, já
// com o impulso de PM quando houver um.
//
// O PM é conferido AQUI, contra a ficha, antes de rolar: a proposta que a ficha não
// pode pagar não chega a existir. Quem cobra é o `session.Store.ProposeAttack`, e o
// cancelamento do mestre devolve (decisão do dono).
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
			"%s tem %d ataque(s) à mão, contando o desarmado, e o pedido veio no %d: %w",
			striker.Label, len(cards), req.Weapon, app.ErrRefused)
	}
	card := cards[req.Weapon]
	if req.OtherUse {
		if card.OtherUse == nil {
			return engine.WeaponCard{}, boost, fmt.Errorf("%s não tem outro uso: %w", card.Name, app.ErrRefused)
		}
		card = *card.OtherUse
	}
	if card.RefusesBoost {
		return engine.WeaponCard{}, boost, fmt.Errorf(
			"%s não recebe este impulso, que só vale corpo a corpo: %w", card.Name, app.ErrRefused)
	}
	return card, boost, nil
}
