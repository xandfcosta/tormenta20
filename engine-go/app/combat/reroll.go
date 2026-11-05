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

// Reroll troca o d20 da proposta por outro e a guarda de novo, que é quando `paid` sai da
// ficha. O d20 da mesa vem em `given`, ou o servidor rola; `naturalFrom` é o "11 ou mais vale 20 natural" da Sorte de
// Nimb (p48), zero nos outros.
//
// SÓ O DADO MUDA, e é a única mudança que a proposta aceita: os bônus são os de quando
// ela foi feita, mesmo que a ficha tenha mudado depois.
//
// SÓ ANTES DA DECISÃO DO MESTRE (p226): depois dela, o teste já foi julgado.
// E não há dano a guardar — ele é rolado depois da decisão.
//
//	pending, err := strike.Reroll(ctx, who, sessionID, nil, live.DieReroll{Name: "Sortudo", PM: 3}, 0)
func (s Strike) Reroll(
	ctx context.Context, who app.Caller, sessionID int64,
	given *int, paid live.DieReroll, naturalFrom int,
) (live.PendingAttack, error) {
	state, err := s.tables.State(ctx, sessionID)
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
	natural := naturalFrom > 0 && d20 >= naturalFrom && d20 != 20
	second, err := s.withAnotherDie(first, d20, natural)
	if err != nil {
		return live.PendingAttack{}, err
	}
	paid.Discarded, paid.Natural = first.Roll, natural
	second.Rerolls = append(slices.Clone(first.Rerolls), paid)
	if _, err := s.tables.RerollAttack(ctx, sessionID, second); err != nil {
		return live.PendingAttack{}, err
	}
	return second, nil
}

// withAnotherDie é a proposta com o d20 novo. Na manobra, os bônus saem dos totais
// guardados: o total menos o dado, dos dois lados, e o d20 do defensor fica (p234).
func (s Strike) withAnotherDie(first live.PendingAttack, d20 int, natural bool) (live.PendingAttack, error) {
	second := first
	if m := first.Maneuver; m != nil {
		out := engine.ResolveManeuver(m.Kind,
			engine.ManeuverSide{Bonus: first.Total - first.Roll},
			engine.ManeuverSide{Bonus: m.Opposed - m.OpposedRoll}, d20, m.OpposedRoll)
		if out.Refused != "" {
			return live.PendingAttack{}, fmt.Errorf("%s: %w", out.Refused, app.ErrRefused)
		}
		rolled := *m
		rolled.Margin = out.Margin
		second.Maneuver = &rolled
		second.Roll, second.Total = out.AttackerRoll, out.AttackerTotal
		return second, nil
	}
	frozen, err := strikeProposedIn(first)
	if err != nil {
		return live.PendingAttack{}, err
	}
	die := engine.D20(d20)
	if natural {
		die = engine.D20CountingAsNaturalTwenty(d20)
	}
	out, err := engine.JudgeAttack(frozen.Weapon, frozen.Target, frozen.Situations, die, s.rollDie)
	if err != nil {
		return live.PendingAttack{}, fmt.Errorf("%w: %w", err, app.ErrRefused)
	}
	second.Roll, second.Total, second.Defense, second.Situations = out.Roll, out.Total, out.Defense, out.Situations
	second.Hit, second.Critical = out.Hit, out.Critical
	return second, nil
}
