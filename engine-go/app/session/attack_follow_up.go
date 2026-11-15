package session

import (
	"fmt"

	"t20engine/domain/live"
)

// A MANOBRA EM SEGUIDA (ALE-651, `engine.FollowUpManeuver`): o gesto rolado a arma na proposta, a decisão do
// mestre sobre ele a abre na vez (`opensWhatTheRulingOpens`), e ela sai livre — sem gastar ação —, uma por gesto
// decidido e contra o mesmo alvo.

// takesTheFollowUp confere e gasta a manobra em seguida aberta, sob a trava. Ela não gasta ação.
func takesTheFollowUp(s *live.SessionRuntimeState, attack *live.PendingAttack) error {
	taken := attack.FollowUpOf
	if !s.CountsRounds() || !attackerIsOnTurn(s, *attack) {
		return fmt.Errorf("%s: a manobra em seguida é na vez de quem a abriu", taken.Name)
	}
	open := s.Scene.FollowUpOpen
	if open == nil || open.EntryID != attack.AttackerEntryID || open.PowerID != taken.PowerID {
		return fmt.Errorf("%s: a manobra em seguida vem %s, nesta vez", taken.Name, taken.When)
	}
	if open.TargetID != attack.TargetEntryID {
		return fmt.Errorf("%s: a manobra em seguida é contra o mesmo alvo do gesto que a abriu", taken.Name)
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
