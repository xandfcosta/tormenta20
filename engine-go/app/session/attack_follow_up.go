package session

import (
	"fmt"

	"t20engine/domain/live"
)

// A MANOBRA EM SEGUIDA (ALE-651, `engine.FollowUpManeuver`): o gesto que a declara a leva na proposta, a decisão
// do mestre sobre ele a abre na vez (`opensWhatTheRulingOpens`), e ela sai livre — sem gastar ação —, uma por
// gesto declarado e contra o mesmo alvo.

// takesTheFollowUp confere e gasta a manobra em seguida aberta, sob a trava. Ela não gasta ação.
func takesTheFollowUp(s *live.SessionRuntimeState, attack *live.PendingAttack) error {
	taken := attack.FollowUpOf
	if !s.CountsRounds() || !attackerIsOnTurn(s, *attack) {
		return fmt.Errorf("%s: a manobra em seguida é na vez de quem a declarou", taken.Name)
	}
	open := s.Scene.FollowUpOpen
	if open == nil || open.EntryID != attack.AttackerEntryID || open.PowerID != taken.PowerID {
		return fmt.Errorf("%s: a manobra em seguida vem depois de um gesto que a declarou nesta vez, e da decisão do mestre sobre ele", taken.Name)
	}
	if open.TargetID != attack.TargetEntryID {
		return fmt.Errorf("%s: a manobra em seguida é contra o mesmo alvo do gesto que a declarou", taken.Name)
	}
	s.Scene.FollowUpOpen = nil
	return nil
}

// givesTheFollowUpBack reabre a manobra em seguida que o cancelamento do mestre desfez: ela não aconteceu.
func givesTheFollowUpBack(s *live.SessionRuntimeState, open live.PendingAttack) {
	if taken := open.FollowUpOf; taken != nil {
		s.Scene.FollowUpOpen = &live.OpenFollowUp{FollowUp: *taken, EntryID: open.AttackerEntryID, TargetID: open.TargetEntryID}
	}
}
