package engine

import "fmt"

// RollEachDieWith rola a notação ("6d6") e devolve cada dado, na ordem: a mesa lê "6d6 (3, 5, 1, 6, 2, 4)" e
// não só a soma.
//
//	RollEachDieWith("2d6", rollDie) // [4 1]
func RollEachDieWith(notation string, rollDie func(faces int) (int, error)) ([]int, error) {
	amount, faces, err := parseDiceNotation(notation)
	if err != nil {
		return nil, err
	}
	out := make([]int, 0, amount)
	for range amount {
		value, err := rollDie(faces)
		if err != nil {
			return nil, fmt.Errorf("rolar %s: %w", notation, err)
		}
		out = append(out, value)
	}
	return out, nil
}

// DiceRange é o menor e o maior total que a notação tira: 6d6 vai de 6 a 36. É o que confere o total que a
// mesa rolou na mão.
//
//	DiceRange("6d6") // 6, 36
func DiceRange(notation string) (lowest, highest int, err error) {
	amount, faces, err := parseDiceNotation(notation)
	if err != nil {
		return 0, 0, err
	}
	return amount, amount * faces, nil
}
