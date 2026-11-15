package combat

import (
	"fmt"
	"strings"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// FollowUpAsk é a manobra em seguida no pedido (ALE-651, `engine.FollowUpManeuver`). `Declares` a declara
// neste gesto, antes de rolar — o agarrar do Lutador de Chão (p77), o golpe da Mão na Boca (p74) —, e `Takes`
// é o poder da manobra em seguida que este gesto faz.
type FollowUpAsk struct {
	Declares bool
	Takes    string
}

// settleFollowUp confere a manobra em seguida do pedido contra o que a ficha dá: a declarada viaja na proposta,
// para o ✓ do mestre a abrir, e a que este gesto faz soma o PM dela ao do golpe. Se ela abriu, e contra quem, é
// conferido sob a trava, no `session.Store.ProposeAttack`.
func (setup *strikeSetup) settleFollowUp(req Request) error {
	if req.FollowUp.Declares {
		offer, err := declaredFollowUp(setup.striker, req)
		if err != nil {
			return err
		}
		setup.followUp = followUpOfOffer(offer)
	}
	if req.FollowUp.Takes == "" {
		return nil
	}
	for _, offer := range setup.striker.FollowUps {
		if offer.PowerID == req.FollowUp.Takes && offer.Maneuver == req.Maneuver {
			setup.followUpOf = followUpOfOffer(offer)
			setup.boost.pm += offer.PM
			return nil
		}
	}
	return fmt.Errorf("%s não faz a manobra em seguida %q com %q: %w", setup.striker.Label, req.FollowUp.Takes, req.Maneuver, app.ErrRefused)
}

// declaredFollowUp é a manobra em seguida que este gesto pode declarar: a do poder que a abre depois desta
// manobra, ou depois do golpe quando ele não é manobra.
func declaredFollowUp(striker Combatant, req Request) (engine.FollowUpManeuver, error) {
	if req.TargetObject != nil {
		return engine.FollowUpManeuver{}, fmt.Errorf("a manobra em seguida é contra criatura, e não contra peça de cenário: %w", app.ErrRefused)
	}
	for _, offer := range striker.FollowUps {
		if offer.After != req.Maneuver {
			continue
		}
		if !offer.Ready {
			return offer, fmt.Errorf("%s: a manobra em seguida vale no golpe com o %s ligado, e ele não está: %w",
				offer.Name, strings.ReplaceAll(offer.NeedsFlag, "-", " "), app.ErrRefused)
		}
		return offer, nil
	}
	after := req.Maneuver
	if after == "" {
		after = "agredir"
	}
	return engine.FollowUpManeuver{}, fmt.Errorf("%s não tem poder que abra uma manobra em seguida depois de %s: %w", striker.Label, after, app.ErrRefused)
}

func followUpOfOffer(offer engine.FollowUpManeuver) *live.FollowUp {
	return &live.FollowUp{PowerID: offer.PowerID, Name: offer.Name, Maneuver: offer.Maneuver, PM: offer.PM}
}
