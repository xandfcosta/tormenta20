package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// settleFollowUp confere a manobra em seguida do pedido contra o que a ficha dá (ALE-651, ALE-659):
//
//   - ARMA no gesto rolado as manobras em seguida que dependem dele — o agarrar arma a derrubada do Lutador de
//     Chão, o golpe com o Ataque Furtivo ligado arma o agarrar da Mão na Boca. É agora que o interruptor que ela
//     pede é conferido, porque é agora que o golpe é rolado; a decisão do mestre é que as abre depois;
//   - e a que este gesto FAZ (`Request.TakesFollowUp`) soma o PM dela ao do golpe. Se ela abriu, e contra quem, é
//     conferido sob a trava, no `session.Store.ProposeAttack`.
func (setup *strikeSetup) settleFollowUp(req Request) error {
	if req.TargetObject == nil {
		for _, offer := range setup.striker.FollowUps {
			if offer.After == req.Maneuver && offer.Ready {
				setup.followUps = append(setup.followUps, *followUpOfOffer(offer))
			}
		}
	}
	if req.TakesFollowUp == "" {
		return nil
	}
	for _, offer := range setup.striker.FollowUps {
		if offer.PowerID == req.TakesFollowUp && offer.Maneuver == req.Maneuver {
			setup.followUpOf = followUpOfOffer(offer)
			setup.boost.pm += offer.PM
			return nil
		}
	}
	return fmt.Errorf("%s não faz a manobra em seguida %q com %q: %w", setup.striker.Label, req.TakesFollowUp, req.Maneuver, app.ErrRefused)
}

func followUpOfOffer(offer engine.FollowUpManeuver) *live.FollowUp {
	return &live.FollowUp{
		PowerID: offer.PowerID, Name: offer.Name, Maneuver: offer.Maneuver, On: offer.On, When: offer.Condition(), PM: offer.PM,
	}
}
