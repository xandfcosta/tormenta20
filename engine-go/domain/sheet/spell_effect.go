package sheet

import (
	"fmt"

	"t20engine/domain/book"
)

// O EFEITO DA MAGIA, lido (ver GLOSSARY, **efeito da magia**): a linha em vigor de uma magia na ficha, com a magia
// como esta ficha a lança. Cada pergunta sobre ela — o dado da vez, as ações, o dado de uma ação, as cargas —
// responde com os aprimoramentos da conjuração gravada, e quem pergunta não os carrega à mão. A escrita é o
// `writeSpellEffect`, do lado do caso de uso.

// SpellEffect é uma magia em vigor na ficha: a linha e a magia como o conjurador a vê.
type SpellEffect struct {
	Effect EffectDTO
	Spell  SpellOnTheSheet
}

// SpellEffects são as magias em vigor da ficha, na ordem das linhas. A linha que não é magia do livro — a postura,
// o poder, a poção — fica de fora.
//
//	for _, e := range sheet.CasterOf(dto).SpellEffects() { e.Actions() }
func (c Caster) SpellEffects() []SpellEffect {
	var out []SpellEffect
	for _, effect := range c.effects {
		if spell := c.Spell(effect.CatalogID); spell.Known {
			out = append(out, SpellEffect{Effect: effect, Spell: spell})
		}
	}
	return out
}

// SpellEffect é a magia em vigor na linha `effectID`, ou a recusa: a linha que já saiu, ou que não é magia.
//
//	fever, err := sheet.CasterOf(dto).SpellEffect(effectID)
func (c Caster) SpellEffect(effectID int64) (SpellEffect, error) {
	for _, effect := range c.effects {
		if effect.ID != effectID {
			continue
		}
		spell := c.Spell(effect.CatalogID)
		if !spell.Known {
			return SpellEffect{}, fmt.Errorf("o efeito em vigor %q não é uma magia do livro", effect.CatalogID)
		}
		return SpellEffect{Effect: effect, Spell: spell}, nil
	}
	return SpellEffect{}, fmt.Errorf("o efeito %d não está mais em vigor nesta ficha", effectID)
}

// TurnRoll é o dado da vez da versão pedida, com os aprimoramentos da conjuração (ver GLOSSARY, **dado da vez**).
func (e SpellEffect) TurnRoll(version string) ([]book.SpellRoll, error) {
	return e.Spell.TurnRoll(e.Effect.CastAugments, version)
}

// Actions são as ações que a magia em vigor dá, pelos aprimoramentos da conjuração, ou nil (ver GLOSSARY, **ação
// da magia**).
func (e SpellEffect) Actions() *book.SpellActions {
	return e.Spell.Actions(e.Effect.CastAugments)
}

// Action é a opção de índice `option` das ações, com o dado montado com os aprimoramentos da conjuração. A magia
// que conta cargas e já gastou a última é recusada: o dardo que não existe não se dispara.
func (e SpellEffect) Action(option int) (book.SpellAction, []book.SpellRoll, error) {
	actions := e.Actions()
	if actions == nil {
		return book.SpellAction{}, nil, fmt.Errorf("o efeito em vigor %q não dá ação nenhuma", e.Effect.CatalogID)
	}
	if option < 0 || option >= len(actions.Options) {
		return book.SpellAction{}, nil, fmt.Errorf("%s tem %d ação(ões), e o pedido foi a de índice %d",
			e.Spell.Spell.Name, len(actions.Options), option)
	}
	if left, counted := e.Charges(); counted && left <= 0 {
		return book.SpellAction{}, nil, fmt.Errorf("%s já foi descarregada", e.Spell.Spell.Name)
	}
	return e.Spell.ActionRoll(e.Effect.CastAugments, actions.Options[option].Name)
}

// Charges são as cargas que a magia em vigor ainda tem, e se ela as conta: só a conjuração com o aprimoramento que
// as cria (os dardos da Lança Ígnea, p196).
func (e SpellEffect) Charges() (left int, counted bool) {
	if e.Spell.Charges(e.Effect.CastAugments) == 0 {
		return 0, false
	}
	return e.Effect.Charges, true
}
