package book

import (
	"strings"
	"testing"
)

// Os nove Julgamentos Divinos estão na mesma página do livro (p83), e dois citavam outra.
func TestEveryDivineJudgmentCitesItsPage(t *testing.T) {
	seen := 0
	for _, power := range ClassPowers() {
		if !strings.HasPrefix(power.Name, "Julgamento Divino: ") {
			continue
		}
		seen++
		if power.BookPage != 83 {
			t.Errorf("%s cita a p%d, e o livro diz p83", power.Name, power.BookPage)
		}
	}
	if seen != 9 {
		t.Errorf("o livro tem nove Julgamentos Divinos, e o catálogo tem %d", seen)
	}
}
