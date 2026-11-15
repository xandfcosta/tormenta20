package session

import (
	"context"

	"t20engine/domain/live"
	"t20engine/infra/events"
)

// O QUE A DECISÃO DO MESTRE ABRE (ALE-659).
//
// Um gesto pode depender do desfecho de outro: o ataque extra vem de agredir (p25), a derrubada do Lutador de
// Chão vem do agarrar que venceu (p77), o agarrar da Mão na Boca vem do golpe furtivo que acertou (p74). Nada
// disso se rola antes de valer a condição — e quem diz o desfecho é o mestre. Por isso o que abre mora num lugar
// só, lido da decisão gravada na proposta, e não em cada gesto que encerra o golpe.
//
// A decisão chega por três portas — o erro do golpe (`RuleAttack`), o ✓ do acerto (`CommitAttack`) e o Sucesso
// ou a Falha da manobra (`RuleManeuver`) —, e as três chamam `opensWhatTheRulingOpens`.

// opensWhatTheRulingOpens abre na vez o que depende da decisão sobre este gesto, se a vez ainda é a do turno em
// que ele nasceu:
//
//   - o ataque extra (p25), o adicional do estilo (p125) e o ataque pago (ALE-651), que vêm de QUALQUER resposta
//     a agredir — errar não desfaz o agredir;
//   - a manobra em seguida armada no gesto (`PendingAttack.FollowUps`), quando a decisão é a que ela pede.
func opensWhatTheRulingOpens(s *live.SessionRuntimeState, attack live.PendingAttack) {
	if !paidInThisTurn(s, attack) {
		return
	}
	if attack.OpensExtra {
		s.Scene.ExtraStrikeEntry, s.Scene.ExtraStrikeBarred = attack.AttackerEntryID, attack.NaturalSource
		s.Scene.ExtraOpenedUnarmed = attack.OpensUnarmed
		s.Scene.ExtraStyle, s.Scene.ExtraStyleItem = attack.OpensStyle, attack.StyleItem
	}
	if attack.TargetEntryID == "" {
		return
	}
	for _, armed := range attack.FollowUps {
		if live.RulingOpens(attack.Ruling, armed.On) {
			s.Scene.FollowUpOpen = &live.OpenFollowUp{FollowUp: armed, EntryID: attack.AttackerEntryID, TargetID: attack.TargetEntryID}
			return
		}
	}
}

// RuleManeuver guarda o Sucesso ou a Falha da manobra e a encerra: ela não rola dano, e a decisão é o fim dela.
// A Falha mantém a ação gasta, como o erro do golpe; o ✕ do mestre continua sendo o cancelamento que devolve.
func (st *Store) RuleManeuver(ctx context.Context, sessionID int64, who live.Attacker, ruling string) (*live.SessionRuntimeState, error) {
	return st.apply(ctx, sessionID, events.AttackSettled{SessionID: sessionID},
		func(s *live.SessionRuntimeState) error {
			if err := live.RuleManeuver(s, who, ruling); err != nil {
				return err
			}
			opensWhatTheRulingOpens(s, *s.PendingAttack)
			live.ClearPendingAttack(s)
			return nil
		})
}
