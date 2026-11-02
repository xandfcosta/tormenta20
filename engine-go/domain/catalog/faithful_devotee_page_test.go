package catalog

import (
	"testing"

	"t20engine/domain/catalog/transcript"
)

// A exceção: o Devoto Fiel do clérigo apontava a página do druida. No livro, ele está na p57.
func TestTheClericFaithfulDevoteePointsAtItsPage(t *testing.T) {
	for _, power := range transcript.ClassPowers {
		if power.ID == "class.clerigo.devoto-fiel" {
			if power.BookPage != 57 {
				t.Errorf("o Devoto Fiel do clérigo aponta a p%d, e está na p57", power.BookPage)
			}
			return
		}
	}
	t.Fatal("o catálogo não tem o Devoto Fiel do clérigo")
}
