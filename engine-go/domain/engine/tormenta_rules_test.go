package engine

import (
	"testing"
)

// Poderes da Tormenta, p136: "Quando escolhe um poder da Tormenta, você perde 1 de
// Carisma. Para cada dois outros poderes da Tormenta, você perde mais 1 de Carisma."
//
// Afirmada onde o MOTOR decide, que é onde a produção a executa.

func TestCarismaLossFromPowers(t *testing.T) {
	// Um e dois poderes: a frase do livro não deixa margem. Com um poder, perde-se 1;
	// com dois, cada um tem apenas UM outro poder, e "para cada DOIS outros" ainda não
	// se paga — 1 + 1 = 2.
	for _, tc := range []struct{ powers, want int }{{0, 0}, {1, 1}, {2, 2}} {
		if got := carismaLossFromPowers(tc.powers); got != tc.want {
			t.Errorf("%d poder(es): perda=%d, livro p136 diz %d", tc.powers, got, tc.want)
		}
	}
}

// A partir do TERCEIRO poder a frase admite duas leituras, e o motor escolheu uma:
// ele congela o custo no momento da escolha (o 3º poder custa 2, total 4), em vez de
// recalcular o custo de todos com a contagem final (que daria 6). As duas leituras
// cabem em "para cada dois outros poderes você perde mais 1", e nada no livro decide.
//
// Este teste fixa a leitura ESCOLHIDA, não a verdade: se a mesa decidir a outra, ele é
// o lugar que muda — e a intenção é que quem mudar leia isto antes.
func TestCarismaLossEscalationIsSequential(t *testing.T) {
	want := []int{0, 1, 2, 4, 6, 9, 12}
	for powers, expected := range want {
		if got := carismaLossFromPowers(powers); got != expected {
			t.Errorf("%d poderes: perda=%d, queria %d (sequência 1,2,4,6,9,12)", powers, got, expected)
		}
	}
}
