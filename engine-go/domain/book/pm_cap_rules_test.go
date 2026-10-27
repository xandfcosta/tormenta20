package book

import "testing"

// O TETO DE PM DO CUSTO VARIÁVEL (p224, ALE-506): "o máximo de PM que você pode
// gastar por uso é igual ao seu nível na classe que fornece a habilidade (mas
// você sempre pode usar a habilidade em seu custo mínimo)". O verbete pode
// apertar mais: Gritar Ordens é "limitado pelo seu Carisma" (p80); o Ladrão
// Arcano paga 1 PM por círculo, até o 4º (p74).
func TestTheVariableCostIsCappedByLevelAndByTheEntry(t *testing.T) {
	orders := ActivationScale{BasePm: 1, StepPm: 1, CapAttribute: "charisma"}
	thief := ActivationScale{BasePm: 1, StepPm: 1, CapPm: 4}
	for _, c := range []struct {
		name  string
		scale ActivationScale
		level int
		attrs map[string]int
		want  int
	}{
		{"Gritar Ordens, nobre 4, Car 3: até 3 PM", orders, 4, map[string]int{"charisma": 3}, 2},
		{"Gritar Ordens, nobre 4, Car 5: o nível manda, 4 PM", orders, 4, map[string]int{"charisma": 5}, 3},
		{"Gritar Ordens, Car 0: o custo mínimo sempre vale", orders, 4, map[string]int{"charisma": 0}, 0},
		{"Ladrão Arcano, ladino 10: até o 4º círculo", thief, 10, nil, 3},
		{"Ladrão Arcano, ladino 2: o nível manda", thief, 2, nil, 1},
	} {
		if got := MaxSteps(c.scale, c.level, c.attrs); got != c.want {
			t.Errorf("%s: %d degraus, esperava %d", c.name, got, c.want)
		}
	}
	// O DEGRAU POR NÍVEL continua como era (a Inspiração, p44).
	inspiration := ActivationScale{BasePm: 2, StepPm: 2, FirstStepLevel: 5, StepEveryLevels: 4}
	if got := MaxSteps(inspiration, 9, nil); got != 2 {
		t.Errorf("Inspiração no nível 9: %d degraus, esperava 2", got)
	}
}
