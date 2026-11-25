package character

import (
	"context"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/live"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// SpellActionOf é a opção `option` (o índice dela) das ações da magia em vigor na linha `effectID` (ver GLOSSARY,
// **ação da magia**), com o dado já montado com os aprimoramentos da conjuração. Quem cobra a ação da vez é quem
// chama, ANTES de `ActOnSpell`: a opção diz quanto ela custa e se é uma vez por turno.
func SpellActionOf(dto sheet.CharacterDTO, effectID int64, option int) (SpellActionAsked, error) {
	spellEffect, err := sheet.CasterOf(dto).SpellEffect(effectID)
	if err != nil {
		return SpellActionAsked{}, err
	}
	action, parts, err := spellEffect.Action(option)
	if err != nil {
		return SpellActionAsked{}, err
	}
	return SpellActionAsked{
		SpellEffect: spellEffect, Action: action, Parts: parts, OncePerTurn: spellEffect.Actions().OncePerTurn,
	}, nil
}

// SpellActionAsked é a ação da magia pedida, pronta para rolar: o efeito da magia de onde ela sai, a opção e o dado.
// O campo `Action` é a opção pedida, e cobre o método do efeito que a acha.
type SpellActionAsked struct {
	sheet.SpellEffect
	Action      book.SpellAction
	Parts       []book.SpellRoll
	OncePerTurn bool
}

// ManeuverBonus é o que a mão soma ao d20 da opção que é manobra (`ManeuverBonusOf`), com a ficha computada aqui.
func (a SpellActionAsked) ManeuverBonus(dto sheet.CharacterDTO) (int, error) {
	computed, err := sheet.Compute(dto)
	if err != nil {
		return 0, fmt.Errorf("computar a ficha para a manobra de %s: %w", a.Spell.Spell.Name, err)
	}
	bonus, found := ManeuverBonusOf(computed, a.Action)
	if !found {
		return 0, fmt.Errorf("a ficha não tem a perícia %q que a manobra de %s soma", a.Action.Skill, a.Spell.Spell.Name)
	}
	return bonus, nil
}

// ManeuverBonusOf é o que a mão de uma magia soma ao d20 da manobra: o total da perícia de quem conjurou e o bônus
// da opção — o Misticismo mais 10 da Mão Poderosa (p198). Falso quando a ficha não tem a perícia. A aba Ações e o
// gesto leem a mesma conta.
//
//	bonus, ok := character.ManeuverBonusOf(computed, grab) // Misticismo +10
func ManeuverBonusOf(computed engine.ComputedSheet, action book.SpellAction) (int, bool) {
	total, found := expertiseTotalOf(computed, action.Skill)
	return total + action.Bonus, found
}

// OnceKey é a marca do "uma vez por turno" desta magia na rodada de quem a conjurou, vazia na que não tem o
// limite. A marca é da MAGIA, e não da opção: as quatro tempestades da Fúria dividem a mesma vez.
func (a SpellActionAsked) OnceKey() string {
	if !a.OncePerTurn {
		return ""
	}
	return "magia:" + a.Spell.Spell.ID
}

// ActOnSpell rola a ação da magia pedida, ou confere o total que a mesa rolou na mão, e grava o que ela gasta da
// magia: o dardo sai das cargas, e o último — como a opção que termina a magia (a energia mista, p204) — encerra
// a linha em vigor. A ação da vez é de quem chama.
func (p Plays) ActOnSpell(ctx context.Context, dto sheet.CharacterDTO, asked SpellActionAsked, fromTheTable []int) (SpellCast, error) {
	cast, powers, err := spellCastOf(dto, asked.Spell)
	if err != nil {
		return SpellCast{}, err
	}
	cast.Kind, cast.Option = live.RollOfTheAction, asked.Action.Name
	if asked.Action.Save != "" {
		cast.Save = asked.Action.Save
	}
	parts := asked.Parts
	if asked.Action.Note != "" && len(parts) > 0 && parts[0].Note == "" {
		parts = append([]book.SpellRoll{}, parts...)
		parts[0].Note = asked.Action.Note
	}
	if cast.Parts, err = p.rollTheParts(cast.Name, parts, fromTheTable); err != nil {
		return SpellCast{}, err
	}
	cast.Bonuses = powers.LaterBonuses(parts)
	// A AÇÃO COM O EFEITO NORMAL da magia deixa a marca dela, como a conjuração: o dardo de lava põe o alvo em
	// chamas (p196; ver GLOSSARY, **marca na fila**).
	if asked.Action.CastRoll {
		if cast.Lingering, err = lingeringOf(asked.Spell, asked.Effect.CastAugments, cast, powers); err != nil {
			return SpellCast{}, err
		}
	}
	ends := asked.Action.Ends
	if charges, counted := asked.Charges(); counted {
		left := charges - 1
		if err := p.queries.SetEffectCharges(ctx, sqlcgen.SetEffectChargesParams{Effectid: asked.Effect.ID, Remaining: int64(left)}); err != nil {
			return SpellCast{}, fmt.Errorf("gastar uma carga de %s: %w", asked.Spell.Spell.Name, err)
		}
		ends = ends || left <= 0
	}
	if ends {
		if err := p.EndAppliedEffect(ctx, dto.ID, asked.Effect.ID); err != nil {
			return SpellCast{}, err
		}
	}
	return cast, nil
}
