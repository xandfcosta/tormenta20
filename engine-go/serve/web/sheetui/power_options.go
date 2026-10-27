package sheetui

import (
	"t20engine/domain/book"
	"t20engine/domain/sheet"
)

// AS OPÇÕES DE CADA ESCOLHA DE PODER: o que o diálogo oferece, dado o que a
// ficha já tem. Mora à parte da montagem do diálogo porque muda por outra
// razão — cada poder novo com escolha traz a sua regra de oferta (ALE-510).

// optionsOf monta as opções do poder com a contagem do que já foi escolhido.
//
// A escolha de perícia TREINADA oferece só as treinadas da ficha: o Foco em
// Perícia pede o treino como pré-requisito (p130), e oferecer a sem treino é
// desenhar uma recusa. As que DÃO o que falta — o Treinamento em Perícia e a
// Proficiência — oferecem só o que a ficha ainda não tem, mais o que elas
// mesmas deram, para poder ser tirado. `owned` é o que a ficha já tem: as
// perícias treinadas e as proficiências (`ownedOptionsOf`).
func optionsOf(p book.ClassPower, picks []string, owned map[string]bool) []powerOption {
	if p.Choice == nil {
		return nil
	}
	offered := p.Choice.Offered()
	out := make([]powerOption, 0, len(offered))
	for _, o := range offered {
		// O Foco em Perícia pede perícia treinada e o Foco em Magia, magia do
		// grimório: as duas escolhem do que a ficha JÁ tem.
		if (p.Choice.Kind == "trained-expertise" || p.Choice.Kind == "spell") && !owned[o.ID] {
			continue
		}
		count := 0
		for _, pick := range picks {
			if pick == o.ID {
				count++
			}
		}
		gives := p.Choice.Kind == "untrained-expertise" || p.Choice.Kind == "proficiency"
		if gives && owned[o.ID] && count == 0 {
			continue
		}
		out = append(out, powerOption{Value: o.ID, Label: o.Name, Count: count})
	}
	return out
}

// ownedOptionsOf é o que a ficha já tem entre as opções de escolha: as perícias
// treinadas, as proficiências e as magias do grimório. As três famílias não
// colidem — "Luta", "armas-marciais" e "bola-de-fogo" —, então um conjunto só
// serve a todas.
func ownedOptionsOf(dto sheet.CharacterDTO) map[string]bool {
	out := sheet.ToStringSet(sheet.UnmarshalStrings(dto.Proficiencies))
	// A magia conta como "tida" quando a ficha a LANÇA: a do grimório acima do
	// círculo alcançado fica fora do Foco em Magia (p131).
	for _, spell := range dto.Spells {
		if sheet.CanCastCircleOf(dto, spell.CatalogSpellID) {
			out[spell.CatalogSpellID] = true
		}
	}
	for _, e := range dto.Expertises {
		if e.Trained {
			out[e.Name] = true
		}
	}
	return out
}
