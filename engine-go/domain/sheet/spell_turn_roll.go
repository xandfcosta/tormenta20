package sheet

import (
	"fmt"
	"strings"

	"t20engine/domain/book"
)

// TurnRoll é o dado da vez desta magia (ver GLOSSARY, **dado da vez**): as parcelas da versão pedida com os
// aprimoramentos escolhidos ao conjurar, pela mesma regra do dado da conjuração (`CastRoll`). A versão é o nome
// dela no Manto do Cruzado, e vazia na magia de versão única.
//
//	sheet.CasterOf(dto).Spell("ferver-sangue").TurnRoll([]sheet.AugmentPick{{AugmentIndex: 0, Stacks: 1}}, "") // 5d8 de fogo
func (s SpellOnTheSheet) TurnRoll(picks []AugmentPick, version string) ([]book.SpellRoll, error) {
	turn := s.Spell.TurnRoll
	if turn == nil {
		return nil, fmt.Errorf("%s não rola nada na vez de quem conjurou", s.Spell.Name)
	}
	for _, v := range turn.Versions {
		if v.Name != version {
			continue
		}
		roll, err := s.augmentedRoll(v.Rolls, picks)
		if err != nil {
			return nil, err
		}
		if roll.ByText {
			return nil, fmt.Errorf("um aprimoramento de %s mudou o dado para um que se rola pelo texto dele", s.Spell.Name)
		}
		return roll.Parts, nil
	}
	if version == "" {
		return nil, fmt.Errorf("%s tem as versões %s: diga qual", s.Spell.Name, strings.Join(s.TurnVersions(), " e "))
	}
	return nil, fmt.Errorf("%s não tem a versão %q; tem %s", s.Spell.Name, version, strings.Join(s.TurnVersions(), " e "))
}

// TurnVersions são os nomes das versões do dado da vez, na ordem do livro: as duas do Manto do Cruzado, ou uma
// só, vazia, na magia de versão única. Nil na magia sem dado da vez.
func (s SpellOnTheSheet) TurnVersions() []string {
	if s.Spell.TurnRoll == nil {
		return nil
	}
	names := make([]string, 0, len(s.Spell.TurnRoll.Versions))
	for _, v := range s.Spell.TurnRoll.Versions {
		names = append(names, v.Name)
	}
	return names
}
