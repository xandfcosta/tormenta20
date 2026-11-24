package sheet

import (
	"fmt"
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// OS PODERES QUE SÃO APRIMORAMENTO vistos pelo conjurador (p131): a Magia Acelerada, a Ampliada e a Discreta
// "seguem todas as regras para aprimoramentos". A magia na ficha lista os que a ficha TEM, com o cadeado
// dela, e o `AugmentPrice` os cobra junto dos aprimoramentos da magia — a folha desenha e o servidor recusa
// pela mesma resposta.

// RitualSwitch é o interruptor do Celebrar Ritual (p131): ligado, a próxima magia é ritual.
var RitualSwitch = engine.FlagGroupID("ritual")

// PowerAugmentOnTheSheet é um poder da ficha que é aprimoramento, e o cadeado dele nesta magia, vazio quando
// ele se aplica: "só em execução de movimento, padrão ou completa" na Acelerada de uma reação. O trancado
// aparece em vez de sumir, como o aprimoramento de círculo alto.
type PowerAugmentOnTheSheet struct {
	Power book.PowerAugment
	Lock  string
}

// ownedPowerAugments são os poderes que são aprimoramento e que a ficha tem, na ordem do livro.
func ownedPowerAugments(dto CharacterDTO) []book.PowerAugment {
	owned := OwnedPowers(dto)
	var out []book.PowerAugment
	for _, power := range book.PowerAugments() {
		if slices.ContainsFunc(owned, func(p OwnedPower) bool { return p.ID == power.PowerID }) {
			out = append(out, power)
		}
	}
	return out
}

// powerAugmentPrice é o PM dos poderes escolhidos como aprimoramento, ou a frase da recusa.
//
// O poder que falta na ficha é recusado aqui porque quem desenha a linha é a tela, e a fronteira é o
// servidor: um pedido montado à mão pagaria o aprimoramento de um poder que o personagem não tem.
func (s SpellOnTheSheet) powerAugmentPrice(spellPicks []AugmentPick, picks []PowerAugmentPick) (int, string) {
	total := 0
	for _, pick := range picks {
		augment, known := book.PowerAugmentOf(pick.PowerID)
		if !known {
			return 0, fmt.Sprintf("o poder %q não é aprimoramento de magia", pick.PowerID)
		}
		at := slices.IndexFunc(s.PowerAugments, func(p PowerAugmentOnTheSheet) bool { return p.Power.PowerID == pick.PowerID })
		if at < 0 {
			return 0, fmt.Sprintf("esta ficha não tem %s", augment.Name)
		}
		if pick.Stacks < 1 {
			return 0, fmt.Sprintf("%s pede um número de repetições de 1 para cima (veio %d)", augment.Name, pick.Stacks)
		}
		if augment.Augment.Kind == "muda" && pick.Stacks > 1 {
			return 0, fmt.Sprintf("%s muda a magia, e mudança não se acumula", augment.Name)
		}
		if lock := s.PowerAugments[at].Lock; lock != "" {
			return 0, fmt.Sprintf("%s nesta magia: %s", augment.Name, lock)
		}
		// "Mudanças na mesma característica da magia nunca se acumulam" (p171).
		if pick.PowerID == book.QuickenedSpellPower && s.ritual {
			return 0, fmt.Sprintf("o ritual e %s mudam a execução, e mudança não se acumula", augment.Name)
		}
		for _, spellPick := range spellPicks {
			own := s.Spell.Augments[spellPick.AugmentIndex]
			if own.Exclusive {
				return 0, fmt.Sprintf(
					"o aprimoramento %d não pode ser usado em conjunto com outros: escolha só ele", spellPick.AugmentIndex)
			}
			// "Mudanças na mesma característica da magia nunca se acumulam" (p171).
			if pick.PowerID == book.QuickenedSpellPower && book.AugmentChangesExecution(own.Description) {
				return 0, fmt.Sprintf("%s e o aprimoramento %d mudam a execução, e mudança não se acumula",
					augment.Name, spellPick.AugmentIndex)
			}
		}
		total += augment.Augment.PmCost * pick.Stacks
	}
	return total, ""
}
