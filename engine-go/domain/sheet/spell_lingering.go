package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
)

// LingeringRoll é o dado que esta magia deixa na criatura atingida, rolado na vez dela (ver GLOSSARY, **marca na
// fila**): as parcelas do verbete com o que os aumentos escolhidos ao conjurar somam a ELE (`LingerAdds`), pela
// regra do `Adds`. O aprimoramento que muda o dado troca o da conjuração, e não o que fica no alvo.
//
//	sheet.CasterOf(dto).Spell("lanca-ignea-de-aleph").LingeringRoll([]sheet.AugmentPick{{AugmentIndex: 0, Stacks: 1}}) // 3d6 fogo
func (s SpellOnTheSheet) LingeringRoll(picks []AugmentPick) ([]book.SpellRoll, error) {
	lingering := s.Spell.Lingering
	if lingering == nil {
		return nil, fmt.Errorf("%s não deixa nada na criatura atingida", s.Spell.Name)
	}
	parts := slices.Clone(lingering.Rolls)
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) || p.Stacks < 1 {
			continue
		}
		for _, add := range s.Spell.Augments[p.AugmentIndex].LingerAdds {
			var err error
			if parts, err = addToTheRoll(parts, add, p.Stacks); err != nil {
				return nil, fmt.Errorf("%s, aprimoramento %d: %w", s.Spell.Name, p.AugmentIndex, err)
			}
		}
	}
	return parts, nil
}
