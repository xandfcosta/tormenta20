package book

import "testing"

// A PÁGINA DA ATIVAÇÃO TEM UM LUGAR SÓ: a do dono — o poder de classe, o geral, o item, o poder do deus, a
// habilidade de raça —, achado pela mesma regra que diz qual poder é dono da ativação (`activation_owner.go`).
// Repetida no registro, ela divergiu em 166 de 481 entradas, e a aba Poderes mostrava a do registro. A ativação
// escreve página só quando nenhum dono tem: o benefício de origem.
func TestNoActivationRepeatsItsOwnersPage(t *testing.T) {
	measured := 0
	for _, a := range transcribedActivations() {
		measured++
		page := ownerPage(a)
		switch {
		case page > 0 && a.BookPage != 0:
			t.Errorf("a ativação %s escreve p%d, e o dono dela já diz p%d: apague a página da ativação", a.ID, a.BookPage, page)
		case page == 0 && a.BookPage == 0:
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
