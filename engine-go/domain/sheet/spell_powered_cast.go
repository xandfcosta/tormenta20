package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
)

// OS PODERES QUE SÃO APRIMORAMENTO E MEXEM NO DADO DA CONJURAÇÃO (ver GLOSSARY, **dado da magia**). O preço e a
// recusa deles são do `AugmentPrice`; aqui mora o que eles fazem com o dado, depois de pagos.

// PoweredCast é o que os poderes escolhidos fazem com o dado da conjuração.
type PoweredCast struct {
	// Maximize é o Espírito dos Solstícios: cada parcela no máximo, sem rolar (p62).
	Maximize bool
	// DamageKind é o tipo que a Magia Sagrada/Profana põe no dano, "luz" ou "trevas"; vazio sem ela (p58).
	DamageKind string
	// VitalShare é a Comunhão Vital: outra criatura recupera metade da cura (p58).
	VitalShare bool
}

// PoweredCastOf lê os poderes escolhidos, com a energia que o deus de quem conjura canaliza ("positiva", "negativa"
// ou "qualquer"): o deus de energia qualquer pede a escolha de quem conjura, e ela vem no `Energy` da escolha.
//
//	powered, err := sheet.PoweredCastOf(picks, book.GodEnergy(god))
func PoweredCastOf(picks []PowerAugmentPick, godEnergy string) (PoweredCast, error) {
	var powered PoweredCast
	for _, pick := range picks {
		if pick.Stacks < 1 {
			continue
		}
		switch pick.PowerID {
		case book.SolsticeSpiritPower:
			powered.Maximize = true
		case book.VitalCommunionPower:
			powered.VitalShare = true
		case book.HolySpellPower:
			kind, err := holyKindOf(godEnergy, pick.Energy)
			if err != nil {
				return PoweredCast{}, err
			}
			powered.DamageKind = kind
		}
	}
	return powered, nil
}

// holyKindOf é o tipo que a Magia Sagrada/Profana põe no dano: "luz (se sua divindade canaliza energia positiva)
// ou trevas (se canaliza energia negativa)" (p58). A energia qualquer é escolha de quem conjura.
func holyKindOf(godEnergy, chosen string) (string, error) {
	switch godEnergy {
	case "positiva":
		return "luz", nil
	case "negativa":
		return "trevas", nil
	case "qualquer":
		if chosen == "luz" || chosen == "trevas" {
			return chosen, nil
		}
		return "", fmt.Errorf("a Magia Sagrada/Profana de quem segue deus de energia qualquer pede a escolha: luz ou trevas (p58)")
	}
	return "", fmt.Errorf("a Magia Sagrada/Profana segue a energia do deus, e a ficha não tem deus que a canalize (p58)")
}

// Apply troca o tipo das parcelas de DANO pela Magia Sagrada/Profana; a cura e os PV temporários ficam. O máximo e a divisão da cura são
// de quem rola.
func (p PoweredCast) Apply(parts []book.SpellRoll) []book.SpellRoll {
	if p.DamageKind == "" {
		return parts
	}
	out := slices.Clone(parts)
	for i := range out {
		if out[i].IsDamage() {
			out[i].Kind = p.DamageKind
		}
	}
	return out
}
