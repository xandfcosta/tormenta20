package book

import (
	"fmt"
	"strings"
)

// spellSaveNames são os testes de resistência das magias (p172).
var spellSaveNames = map[string]string{"fortitude": "Fortitude", "reflexos": "Reflexos", "vontade": "Vontade"}

// SpellResistanceWritten diz o que se testa para resistir, como o livro escreve, ou "" na magia
// sem resistência. O efeito sozinho não serve: "metade" não diz quem rola o quê.
//
//	SpellResistanceWritten(relampago) // "Reflexos reduz à metade"
func SpellResistanceWritten(m Spell) string {
	save, known := spellSaveNames[m.SaveType]
	if !known || m.Resistance == "" {
		return ""
	}
	if m.Resistance == "metade" {
		return save + " reduz à metade"
	}
	return save + " " + m.Resistance
}

// PowerRequirements é o pré-requisito do poder por extenso, como o livro escreve, ou "" sem
// nenhum. O id sem verbete sai como está: um nome estranho na tela é melhor que uma regra sumida.
//
//	PowerRequirements("acuidade-com-arma") // "Des 1"
func PowerRequirements(id string) string {
	PowerCatalogs()
	var requirements []PowerPrerequisite
	if power, found := poderesDeClassePorID[id]; found {
		requirements = power.Prerequisites
	} else if general, found := poderesGeraisPorID[id]; found {
		requirements = general.Prerequisites
	}
	written := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		if text := requirement.written(); text != "" {
			written = append(written, text)
		}
	}
	return strings.Join(written, ", ")
}

func (p PowerPrerequisite) written() string {
	switch p.Kind {
	case "attribute":
		return fmt.Sprintf("%s %d", AttributeAbbrev(p.Attr), p.Min)
	case "trained":
		return "treinado em " + p.Expertise
	case "power":
		return powerNameOf(p.ID)
	case "anyPower":
		names := make([]string, 0, len(p.IDs))
		for _, id := range p.IDs {
			names = append(names, powerNameOf(id))
		}
		return strings.Join(names, " ou ")
	case "classChoice":
		return p.Label
	case "note":
		return p.Description
	}
	return ""
}

func powerNameOf(id string) string {
	if power, found := poderesDeClassePorID[id]; found {
		return power.Name
	}
	if general, found := poderesGeraisPorID[id]; found {
		return general.Name
	}
	return id
}
