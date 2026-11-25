package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
)

// SpellHand é a mão de uma magia que faz a manobra no lugar da arma: o nome dela e o que ela soma ao d20, já com a
// perícia de quem conjurou (a Mão Poderosa de Talude: o Misticismo mais 10, p198; ver GLOSSARY, **ação da magia**).
type SpellHand struct {
	Name  string
	Bonus int
}

// handFor é a carta da mão para a manobra. A mão só faz manobra: não golpeia, não paga impulso, não é o ataque
// extra nem a manobra em seguida de um poder de quem conjurou, e não investe.
func handFor(req Request) (engine.WeaponCard, paidBoost, error) {
	switch {
	case req.Maneuver == "":
		return engine.WeaponCard{}, paidBoost{}, fmt.Errorf("%s faz manobra, e não golpe: %w", req.Hand.Name, app.ErrRefused)
	case req.Boost.PowerID != "", req.ExtraStrike, req.TakesFollowUp != "", req.Charge != "", req.Impulsive:
		return engine.WeaponCard{}, paidBoost{}, fmt.Errorf("a manobra de %s é só da mão: sem impulso, ataque extra, manobra em seguida, investida ou Golpe Imprudente: %w",
			req.Hand.Name, app.ErrRefused)
	}
	return engine.WeaponCard{Name: req.Hand.Name, Attack: req.Hand.Bonus}, paidBoost{}, nil
}
