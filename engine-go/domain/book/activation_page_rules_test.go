package book

import "testing"

// A PÁGINA DA ATIVAÇÃO TEM UM LUGAR SÓ (ALE-495): a do dono — o poder de
// classe, o geral, o item, o poder do deus. Repetida no registro, ela divergiu
// em 166 de 481 entradas, e a aba Poderes mostrava a do registro. A ativação
// escreve página só quando o dono não tem — a variante sem verbete próprio
// (os Julgamentos Divinos, p83), a habilidade de raça, o benefício de origem.
func TestNoActivationRepeatsItsOwnersPage(t *testing.T) {
	measured := 0
	for _, a := range transcribedActivations() {
		measured++
		page, owned := ownerPageOf(a)
		switch {
		case owned && a.BookPage != 0:
			t.Errorf("a ativação %s escreve p%d, e o dono dela já diz p%d: apague a página da ativação", a.ID, a.BookPage, page)
		case !owned && a.BookPage == 0:
			t.Errorf("a ativação %s não tem dono com página e não escreve a sua: confira no livro e escreva-a", a.ID)
		}
	}
	if measured < 400 {
		t.Fatalf("mediu %d ativações; o catálogo tem mais de 400", measured)
	}
	for _, a := range Activations() {
		if a.BookPage == 0 {
			t.Errorf("a ativação %s chegou sem página depois da herança", a.ID)
		}
	}
}
