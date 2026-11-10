package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// paidBoost é o impulso que o golpe leva: o poder e o PM que a proposta paga.
type paidBoost struct {
	powerID string
	pm      int
	// hitPoints é o PV que o gasto tira de quem ataca, já rolado (a Tumular).
	hitPoints int
	// against é o tipo contra o qual o gasto vale e o que a mesa sabe do alvo (a Anticriatura).
	against *live.ChosenKindRead
}

// chosenKindAgainst é o tipo de criatura escolhido no encanto do gasto pedido, com o que a mesa sabe do alvo,
// ou nulo no gasto que vale contra qualquer um. O gasto não é recusado contra outro tipo: quem confere é o
// mestre, pela faixa dele (decisão do dono), e recusar diria ao jogador o tipo do alvo.
func chosenKindAgainst(striker Combatant, req Request, target book.CreatureNature) *live.ChosenKindRead {
	if req.Boost.PowerID == "" || striker.BoostAgainst == nil {
		return nil
	}
	chosen := striker.BoostAgainst(req.Boost)
	if chosen == "" {
		return nil
	}
	return &live.ChosenKindRead{Chosen: chosen, TargetKind: target.Kind, TargetRace: target.Race}
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
			"%s tem %d ataque(s), contando o desarmado e as armas naturais, e o pedido veio no %d: %w",
			striker.Label, len(cards), req.Weapon, app.ErrRefused)
	}
	card := cards[req.Weapon]
	if req.OtherUse {
		if card.OtherUse == nil {
			return engine.WeaponCard{}, boost, fmt.Errorf("%s não tem outro uso: %w", card.Name, app.ErrRefused)
		}
		card = *card.OtherUse
	}
	// O GASTO DE UM ENCANTO é da arma dele (p335-336): o anúncio da carta não tem escopo de arma, e
	// sem esta conferência a outra mão pagaria e anunciaria o efeito do encanto que não tem.
	if itemID, _, isEnchant := engine.EnchantBoostParts(boost.powerID); isEnchant && card.ItemID != itemID {
		owner := "outra arma"
		for _, c := range cards {
			if c.ItemID == itemID {
				owner = c.Name
			}
		}
		return engine.WeaponCard{}, boost, fmt.Errorf(
			"o gasto de 2 PM é do encanto de %s, e o golpe veio com %s: %w", owner, card.Name, app.ErrRefused)
	}
	if card.RefusesBoost {
		return engine.WeaponCard{}, boost, fmt.Errorf(
			"%s não recebe este impulso, que só vale corpo a corpo: %w", card.Name, app.ErrRefused)
	}
	return card, boost, nil
}
