package ui

import (
	"strings"
	"testing"
)

// Os unitários da gramática da cena de seleção.
//
// Eles vieram de `web/characters` na ALE-297, junto com o código que medem: as
// duas perguntas são sobre ÍNDICE e sobre a FORMA da expressão, e nenhuma das
// duas precisa de banco, de servidor ou de uma cena em particular. O que exige
// a página montada — que a cena de fato ESCREVE estes gestos — continua no
// `api`, onde há um roteador de verdade.

// TestNoNeighborIsInventedOutsideTheRail (ALE-278).
//
// Ele nasceu de um caso do hospedeiro que afirmava DUAS coisas: que o
// `NeighborAt` devolve nulo fora do trilho, e que o HTML do herói único não
// desenha um "Próximo". A segunda é o que a pessoa vê e ficou lá; a primeira é
// regra interna e mora aqui, onde não precisa de banco nem de servidor para uma
// pergunta que é sobre índice.
func TestNoNeighborIsInventedOutsideTheRail(t *testing.T) {
	um := []Neighbor{{ID: 1, Name: "Thalen"}}
	if NeighborAt(um, -1) != nil {
		t.Error("índice negativo inventou um vizinho antes do primeiro")
	}
	if NeighborAt(um, 1) != nil {
		t.Error("índice além do fim inventou um vizinho depois do último")
	}
	// O CONTROLE: dentro do trilho ele ACHA. Sem isto, um `NeighborAt` que
	// devolvesse nulo sempre passaria nas duas asserções acima.
	if vz := NeighborAt(um, 0); vz == nil || vz.Name != "Thalen" {
		t.Fatalf("o índice válido não achou o herói: %+v", vz)
	}
}

// O GESTO É IDEMPOTENTE, e este guarda nasceu de um defeito medido no navegador.
//
// Um clique num quadro do filme dispara `focusin` E `click`, os dois com o mesmo
// gesto. Sem a guarda `if ($last_index != N)`, a primeira passagem calcula o sentido
// certo e escreve o índice; a SEGUNDA recalcula com o índice já atualizado —
// `N >= N` é sempre verdade — e o palco entra "adiante" mesmo andando para trás.
//
// Eu não vi isso na primeira medição porque cliquei por `element.click()`, que
// NÃO move o foco: só o gesto de verdade dispara os dois eventos. É a mesma
// família do evento sintético que o guia do pacote registra — a sonda que não
// reproduz o gesto mede outra coisa.
func TestTheCursorGestureDoesNotRecomputeTheDirectionTwice(t *testing.T) {
	// A guarda tem de estar na expressão, e ela é o que torna a segunda passagem
	// um nada. Sem `if`, rodar duas vezes é o defeito.
	gesto := CursorGesture(3, 16)

	if !strings.HasPrefix(gesto, "if ($last_index != 3)") {
		t.Fatalf("o gesto não é idempotente: %q — o focusin e o click seguidos apagariam o sentido", gesto)
	}
	// E a ordem importa: o sentido é calculado ANTES de o índice ser escrito,
	// senão ele compara o índice novo consigo mesmo.
	sentido := strings.Index(gesto, "$direction =")
	escrita := strings.Index(gesto, "$last_index = ")
	if sentido < 0 || escrita < 0 || sentido > escrita {
		t.Errorf("o índice é escrito antes de o sentido ser calculado: %q", gesto)
	}
}

// Os sinais que a cena declara e os que o gesto ESCREVE são o mesmo conjunto, e
// eles são declarados num lugar e escritos noutro.
//
// Uma cena que declarasse só `cursor` não daria erro nenhum: o gesto escreveria
// em `$direction` e `$last_index` recém-inventados, e o primeiro passo do cursor
// entraria pelo lado errado — `undefined >= undefined` é `false`, então o palco
// escolheria "atrás" e ninguém leria isso como defeito.
func TestTheStageSignalsCoverEveryOneTheGestureWrites(t *testing.T) {
	declarados := StageSignals(41)
	for _, sinal := range []string{"cursor", "direction", "last_index"} {
		if !strings.Contains(declarados, sinal+":") {
			t.Errorf("a cena não declara $%s, que o gesto escreve: %q", sinal, declarados)
		}
	}
	if !strings.Contains(declarados, "cursor: 41") {
		t.Errorf("o cursor não nasce onde a cena mandou: %q", declarados)
	}
}
