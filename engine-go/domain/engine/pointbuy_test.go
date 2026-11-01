package engine

import "testing"

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
