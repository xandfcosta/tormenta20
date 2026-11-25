package character

import (
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// RollTurn rola o dado da vez da magia em vigor na linha `effectID` da ficha (ver GLOSSARY, **dado da vez**), com
// os aprimoramentos escolhidos ao conjurá-la, ou confere o total que a mesa rolou na mão. Não cobra nada: o PM
// foi pago na conjuração, e a manutenção da sustentada é cobrada na entrada da vez. Quem diz se é a hora é a
// mesa.
//
// A versão é a que quem joga apertou, no Manto do Cruzado; vazia na magia de versão única.
func (p Plays) RollTurn(dto sheet.CharacterDTO, effectID int64, version string, fromTheTable []int) (SpellCast, error) {
	spellEffect, err := sheet.CasterOf(dto).SpellEffect(effectID)
	if err != nil {
		return SpellCast{}, err
	}
	parts, err := spellEffect.TurnRoll(version)
	if err != nil {
		return SpellCast{}, err
	}
	cast, err := spellCastOf(dto, spellEffect.Spell)
	if err != nil {
		return SpellCast{}, err
	}
	cast.When = book.TurnRollAt(spellEffect.Spell.Spell.TurnRoll.At)
	if version != "" {
		cast.Name += " (" + version + ")"
	}
	if cast.Parts, err = p.rollTheParts(cast.Name, parts, fromTheTable); err != nil {
		return SpellCast{}, err
	}
	return cast, nil
}
