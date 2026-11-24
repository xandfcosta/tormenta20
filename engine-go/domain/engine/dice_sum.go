package engine

import (
	"fmt"
	"regexp"
	"strconv"
)

// DiceSum é o dado de uma parcela do dado da magia: "2d8+2" é 2 dados de 8 faces e +2 (ver GLOSSARY, **dado da
// magia**). Ele soma os aumentos do aprimoramento, rola e diz o total que a mesa pode ter rolado na mão.
type DiceSum struct {
	Count int
	Faces int
	Bonus int
}

var diceSumNotation = regexp.MustCompile(`^(\d+)d(\d+)(?:\+(\d+))?$`)

// ParseDiceSum lê "NdM" ou "NdM+K".
//
//	ParseDiceSum("2d8+2") // DiceSum{Count: 2, Faces: 8, Bonus: 2}
func ParseDiceSum(notation string) (DiceSum, error) {
	parts := diceSumNotation.FindStringSubmatch(notation)
	if parts == nil {
		return DiceSum{}, fmt.Errorf("o dado %q não tem a forma NdM ou NdM+K, como 2d8+2", notation)
	}
	sum := DiceSum{}
	sum.Count, _ = strconv.Atoi(parts[1])
	sum.Faces, _ = strconv.Atoi(parts[2])
	if parts[3] != "" {
		sum.Bonus, _ = strconv.Atoi(parts[3])
	}
	if sum.Count < 1 || sum.Count > maxDiceRolled || sum.Faces < 2 {
		return DiceSum{}, fmt.Errorf("o dado %q pede de 1 a %d dados de 2 faces ou mais", notation, maxDiceRolled)
	}
	return sum, nil
}

// Plus soma `add` `times` vezes: o aprimoramento que aumenta, por repetição (p171). Faces diferentes não se
// somam numa parcela, e a recusa diz quais.
//
//	six.Plus(two, 2) // 10d6, com six = 6d6 e two = 2d6
func (d DiceSum) Plus(add DiceSum, times int) (DiceSum, error) {
	if add.Faces != d.Faces {
		return d, fmt.Errorf("somar %s a %s junta d%d com d%d numa parcela", add, d, add.Faces, d.Faces)
	}
	d.Count += add.Count * times
	d.Bonus += add.Bonus * times
	return d, nil
}

// String é a notação: "4d8+4", ou "6d6" sem bônus.
func (d DiceSum) String() string {
	if d.Bonus == 0 {
		return fmt.Sprintf("%dd%d", d.Count, d.Faces)
	}
	return fmt.Sprintf("%dd%d+%d", d.Count, d.Faces, d.Bonus)
}

// Range é o menor e o maior total que o dado tira: 2d8+2 vai de 4 a 18. É o que confere o total rolado na mão.
func (d DiceSum) Range() (lowest, highest int) {
	return d.Count + d.Bonus, d.Count*d.Faces + d.Bonus
}

// RollWith rola cada dado com `rollDie` e devolve os dados e o total, com o bônus.
func (d DiceSum) RollWith(rollDie func(faces int) (int, error)) (rolls []int, total int, err error) {
	rolls = make([]int, 0, d.Count)
	for range d.Count {
		value, err := rollDie(d.Faces)
		if err != nil {
			return nil, 0, fmt.Errorf("rolar %s: %w", d, err)
		}
		rolls = append(rolls, value)
		total += value
	}
	return rolls, total + d.Bonus, nil
}
