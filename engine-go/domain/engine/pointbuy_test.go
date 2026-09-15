package engine

import (
	"reflect"
	"testing"
)

func spread(over map[string]int) map[string]int {
	m := map[string]int{"strength": 0, "dexterity": 0, "constitution": 0, "intelligence": 0, "wisdom": 0, "charisma": 0}
	for k, v := range over {
		m[k] = v
	}
	return m
}

func TestPointBuySpent(t *testing.T) {
	// Book example: 3(4)+2(2)+2(2)+1(1)+1(1) = 10, legal.
	got, ok := PointBuySpent(spread(map[string]int{"strength": 3, "dexterity": 2, "constitution": 2, "intelligence": 1, "wisdom": 1}))
	if !ok || got != PointBuyBudget {
		t.Fatalf("book spread: got (%d, %v), want (10, true)", got, ok)
	}
	// −1 refunds a point.
	if s, ok := PointBuySpent(spread(map[string]int{"charisma": -1, "strength": 4, "dexterity": 2, "constitution": 2})); !ok || s != 10 {
		t.Fatalf("refund spread: got (%d, %v), want (10, true)", s, ok)
	}
	// Out of range → ok=false.
	if _, ok := PointBuySpent(spread(map[string]int{"strength": 5})); ok {
		t.Fatalf("out-of-range spread: ok=true, want false")
	}
}

func TestPointBuyWarnings(t *testing.T) {
	if w := PointBuyWarnings(spread(map[string]int{"strength": 3, "dexterity": 2, "constitution": 2, "intelligence": 1, "wisdom": 1})); len(w) != 0 {
		t.Fatalf("legal spread warnings: %v, want none", w)
	}
	// Exact pt-BR strings (byte-parity with point-buy.ts — note U+2212 in the −1 message).
	overW := PointBuyWarnings(spread(map[string]int{"strength": 4, "dexterity": 4}))
	if !reflect.DeepEqual(overW, []string{"compra de pontos: 14 pontos gastos excedem o limite de 10"}) {
		t.Fatalf("over-budget warnings: %q", overW)
	}
	reducedW := PointBuyWarnings(spread(map[string]int{"charisma": -1, "wisdom": -1}))
	if !reflect.DeepEqual(reducedW, []string{"compra de pontos: apenas UM atributo pode ser reduzido a −1 (p17), há 2"}) {
		t.Fatalf("two-reduced warnings: %q", reducedW)
	}
	rangeW := PointBuyWarnings(spread(map[string]int{"strength": 5}))
	if !reflect.DeepEqual(rangeW, []string{"compra de pontos: strength = 5 fora do intervalo [-1, 4]"}) {
		t.Fatalf("out-of-range warnings: %q", rangeW)
	}
}
