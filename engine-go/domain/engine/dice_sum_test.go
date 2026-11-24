package engine

import "testing"

// A SOMA DE DADOS do dado da magia: o aprimoramento que aumenta soma ao dado da parcela, por repetição (p171).
func TestDiceSumsAddUp(t *testing.T) {
	for _, c := range []struct {
		base, add string
		times     int
		want      string
	}{
		{"6d6", "2d6", 2, "10d6"},      // a Bola de Fogo com dois aumentos
		{"2d8+2", "1d8+1", 2, "4d8+4"}, // a Curar Ferimentos com dois aumentos
		{"10d8+10", "", 0, "10d8+10"},
	} {
		base, err := ParseDiceSum(c.base)
		if err != nil {
			t.Fatalf("%s: %v", c.base, err)
		}
		got := base
		if c.add != "" {
			add, err := ParseDiceSum(c.add)
			if err != nil {
				t.Fatalf("%s: %v", c.add, err)
			}
			if got, err = base.Plus(add, c.times); err != nil {
				t.Fatalf("%s + %s ×%d: %v", c.base, c.add, c.times, err)
			}
		}
		if got.String() != c.want {
			t.Errorf("%s + %s ×%d = %s; want %s", c.base, c.add, c.times, got, c.want)
		}
	}
	if _, err := mustDiceSum("4d8").Plus(mustDiceSum("1d6"), 1); err == nil {
		t.Error("somar d6 a d8 passou, e a parcela teria duas faces")
	}
	if lowest, highest := mustDiceSum("2d8+2").Range(); lowest != 4 || highest != 18 {
		t.Errorf("2d8+2 vai de %d a %d; want 4 a 18", lowest, highest)
	}
}

func mustDiceSum(notation string) DiceSum {
	sum, err := ParseDiceSum(notation)
	if err != nil {
		panic(err)
	}
	return sum
}
