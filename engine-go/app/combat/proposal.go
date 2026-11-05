package combat

import (
	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// proposalOf é a proposta inteira: o que o teste deu, a dose em jogo e o golpe congelado,
// que é de onde a decisão, o dano e o rolar de novo leem.
func (setup strikeSetup) proposalOf(who app.Caller, req Request, out engine.AttackOutcome) (live.PendingAttack, error) {
	pending := setup.pendingOf(who, req, out)
	var err error
	if pending.Dose, err = setup.doseOf(req); err != nil {
		return live.PendingAttack{}, err
	}
	if pending.Frozen, err = engine.FreezeStrike(setup.weapon, setup.target, setup.situations); err != nil {
		return live.PendingAttack{}, err
	}
	return pending, nil
}

// pendingOf é o provisório do golpe, com o que o pedido escolheu e o teste
// devolveu. O dano fica de fora: ele é rolado depois da decisão do mestre.
func (setup strikeSetup) pendingOf(who app.Caller, req Request, out engine.AttackOutcome) live.PendingAttack {
	margin, multiplier := engine.CriticalOf(setup.weapon)
	pending := live.PendingAttack{
		AttackerEntryID: setup.attacker.ID, TargetEntryID: setup.targetID, Weapon: setup.weapon.Name,
		// A DEFESA do provisório é a que o ataque ENFRENTOU, já com a Tabela
		// 5-3 — não a da ficha. "Errei por 1" e "errei por 1 porque ele está
		// atrás da carroça" são leituras diferentes do mesmo número.
		Roll: out.Roll, Total: out.Total, Defense: out.Defense, Situations: out.Situations,
		Hit: out.Hit, Critical: out.Critical, CritRange: margin, CritMult: multiplier,
		Melee: setup.weapon.IsMelee(), ByUserID: who.ID,
		BoostPowerID: setup.boost.powerID, BoostPM: setup.boost.pm, Charge: req.Charge,
		WeaponID: setup.weapon.CatalogID, AttackerStances: setup.striker.Stances, TargetStances: setup.victim.Stances,
	}
	if natural := setup.weapon.Natural; natural != nil {
		pending.NaturalSource = natural.SourceID
		if req.ExtraStrike {
			pending.ExtraOf, pending.ExtraLimit = natural.SourceID, natural.ExtraAttacks
		}
	}
	// A INVESTIDA É OUTRA AÇÃO (p235), e o extra não é a ação agredir.
	pending.OpensExtra = req.Charge == "" && !req.ExtraStrike
	if req.TargetObject != nil {
		pending.TargetTokenID, pending.TargetLabel = req.TargetObject.TokenID, req.TargetObject.Label
	}
	return pending
}
