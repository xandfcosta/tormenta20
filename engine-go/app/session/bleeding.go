package session

import (
	"context"
	"errors"
	"fmt"

	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/infra/events"
)

// O TESTE DE QUEM SANGRA, no início da vez dele (T20 p236). O jogador rola e
// digita (decisão do dono, ALE-366): esta camada abre o teste no giro da vez,
// recebe os dados, soma a Constituição da ficha e aplica o resultado. A REGRA —
// alcançar 15 estabiliza — é do motor.

// ErrNoBleedingCheck é mandar um dado que a vez não pediu.
var ErrNoBleedingCheck = errors.New("não há teste de sangramento esperando este dado")

// openBleedingCheck abre o teste se quem entrou na vez sangra.
//
// O ERRO é engolido pela razão que o `payUpkeep` explica: uma mesa travada no
// turno de alguém é pior que um teste que não abriu, e o mestre pode tirar o
// Sangrando à mão.
func (st *Store) openBleedingCheck(u Unit, s *live.SessionRuntimeState) error {
	if !s.Scene.CountsRounds() || u.TurnEffects == nil {
		return nil
	}
	if s.TurnIndex < 0 || s.TurnIndex >= len(s.Initiative) {
		return nil
	}
	entry := s.Initiative[s.TurnIndex]
	if entry.CharacterID == nil {
		return nil
	}
	// A LEITURA QUE FALHA RECUSA A VEZ (ALE-373): sem as condições não dá para
	// saber se alguém entra na vez sangrando, e passar a vez assim mesmo é
	// afirmar que ninguém estava.
	conditions, err := u.TurnEffects.ConditionsOf(context.Background(), *entry.CharacterID)
	if err != nil {
		return fmt.Errorf("ler as condições de %s: %w", entry.Label, err)
	}
	for _, c := range conditions {
		if c == engine.ConditionBleeding {
			s.Scene.Bleeding = &live.BleedingCheck{EntryID: entry.ID, CharacterID: *entry.CharacterID, Label: entry.Label}
			return nil
		}
	}
	return nil
}

// RollBleedingD20 recebe o d20 do teste: alcançar 15 com a Constituição
// estabiliza; ficar abaixo pede o d6.
func (st *Store) RollBleedingD20(sessionID int64, d20 int) (*live.SessionRuntimeState, error) {
	if d20 < 1 || d20 > 20 {
		return nil, fmt.Errorf("o d20 do teste de Constituição vai de 1 a 20, e veio %d", d20)
	}
	check := st.GetState(sessionID).Scene.PendingBleeding(false)
	if check == nil {
		return nil, fmt.Errorf("%w (o d20)", ErrNoBleedingCheck)
	}
	con, err := st.turnEffects.ConstitutionOf(context.Background(), check.CharacterID)
	if err != nil {
		return nil, fmt.Errorf("ler a Constituição de %s: %w", check.Label, err)
	}
	total := d20 + con
	passed := engine.BleedingCheckPasses(d20, con)
	if passed {
		if err := st.turnEffects.StabilizeBleeding(context.Background(), check.CharacterID); err != nil {
			return nil, fmt.Errorf("estabilizar %s: %w", check.Label, err)
		}
	}
	return st.apply(sessionID, events.TurnAdvanced{SessionID: sessionID}, func(s *live.SessionRuntimeState) error {
		c := s.Scene.PendingBleeding(false)
		if c == nil || c.CharacterID != check.CharacterID {
			return fmt.Errorf("%w (a vez mudou)", ErrNoBleedingCheck)
		}
		c.Roll, c.Total = d20, total
		if passed {
			c.Outcome = fmt.Sprintf("%s estabilizou (%d + %d = %d contra %d)", c.Label, d20, con, total, engine.BleedingCheckDC)
			return nil
		}
		c.AwaitingD6 = true
		return nil
	})
}

// RollBleedingD6 recebe o d6 da falha e o tira do PV pelo caminho de toda
// pancada — ficha e espelho da fila —, que já sabe matar no limiar.
func (st *Store) RollBleedingD6(sessionID int64, d6 int) (*live.SessionRuntimeState, error) {
	if d6 < 1 || d6 > 6 {
		return nil, fmt.Errorf("o d6 do dano vai de 1 a 6, e veio %d", d6)
	}
	check := st.GetState(sessionID).Scene.PendingBleeding(true)
	if check == nil {
		return nil, fmt.Errorf("%w (o d6)", ErrNoBleedingCheck)
	}
	before := st.hitPointsOf(check.CharacterID)
	if _, err := st.DeltaCharacterVitals(sessionID, check.CharacterID, live.PtrInt64(int64(-d6)), nil); err != nil {
		return nil, fmt.Errorf("tirar o d6 de %s: %w", check.Label, err)
	}
	after := st.hitPointsOf(check.CharacterID)
	return st.apply(sessionID, events.TurnAdvanced{SessionID: sessionID}, func(s *live.SessionRuntimeState) error {
		c := s.Scene.PendingBleeding(true)
		if c == nil || c.CharacterID != check.CharacterID {
			return fmt.Errorf("%w (a vez mudou)", ErrNoBleedingCheck)
		}
		c.AwaitingD6 = false
		c.Outcome = fmt.Sprintf("%s perdeu %d PV (%d → %d)", c.Label, d6, before, after)
		return nil
	})
}

// hitPointsOf lê o PV atual da ficha, para a frase do resultado.
func (st *Store) hitPointsOf(characterID int64) int64 {
	if st.sheet == nil {
		return 0
	}
	pools, err := st.sheet.PoolsOf(context.Background(), []int64{characterID})
	if err != nil {
		return 0
	}
	return pools[characterID].HpCurrent
}
