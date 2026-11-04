package combat

import (
	"fmt"
	"slices"

	"t20engine/app"
	"t20engine/domain/live"
)

// doseOf é a dose em jogo neste golpe (p165). Quem a gasta é a decisão do mestre,
// e não este gesto: errar não libera nada.
func (setup strikeSetup) doseOf(req Request) (*live.AttackDose, error) {
	loaded := setup.weapon.Doses
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
