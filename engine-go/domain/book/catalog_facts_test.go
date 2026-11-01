package book

import "testing"

// Todo verbete dos catálogos aponta a página do livro: sem ela, o cartão do mestre não tem o que
// abrir, e a falta não aparece em lugar nenhum.
func TestEveryCatalogEntryHasABookPage(t *testing.T) {
	measured := 0
	for _, p := range FlattenedPowers() {
		measured++
		if p.BookPage == 0 {
			t.Errorf("o poder %s (%s) não tem página", p.ID, p.Source)
		}
	}
	for _, i := range Catalogs().Items {
		measured++
		if i.BookPage == 0 {
			t.Errorf("o item %s não tem página", i.ID)
		}
	}
	if measured < 500 {
		t.Fatalf("só %d verbetes medidos: o catálogo não carregou", measured)
	}
}
