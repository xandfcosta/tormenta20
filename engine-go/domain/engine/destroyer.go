package engine

// O DESTRUIDOR (bárbaro p41, guerreiro p65): "Quando causa dano com uma arma corpo a corpo de duas mãos, você
// pode rolar novamente qualquer resultado 1 ou 2 das rolagens de dano da arma."
//
// Rola de novo TODOS os 1 e 2 de uma vez (decisão do dono): rolar de novo um 1 ou um 2 nunca piora a média, e
// ninguém escolheria guardar um. Vale o segundo resultado, mesmo que pior (p226), e o dado não se rola outra
// vez. Só os dados DA ARMA: os dados a mais (o Golpe Divino, o encanto) ficam como saíram.

// rerollsLowDiceOf diz se a carta rola de novo os 1 e 2: o Destruidor escreve o escopo da arma corpo a corpo de
// duas mãos, e a carta o lê pelos escopos dela.
func rerollsLowDiceOf(effects ItemEffects, scopes []string) bool {
	for _, scope := range scopes {
		if StatFor(effects, ModifierTarget{K: "rerollsLowWeaponDice", Scope: scope}).Total > 0 {
			return true
		}
	}
	return false
}

// isLowDie é o dado que o Destruidor rola de novo.
func isLowDie(value int) bool { return value == 1 || value == 2 }

// LowWeaponDice conta quantos dados da arma saíram 1 ou 2 — quantos o Destruidor rola de novo.
func LowWeaponDice(dice []int) int {
	count := 0
	for _, value := range dice {
		if isLowDie(value) {
			count++
		}
	}
	return count
}

// RerollLowWeaponDice devolve os dados da arma com cada 1 e 2 rolado de novo por `roll`, na mesma ordem. O
// `roll` é o sorteio do servidor ou os dados que a mesa digitou (`DiceRolledByHand`).
//
//	after, err := RerollLowWeaponDice([]int{1, 5, 2}, 8, rolar) // [?, 5, ?]
func RerollLowWeaponDice(dice []int, faces int, roll func(int) (int, error)) ([]int, error) {
	after := make([]int, len(dice))
	for i, value := range dice {
		after[i] = value
		if !isLowDie(value) {
			continue
		}
		again, err := roll(faces)
		if err != nil {
			return nil, err
		}
		after[i] = again
	}
	return after, nil
}
