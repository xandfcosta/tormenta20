package sheetui

import (
	"net/http"

	"t20engine/domain/engine"
	"t20engine/infra/db/sqlcgen"
)

// takeWildShape transforma o druida, com a escolha do diálogo. É ação completa (p62), perguntada ao turno
// antes e cobrada depois, como o `usePower`: a transformação ainda pode ser recusada pelo PM ou pela
// escolha, e cobrar antes tiraria a ação por uma forma que não veio.
func takeWildShape(s Scene, r *http.Request, row sqlcgen.Character, signals Signals) error {
	dto, err := s.deps.LoadCharacter(r.Context(), row)
	if err != nil {
		return err
	}
	if err := s.deps.ActionFitsOnTurn(r.Context(), row.ID, engine.ActionFull); err != nil {
		return err
	}
	shape := engine.WildShape{
		Form: engine.WildForm(signals.WildForm), Tier: engine.WildTier(signals.WildTier),
		DamageType: signals.WildDamage, Movement: engine.WildMovement(signals.WildMove),
	}
	if err := s.plays.TakeWildShape(r.Context(), row, dto, shape); err != nil {
		return err
	}
	return s.deps.SpendActionOnTurn(r.Context(), row.ID, engine.ActionFull)
}

// leaveWildShape volta o druida ao normal, na aba Efeitos, onde a forma em vigor aparece.
func leaveWildShape(s Scene, r *http.Request, row sqlcgen.Character, _ Signals) error {
	return s.plays.LeaveWildShape(r.Context(), row)
}
