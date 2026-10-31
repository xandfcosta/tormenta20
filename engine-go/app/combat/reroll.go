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

// attackDie é o d20 do golpe: o rolado, ou o que conta como 20 natural.
func (r Request) attackDie(d20 int) engine.AttackDie {
	if r.CountsAsNaturalTwenty {
		return engine.D20CountingAsNaturalTwenty(d20)
	}
	return engine.D20(d20)
}

// replayOf é o que o provisório guarda do pedido para rolar de novo.
func (r Request) replayOf(weapon engine.WeaponCard) live.AttackReplay {
	return live.AttackReplay{
		WeaponIndex: r.Weapon, WeaponID: weapon.CatalogID, SwitchesDamageType: r.SwitchesDamageType,
		BoostSteps: r.Boost.Steps, BoostSplit: r.Boost.Split,
	}
}

// ReplayRequest é o pedido que REFAZ o provisório — para rolar de novo, para a
// decisão do mestre e para o dano: o mesmo alvo, a mesma arma,
// o mesmo impulso e a mesma investida, e o d20 do defensor da manobra intacto.
// O que só a cena sabe — a posse e a peça atacada — vem de quem chama.
func ReplayRequest(
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
// SÓ ANTES DA DECISÃO DO MESTRE (p226): depois dela, o teste já foi julgado.
// E não há dano a guardar — ele é rolado depois da decisão.
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
	if first.Ruling != "" {
		return live.PendingAttack{}, fmt.Errorf(
			"o mestre já decidiu este ataque, e rolar de novo vale antes da decisão (p226): %w", app.ErrRefused)
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
	paid.Discarded, paid.Natural = first.Roll, req.CountsAsNaturalTwenty
	second.Rerolls = append(slices.Clone(first.Rerolls), paid)
	if _, err := s.tables.ProposeAttack(ctx, req.SessionID, second); err != nil {
		return live.PendingAttack{}, err
	}
	return second, nil
}
