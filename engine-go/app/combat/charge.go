package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
)

// A INVESTIDA NO GOLPE (p235, p42, ALE-499). A regra do número mora no motor
// (`engine/charge.go`); aqui ficam as recusas do pedido e o que ele muda na
// carta. A penalidade na Defesa NÃO sai daqui: ela pousa na confirmação do
// mestre, como o PM do impulso — o golpe cancelado não abriu a guarda de
// ninguém.

// As investidas, com os nomes do motor.
const (
	ChargePlain    = engine.ChargePlain
	ChargeReckless = engine.ChargeReckless
)

// chargedWeapon confere a investida pedida e devolve a carta com o que ela
// muda no dano — o +1d12 da Imprudente. O +2 no ataque é situação e entra pelo
// `specialSituations`.
func chargedWeapon(striker Combatant, req Request, weapon engine.WeaponCard) (engine.WeaponCard, error) {
	if req.Charge == "" {
		return weapon, nil
	}
	if _, known := engine.ChargeSituation(req.Charge); !known {
		return weapon, fmt.Errorf("a investida %q não existe; o pedido vem como %q ou %q: %w",
			req.Charge, ChargePlain, ChargeReckless, app.ErrRefused)
	}
	if req.Maneuver != "" {
		return weapon, fmt.Errorf("a investida termina num ataque, e não numa manobra (p235): %w", app.ErrRefused)
	}
	if !weapon.IsMelee() {
		return weapon, fmt.Errorf("a investida termina num ataque corpo a corpo, e %s é de %s (p235): %w",
			weapon.Name, weapon.Skill, app.ErrRefused)
	}
	if req.Charge != ChargeReckless {
		return weapon, nil
	}
	if !striker.ChargesRecklessly {
		return weapon, fmt.Errorf("%s não tem a Investida Imprudente (p42): %w", striker.Label, app.ErrRefused)
	}
	// UMA CÓPIA da lista: a carta veio do combatente, e acrescentar no
	// `ExtraDamage` dela mexeria na carta dos próximos golpes.
	extras := append([]engine.ExtraDamage{}, weapon.ExtraDamage...)
	weapon.ExtraDamage = append(extras, engine.ExtraDamage{Dice: engine.RecklessChargeDice, Type: weapon.DamageType})
	return weapon, nil
}
