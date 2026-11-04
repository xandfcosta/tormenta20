package session

import (
	"context"

	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/dbvalue"
	"t20engine/infra/db/sqlcgen"
	"t20engine/infra/events"
)

// A DOSE DA INJEÇÃO ALQUÍMICA NO GOLPE (p165), e são duas decisões do dono: ela sai
// quando o mestre diz que acertou, e volta se o ataque for cancelado depois disso.
//
// A mesa e a ficha gravam na mesma transação, e o que fazer com a dose se decide
// dentro dela, sob a trava: conferido antes, dois toques no "acertou" gastariam duas.

// withTheDoseSpent grava a decisão de acerto e tira a dose da arma. Se ela já não
// estiver lá, o golpe segue sem dose: o acerto é do mestre, e não depende dela.
func (st *Store) withTheDoseSpent(
	ctx context.Context, sessionID int64, rule func(*live.SessionRuntimeState) error,
) (*live.SessionRuntimeState, error) {
	var moved *int64
	clone, err := st.unitGesture(ctx, sessionID,
		func(inUnit context.Context, u Unit, s *live.SessionRuntimeState) error {
			if err := rule(s); err != nil {
				return err
			}
			dose := s.PendingAttack.Dose
			if dose == nil {
				return nil
			}
			spent, err := u.InTx.SpendItemDose(inUnit, sqlcgen.SpendItemDoseParams{ItemId: dose.ItemID, CatalogId: dose.CatalogID})
			if err != nil {
				return err
			}
			if spent == 0 {
				s.PendingAttack.Dose = nil
				return nil
			}
			dose.Spent = true
			moved = attackerCharacterOf(s)
			return stampTheSheet(inUnit, u, moved)
		})
	if err != nil {
		return nil, err
	}
	st.bus.Publish(events.AttackRolled{SessionID: sessionID})
	st.wakeTheSheet(moved)
	return clone, nil
}

// returnsTheDose devolve à arma a dose que o acerto gastou.
func returnsTheDose(ctx context.Context, u Unit, dose *live.AttackDose) error {
	if dose == nil || !dose.Spent {
		return nil
	}
	return u.InTx.ReturnItemDose(ctx, sqlcgen.ReturnItemDoseParams{
		CatalogId: dose.CatalogID, ItemId: dose.ItemID, Improvement: sheet.AlchemicalInjection,
	})
}

func attackerCharacterOf(s *live.SessionRuntimeState) *int64 {
	idx := live.FindEntryIndex(s, s.PendingAttack.AttackerEntryID)
	if idx < 0 {
		return nil
	}
	return s.Initiative[idx].CharacterID
}

// stampTheSheet e wakeTheSheet são o que faz a linha da arma, nas Ações, redesenhar: o
// gesto veio da mesa, então não passa pelo aviso do `sheetCommand`, e a mesa do
// jogador repede a ficha quando o carimbo dela muda.
func stampTheSheet(ctx context.Context, u Unit, characterID *int64) error {
	if characterID == nil {
		return nil
	}
	return u.InTx.TouchCharacter(ctx, sqlcgen.TouchCharacterParams{UpdatedAt: dbvalue.NowISO(), ID: *characterID})
}

func (st *Store) wakeTheSheet(characterID *int64) {
	if characterID != nil {
		st.bus.Publish(events.CharacterChanged{CharacterID: *characterID})
	}
}
