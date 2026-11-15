package combat

import (
	"fmt"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// rollManeuver resolve o teste OPOSTO da manobra e guarda o provisório.
//
// Ele divide com o golpe tudo que vem antes: a vez, a posse, a ação padrão e as
// duas pontas da fila. O que muda é só a REGRA no meio — e é por isso que ele é
// um ramo aqui, e não um caso de uso irmão que repetiria sete conferências.
func (s Strike) rollManeuver(who app.Caller, req Request, setup strikeSetup, d20 int) (live.PendingAttack, error) {
	if req.TargetObject != nil {
		return live.PendingAttack{}, fmt.Errorf("manobra é contra criatura, e não contra peça de cenário: %w", app.ErrRefused)
	}
	striker, victim, weapon := setup.striker, setup.victim, setup.weapon
	// O LUTA DO DEFENSOR, e não a Defesa dele: "mesmo que ela esteja usando uma
	// arma de ataque à distância, deve fazer o teste usando seu valor de Luta"
	// (p234). Quem não tem arma empunhada não tem carta, e aí o Luta dele é
	// zero — um NPC de bloco escrito à mão não traz perícia.
	defesa := engine.ManeuverSide{Bonus: victim.Melee + victim.ManeuverDefense[req.Maneuver]}
	ataque := engine.ManeuverSide{
		Bonus:  weapon.Attack + striker.ManeuverOffense[req.Maneuver],
		Ranged: !weapon.IsMelee(),
	}

	opposed, err := s.d20Of(req.OpposedD20)
	if err != nil {
		return live.PendingAttack{}, err
	}
	out := engine.ResolveManeuver(req.Maneuver, ataque, defesa, d20, opposed)
	if out.Refused != "" {
		return live.PendingAttack{}, fmt.Errorf("%s: %w", out.Refused, app.ErrRefused)
	}

	pending := live.PendingAttack{
		AttackerEntryID: setup.attacker.ID, TargetEntryID: setup.targetID, Weapon: weapon.Name,
		Roll: out.AttackerRoll, Total: out.AttackerTotal, ByUserID: who.ID, WeaponID: weapon.CatalogID,
		AttackerStances: striker.Stances, TargetStances: victim.Stances,
		Maneuver: &live.ManeuverRoll{
			Kind: out.Kind, OpposedRoll: out.DefenderRoll, Opposed: out.DefenderTotal,
			Margin: out.Margin, ConditionOnAWin: out.ConditionOnAWin,
		},
		// A MANOBRA EM SEGUIDA é livre, e não a ação agredir: ela não abre o extra nem o estilo.
		BoostPM: setup.boost.pm, FollowUps: setup.followUps, FollowUpOf: setup.followUpOf, OpensExtra: setup.followUpOf == nil,
	}
	if pending.OpensExtra {
		opensTheStyle(&pending, weapon)
	}
	return pending, nil
}
