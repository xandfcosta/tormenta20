package character

import (
	"context"
	"fmt"

	"t20engine/domain/book"
	"t20engine/domain/sheet"
	"t20engine/infra/db/sqlcgen"
)

// SpellActionOf é a opção `option` (o índice dela) das ações da magia em vigor na linha `effectID` (ver GLOSSARY,
// **ação da magia**), com o dado já montado com os aprimoramentos da conjuração. Quem cobra a ação da vez é quem
// chama, ANTES de `ActOnSpell`: a opção diz quanto ela custa e se é uma vez por turno.
func SpellActionOf(dto sheet.CharacterDTO, effectID int64, option int) (SpellActionAsked, error) {
	var effect *sheet.EffectDTO
	for i := range dto.ActiveEffects {
		if dto.ActiveEffects[i].ID == effectID {
			effect = &dto.ActiveEffects[i]
		}
	}
	if effect == nil {
		return SpellActionAsked{}, fmt.Errorf("o efeito %d não está mais em vigor na ficha de %s", effectID, dto.Name)
	}
	onSheet := sheet.CasterOf(dto).Spell(effect.CatalogID)
	actions := onSheet.Actions(effect.CastAugments)
	if !onSheet.Known || actions == nil {
		return SpellActionAsked{}, fmt.Errorf("o efeito em vigor %q não dá ação nenhuma", effect.CatalogID)
	}
	if option < 0 || option >= len(actions.Options) {
		return SpellActionAsked{}, fmt.Errorf("%s tem %d ação(ões), e o pedido foi a de índice %d", onSheet.Spell.Name, len(actions.Options), option)
	}
	action, parts, err := onSheet.ActionRoll(effect.CastAugments, actions.Options[option].Name)
	if err != nil {
		return SpellActionAsked{}, err
	}
	if onSheet.Charges(effect.CastAugments) > 0 && effect.Charges <= 0 {
		return SpellActionAsked{}, fmt.Errorf("%s já foi descarregada", onSheet.Spell.Name)
	}
	return SpellActionAsked{
		Effect: *effect, Spell: onSheet, Action: action, Parts: parts, OncePerTurn: actions.OncePerTurn,
	}, nil
}

// SpellActionAsked é a ação da magia pedida, pronta para rolar.
type SpellActionAsked struct {
	Effect      sheet.EffectDTO
	Spell       sheet.SpellOnTheSheet
	Action      book.SpellAction
	Parts       []book.SpellRoll
	OncePerTurn bool
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
	cast, err := spellCastOf(dto, asked.Spell)
	if err != nil {
		return SpellCast{}, err
	}
	cast.When = asked.Action.Name
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
	ends := asked.Action.Ends
	if asked.Spell.Charges(asked.Effect.CastAugments) > 0 {
		left := asked.Effect.Charges - 1
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
