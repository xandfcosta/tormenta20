package session

import (
	"t20engine/domain/engine"
	"t20engine/domain/live"
)

// OS VITAIS DA FILA com o piso do livro. O PV desce abaixo de zero até o limiar
// da morte (p236), e esse número é do motor, que o regime não alcança — então
// esta camada, que conhece os dois, o calcula e o entrega (ALE-366).

// patchEntryVitals é a mutação de "PV/PM absolutos" com o piso da linha.
func patchEntryVitals(entryID string, hp, mp *int64) func(*live.SessionRuntimeState) error {
	return func(s *live.SessionRuntimeState) error {
		return live.PatchEntryVitals(s, entryID, hp, mp, hpFloorOf(s, entryID))
	}
}

// deltaEntryVitals é a mutação de "levou N" com o piso da linha.
func deltaEntryVitals(entryID string, hpDelta, mpDelta *int64) func(*live.SessionRuntimeState) error {
	return func(s *live.SessionRuntimeState) error {
		return live.DeltaEntryVitals(s, entryID, hpDelta, mpDelta, hpFloorOf(s, entryID))
	}
}

// hpFloorOf é o PV mais baixo a que esta linha pode chegar. Linha sem PV máximo
// — o capanga digitado na hora — cai no –10, que é o único número que o livro
// dá para quem não tem total conhecido.
func hpFloorOf(s *live.SessionRuntimeState, entryID string) int64 {
	idx := live.FindEntryIndex(s, entryID)
	if idx < 0 {
		return engine.DeathThreshold(0)
	}
	return engine.DeathThreshold(live.DerefOr(s.Initiative[idx].HpMax, 0))
}
