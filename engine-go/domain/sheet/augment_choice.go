package sheet

import (
	"slices"

	"t20engine/domain/book"
	"t20engine/domain/engine"
)

// A ESCOLHA DOS APRIMORAMENTOS vista pelo conjurador: o que cada um apaga ao ser ligado, e a execução que a
// conjuração cobra do turno com as escolhas feitas. A folha monta os gestos por aqui e a cena cobra o turno por
// aqui, e o `AugmentPrice` recusa pela mesma regra — um aprimoramento novo entra num lugar só.

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
//   - o poder que muda a execução (a Magia Acelerada, a Prece de Combate) apaga o aprimoramento da magia e o
//     outro poder que a mudam, e é apagado por eles — "mudanças na mesma característica da magia nunca se
//     acumulam". Dois aprimoramentos da própria magia que a mudam ainda não se apagam entre si, e o
//     `AugmentPrice` também não os recusa — a mesma lacuna nas duas pontas, de propósito: a folha não tranca
//     o que o servidor aceita.
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
	var powerExecution []AugmentKey
	for _, p := range s.PowerAugments {
		all = append(all, PowerAugmentKey(p.Power.PowerID))
		if p.Power.Augment.Execution != "" {
			powerExecution = append(powerExecution, PowerAugmentKey(p.Power.PowerID))
		}
	}
	clears := exclusive
	switch {
	case slices.Contains(exclusive, key):
		clears = all
	case slices.Contains(powerExecution, key):
		clears = slices.Concat(exclusive, execution, powerExecution)
	case slices.Contains(execution, key):
		clears = slices.Concat(exclusive, powerExecution)
	}
	return slices.DeleteFunc(slices.Clone(clears), func(k AugmentKey) bool { return k == key })
}

// Execution é a execução que a conjuração cobra do turno com estas escolhas: a longa do ritual (p131), a do
// poder escolhido que a muda (a livre da Magia Acelerada, p131; a de movimento da Prece de Combate, p58), a
// que o aprimoramento da magia dá (`book.SpellAugment.Execution`), ou a da magia. Quem cobra é a cena, ANTES de conjurar; a combinação que o livro recusa (o ritual com a Acelerada,
// duas mudanças de execução) é recusada pelo `AugmentPrice` logo depois, sem gastar nada.
//
//	sheet.CasterOf(dto).Spell("armamento-da-natureza").Execution(movimento, nil) // engine.ActionMovement
func (s SpellOnTheSheet) Execution(picks []AugmentPick, powerPicks []PowerAugmentPick) engine.ActionCost {
	if s.ritual {
		return engine.ActionLong
	}
	for _, p := range powerPicks {
		if power, known := book.PowerAugmentOf(p.PowerID); known && power.Augment.Execution != "" {
			return engine.ActionCost(power.Augment.Execution)
		}
	}
	for _, p := range picks {
		if p.AugmentIndex < 0 || p.AugmentIndex >= len(s.Spell.Augments) {
			continue
		}
		if gives := s.Spell.Augments[p.AugmentIndex].Execution; gives != "" {
			return engine.ActionCost(gives)
		}
	}
	return engine.ActionCost(s.Spell.Execution)
}
