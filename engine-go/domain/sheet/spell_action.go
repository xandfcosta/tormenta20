package sheet

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
)

// Actions são as ações que esta magia em vigor dá a quem a conjurou (ver GLOSSARY, **ação da magia**): as da
// própria magia, ou as do aprimoramento escolhido ao conjurar que as cria (os dardos da Lança Ígnea). Nil na que
// não dá nenhuma.
func (s SpellOnTheSheet) Actions(picks []AugmentPick) *book.SpellActions {
	if s.Spell.Actions != nil {
		return s.Spell.Actions
	}
	if a := s.chargingAugment(picks); a != nil {
		return a.Actions
	}
	return nil
}

// Charges são as cargas que o aprimoramento escolhido cria, zero sem ele.
func (s SpellOnTheSheet) Charges(picks []AugmentPick) int {
	if a := s.chargingAugment(picks); a != nil {
		return a.Charges
	}
	return 0
}

// chargingAugment é o aprimoramento escolhido que cria cargas com ações, ou nil.
func (s SpellOnTheSheet) chargingAugment(picks []AugmentPick) *book.SpellAugment {
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) || p.Stacks < 1 {
			continue
		}
		if a := s.Spell.Augments[p.AugmentIndex]; a.Charges > 0 && a.Actions != nil {
			return &a
		}
	}
	return nil
}

// ActionRoll é a opção pedida das ações desta magia e o dado dela, com os aprimoramentos escolhidos ao conjurar,
// pela regra do dado da conjuração (`CastRoll`). A opção que rola "o efeito normal da magia" rola o dado da
// conjuração sem as cargas — o dardo é a Lança Ígnea que não saiu ao conjurar. A opção que é MANOBRA não rola dado:
// ela volta sem parcelas, com o bônus que o aprimoramento escolhido pôs no lugar (`ManeuverBonus`).
//
//	action, parts, err := sheet.CasterOf(dto).Spell("mao-poderosa-de-talude").ActionRoll(picks, "Esmagar")
func (s SpellOnTheSheet) ActionRoll(picks []AugmentPick, option string) (book.SpellAction, []book.SpellRoll, error) {
	actions := s.Actions(picks)
	if actions == nil {
		return book.SpellAction{}, nil, fmt.Errorf("%s, como foi conjurada, não dá ação nenhuma depois", s.Spell.Name)
	}
	names := make([]string, 0, len(actions.Options))
	for _, action := range actions.Options {
		names = append(names, action.Name)
		if action.Name != option {
			continue
		}
		if action.Maneuver != "" {
			for _, p := range picks {
				if p.AugmentIndex >= 0 && p.AugmentIndex < len(s.Spell.Augments) && s.Spell.Augments[p.AugmentIndex].ManeuverBonus > 0 {
					action.Bonus = s.Spell.Augments[p.AugmentIndex].ManeuverBonus
				}
			}
			return action, nil, nil
		}
		base := action.Rolls
		if action.CastRoll {
			base = s.Spell.Rolls
		}
		roll, err := s.augmentedRoll(base, picks)
		if err != nil {
			return book.SpellAction{}, nil, err
		}
		if roll.ByText {
			return book.SpellAction{}, nil, fmt.Errorf("um aprimoramento de %s mudou o dado de %s para um que se rola pelo texto dele", s.Spell.Name, action.Name)
		}
		return action, roll.Parts, nil
	}
	return book.SpellAction{}, nil, fmt.Errorf("%s não tem a ação %q; tem %s", s.Spell.Name, option, strings.Join(names, ", "))
}
