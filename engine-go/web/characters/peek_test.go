package characters

import "testing"

// TestNoNeighborIsInventedOutsideTheRail (ALE-278).
//
// Ele nasceu de um caso do hospedeiro que afirmava DUAS coisas: que o `peekAt`
// devolve nulo fora do trilho, e que o HTML do herói único não desenha um
// "Próximo". A segunda é o que a pessoa vê e ficou lá; a primeira é regra
// interna e mora aqui, onde não precisa de banco nem de servidor para uma
// pergunta que é sobre índice.
func TestNoNeighborIsInventedOutsideTheRail(t *testing.T) {
	um := []HeroCard{{ID: 1, Name: "Thalen"}}
	if peekAt(um, -1) != nil {
		t.Error("índice negativo inventou um vizinho antes do primeiro")
	}
	if peekAt(um, 1) != nil {
		t.Error("índice além do fim inventou um vizinho depois do último")
	}
	// O CONTROLE: dentro do trilho ele ACHA. Sem isto, um `peekAt` que devolvesse
	// nulo sempre passaria nas duas asserções acima.
	if vz := peekAt(um, 0); vz == nil || vz.Name != "Thalen" {
		t.Fatalf("o índice válido não achou o herói: %+v", vz)
	}
}
