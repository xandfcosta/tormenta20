package combat

import (
	"context"
	"fmt"
	"slices"

	"t20engine/app"
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// ROLAR DE NOVO O ATAQUE PROVISÓRIO (p226, ALE-508): "Efeitos que permitem que
// você role novamente o dado devem ser usados antes de o mestre declarar se
// você passou ou não no teste (e você deve ficar com o segundo valor rolado,
// mesmo que seja pior que o primeiro)." O provisório é a janela: depois da
// confirmação ele não existe mais.

// resolver é o motor do golpe: o comum, ou o que conta o d20 como 20 natural.
func (r Request) resolver() func(
	engine.WeaponCard, engine.AttackTarget, []engine.SpecialSituation, int, func(int) (int, error),
) (engine.AttackOutcome, error) {
	if r.CountsAsNaturalTwenty {
		return engine.ResolveAttackCountingAsNaturalTwenty
	}
	return engine.ResolveAttackUnder
}

// replayOf é o que o provisório guarda do pedido para rolar de novo.
func (r Request) replayOf(weapon engine.WeaponCard) live.AttackReplay {
	return live.AttackReplay{
		WeaponIndex: r.Weapon, WeaponID: weapon.CatalogID, SwitchesDamageType: r.SwitchesDamageType,
		BoostSteps: r.Boost.Steps, BoostSplit: r.Boost.Split,
	}
}

// RerollRequest é o pedido que refaz o provisório: o mesmo alvo, a mesma arma,
// o mesmo impulso e a mesma investida, e o d20 do defensor da manobra intacto.
// O que só a cena sabe — a posse e a peça atacada — vem de quem chama.
func RerollRequest(
	campaignID, sessionID int64, pending live.PendingAttack, ownsAttacker bool, object *ObjectTarget,
) Request {
	req := Request{
		CampaignID: campaignID, SessionID: sessionID,
		AttackerEntryID: pending.AttackerEntryID, TargetEntryID: pending.TargetEntryID,
		TargetObject: object, OwnsAttacker: ownsAttacker,
		Weapon: pending.Replay.WeaponIndex, SwitchesDamageType: pending.Replay.SwitchesDamageType,
		Charge: pending.Charge,
	}
	if pending.BoostPowerID != "" {
		req.Boost = engine.BoostChoice{
			PowerID: pending.BoostPowerID, Steps: pending.Replay.BoostSteps, Split: pending.Replay.BoostSplit,
		}
	}
	if pending.Maneuver != nil {
		opposed := pending.Maneuver.OpposedRoll
		req.Maneuver, req.OpposedD20 = pending.Maneuver.Kind, &opposed
	}
	return req
}

// Reroll rola o provisório de novo com outro d20 e o guarda no lugar do
// primeiro. O d20 da mesa vem em `given`, ou o servidor rola; `naturalFrom` é
// o "11 ou mais vale 20 natural" da Sorte de Nimb (p48), zero nos outros.
//
// O DANO JÁ ROLADO FICA quando o segundo dado não muda o acerto nem o crítico:
// rolar de novo o ATAQUE não é rolar de novo o dano. Quando muda, o golpe é o
// que o segundo dado diz, com o dano que ele pede (decisão do dono).
//
//	pending, err := strike.Reroll(ctx, who, role, req, nil, live.DieReroll{Name: "Sortudo", PM: 3}, 0)
func (s Strike) Reroll(
	ctx context.Context, who app.Caller, role string, req Request,
	given *int, paid live.DieReroll, naturalFrom int,
) (live.PendingAttack, error) {
	state, err := s.tables.State(ctx, req.SessionID)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if state == nil || state.PendingAttack == nil {
		return live.PendingAttack{}, fmt.Errorf("não há ataque provisório para rolar de novo: %w", app.ErrRefused)
	}
	first := *state.PendingAttack
	if first.ByUserID != who.ID {
		return live.PendingAttack{}, fmt.Errorf("só quem rolou o ataque rola de novo: %w", app.ErrRefused)
	}
	d20, err := s.d20Of(given)
	if err != nil {
		return live.PendingAttack{}, err
	}
	req.D20 = &d20
	req.CountsAsNaturalTwenty = naturalFrom > 0 && d20 >= naturalFrom && d20 != 20
	second, err := s.roll(ctx, who, role, req)
	if err != nil {
		return live.PendingAttack{}, err
	}
	if second.Maneuver == nil && second.Hit == first.Hit && second.Critical == first.Critical {
		kept := first
		kept.Roll, kept.Total = second.Roll, second.Total
		second = kept
	}
	paid.Discarded, paid.Natural = first.Roll, req.CountsAsNaturalTwenty
	second.Rerolls = append(slices.Clone(first.Rerolls), paid)
	if _, err := s.tables.ProposeAttack(ctx, req.SessionID, second); err != nil {
		return live.PendingAttack{}, err
	}
	return second, nil
}
