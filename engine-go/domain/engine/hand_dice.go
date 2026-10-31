package engine

import "fmt"

// HandDice são os dados que a mesa rolou NA MÃO, entregues na ordem em que a
// regra os pede — a mesma ordem da `DamageFormula`: a arma primeiro, os extras
// depois. Ele toma o lugar do sorteio, e por isso a regra não distingue o dado
// da mesa do dado do servidor.
type HandDice struct {
	values []int
	next   int
}

// DiceRolledByHand embrulha os valores digitados.
//
//	hand := DiceRolledByHand([]int{8, 2, 5, 6})
//	out, err := RollAttackDamage(carta, alvo, nil, D20(20), RulingCritical, hand.Roll)
func DiceRolledByHand(values []int) *HandDice { return &HandDice{values: values} }

// Roll entrega o próximo dado e confere que ele cabe nas faces pedidas: um d8
// digitado como 9 não é um dado, é um número errado.
func (h *HandDice) Roll(faces int) (int, error) {
	if h.next >= len(h.values) {
		return 0, fmt.Errorf("faltou o %dº dado (um d%d): foram digitados %d", h.next+1, faces, len(h.values))
	}
	value := h.values[h.next]
	h.next++
	if value < 1 || value > faces {
		return 0, fmt.Errorf("o %dº dado foi %d, e um d%d vai de 1 a %d", h.next, value, faces, faces)
	}
	return value, nil
}

// Unused diz quantos dados digitados sobraram: mais dados que a fórmula pede é
// sinal de que a mão leu outra fórmula.
func (h *HandDice) Unused() int { return len(h.values) - h.next }
