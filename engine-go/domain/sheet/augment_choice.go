package sheet

import (
	"slices"

	"t20engine/domain/book"
)

// A ESCOLHA DOS APRIMORAMENTOS vista pelo conjurador: o que cada um apaga ao ser ligado. A folha monta os
// gestos por aqui, e o `AugmentPrice` recusa pela mesma regra — um aprimoramento novo entra num lugar só.

// AugmentKey identifica um aprimoramento na folha da magia: o da própria magia pelo índice, o poder que é
// aprimoramento (p131) pelo id. Monte-a com `SpellAugmentKey` ou `PowerAugmentKey`.
type AugmentKey struct {
	Index   int
	PowerID string
}

// SpellAugmentKey é a chave do aprimoramento `index` da magia.
func SpellAugmentKey(index int) AugmentKey { return AugmentKey{Index: index} }

// PowerAugmentKey é a chave do poder que é aprimoramento.
func PowerAugmentKey(powerID string) AugmentKey { return AugmentKey{Index: -1, PowerID: powerID} }

// Clears são os aprimoramentos que ligar `key` apaga, na ordem da folha (os da magia, depois os poderes), sem
// ele mesmo (p171):
//
//   - o EXCLUSIVO apaga todos os outros, e os outros apagam o exclusivo — "truques não podem ser usados em
//     conjunto com outros aprimoramentos";
//   - a Magia Acelerada e o aprimoramento que muda a execução se apagam entre si — "mudanças na mesma
//     característica da magia nunca se acumulam".
//
// Sem isto a folha deixa montar uma combinação que o `AugmentPrice` recusa, e mostra um custo que não vale.
//
//	sheet.CasterOf(dto).Spell("explosao-de-chamas").Clears(sheet.SpellAugmentKey(0)) // o truque apaga todos
func (s SpellOnTheSheet) Clears(key AugmentKey) []AugmentKey {
	var all, exclusive, execution []AugmentKey
	for i, a := range s.Spell.Augments {
		all = append(all, SpellAugmentKey(i))
		if a.Exclusive {
			exclusive = append(exclusive, SpellAugmentKey(i))
		}
		if a.Execution != "" {
			execution = append(execution, SpellAugmentKey(i))
		}
	}
	var quickened []AugmentKey
	for _, p := range s.PowerAugments {
		all = append(all, PowerAugmentKey(p.Power.PowerID))
		if p.Power.PowerID == book.QuickenedSpellPower {
			quickened = append(quickened, PowerAugmentKey(p.Power.PowerID))
		}
	}
	clears := exclusive
	switch {
	case slices.Contains(exclusive, key):
		clears = all
	case key == PowerAugmentKey(book.QuickenedSpellPower):
		clears = slices.Concat(exclusive, execution)
	case slices.Contains(execution, key):
		clears = slices.Concat(exclusive, quickened)
	}
	return slices.DeleteFunc(slices.Clone(clears), func(k AugmentKey) bool { return k == key })
}
