package live

import "fmt"

// OS VITAIS DA FILA, e a regra deles é o piso do PV.
//
// O PV desce abaixo de zero até o limiar da morte (T20 p236), e o PM não
// (decisão do dono, ALE-366). O limiar é regra do LIVRO e mora no motor, que o
// regime não alcança (ver `boundary_test.go`) — então ele chega aqui como
// PARÂMETRO, calculado por quem conhece os dois: a mesma forma da economia de
// ação, em que o regime guarda e o motor decide.

// PatchEntryVitals grava PV/PM absolutos numa linha, presos ao máximo quando ele
// existe e o PV preso ao `hpFloor`. A escrita no banco é da camada de store.
func PatchEntryVitals(st *SessionRuntimeState, entryID string, hpCurrent, mpCurrent *int64, hpFloor int64) error {
	idx := FindEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("a linha %s não está na fila", entryID)
	}
	e := &st.Initiative[idx]
	if hpCurrent != nil {
		e.HpCurrent = PtrInt64(ClampVital(*hpCurrent, e.HpMax, hpFloor))
	}
	if mpCurrent != nil {
		e.MpCurrent = PtrInt64(ClampVital(*mpCurrent, e.MpMax, 0))
	}
	return nil
}

// DeltaEntryVitals aplica um delta de PV/PM ("sofreu 10 de dano" ⇒ hpDelta -10).
// Atual ausente conta como 0.
func DeltaEntryVitals(st *SessionRuntimeState, entryID string, hpDelta, mpDelta *int64, hpFloor int64) error {
	idx := FindEntryIndex(st, entryID)
	if idx < 0 {
		return fmt.Errorf("a linha %s não está na fila", entryID)
	}
	e := &st.Initiative[idx]
	if hpDelta != nil {
		e.HpCurrent = PtrInt64(ClampVital(DerefOr(e.HpCurrent, 0)+*hpDelta, e.HpMax, hpFloor))
	}
	if mpDelta != nil {
		e.MpCurrent = PtrInt64(ClampVital(DerefOr(e.MpCurrent, 0)+*mpDelta, e.MpMax, 0))
	}
	return nil
}

// ClampVital prende um vital entre o piso e o máximo, quando há máximo. O PM
// passa piso 0; o PV passa o limiar da morte.
func ClampVital(value int64, max *int64, floor int64) int64 {
	floored := value
	if floored < floor {
		floored = floor
	}
	if max == nil {
		return floored
	}
	if floored > *max {
		return *max
	}
	return floored
}
