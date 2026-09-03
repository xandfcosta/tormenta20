package book

import "testing"

// Os dois guardas do que a RAÇA escreve na tela (ALE-264).
//
// Eles vieram do `api` na ALE-278, e o caminho conta uma coisa: o arquivo que
// os hospedava se dividiu em TRÊS camadas nesta fatia — dois casos de regra
// pura vieram para cá, um de renderização ficou no `web/master`, e dois de
// composição por HTTP ficaram no `api`. É a quinta vez seguida que a fronteira
// separa um arquivo de teste que misturava camadas, e a primeira em que ele
// misturava três.

// TestModifiersComeOutInTheBookOrder.
//
// O defeito que ele prende não é de conteúdo, é de DETERMINISMO: os
// modificadores vêm num `map[string]int`, e a ordem de um mapa em Go é aleatória
// por projeto. Imprimir direto do mapa daria uma ordem diferente a cada render —
// a mesma página mudando sozinha entre dois pedidos iguais, e qualquer teste
// sobre o texto ficaria intermitente.
//
// Rodar cem vezes é o que torna a asserção honesta: uma passada só teria 1/6 de
// chance de pegar a ordem errada por acaso.
func TestModifiersComeOutInTheBookOrder(t *testing.T) {
	elfo := RaceAttribute{
		Kind: "fixed",
		Mods: map[string]int{"intelligence": 2, "dexterity": 1, "constitution": -1},
	}
	const esperado = "+1 Des, -1 Con, +2 Int"
	for i := 0; i < 100; i++ {
		if escrito := elfo.Escrito(); escrito != esperado {
			t.Fatalf("volta %d: %q — a ordem do livro é For, Des, Con, Int, Sab, Car", i, escrito)
		}
	}
}

// TestAFreeChoiceDoesNotBecomeThreeInventedAttributes: as duas formas do livro.

func TestAFreeChoiceDoesNotBecomeThreeInventedAttributes(t *testing.T) {
	humano := RaceAttribute{Kind: "floating", Count: 3, Value: 1}
	if escrito := humano.Escrito(); escrito != "+1 em três atributos" {
		t.Errorf("o humano escolhe onde põe os três +1, e a tela disse %q", escrito)
	}
}

// TestTheUnifiedSearchReachesTheThreeNewOnes: a busca sem aba varre os SETE.
