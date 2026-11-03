package combat

import (
	"context"
	"fmt"
	"slices"

	"t20engine/app"
	"t20engine/domain/live"
)

func (s Strike) doseFor(ctx context.Context, req Request, setup strikeSetup) (*live.AttackDose, error) {
	state, err := s.tables.State(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	return setup.doseOf(req, doseComingBack(state))
}

// doseComingBack é a dose gasta do acerto que este golpe vai tirar da mesa: guardar o
// golpe novo a devolve à arma, e ela já conta para ele.
func doseComingBack(state *live.SessionRuntimeState) *live.AttackDose {
	if state == nil || state.PendingAttack == nil || state.PendingAttack.Dose == nil || !state.PendingAttack.Dose.Spent {
		return nil
	}
	return state.PendingAttack.Dose
}

// doseOf é a dose em jogo neste golpe (p165). Quem a gasta é a decisão do mestre,
// e não este gesto: errar não libera nada.
func (setup strikeSetup) doseOf(req Request, comingBack *live.AttackDose) (*live.AttackDose, error) {
	loaded := setup.weapon.Doses
	if comingBack != nil && comingBack.ItemID == setup.weapon.ItemID {
		loaded = append(slices.Clone(loaded), comingBack.CatalogID)
	}
	if len(loaded) == 0 && req.Dose == "" {
		return nil, nil
	}
	chosen := req.Dose
	if chosen == "" {
		chosen = loaded[0]
	}
	if !slices.Contains(loaded, chosen) {
		return nil, fmt.Errorf("%s não tem essa dose carregada, e o pedido veio com %q: %w",
			setup.weapon.Name, chosen, app.ErrRefused)
	}
	name, effect := setup.striker.WrittenDose(chosen)
	return &live.AttackDose{ItemID: setup.weapon.ItemID, CatalogID: chosen, Name: name, Effect: effect}, nil
}
