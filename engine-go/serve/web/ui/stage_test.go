package ui

import (
	"strings"
	"testing"
)

// Os unitários da gramática da cena de seleção.
//
// UNITÁRIO porque as duas perguntas são sobre ÍNDICE e sobre a FORMA da
// expressão, e nenhuma precisa de banco, de servidor ou de uma cena em
// particular. Que a cena de fato ESCREVA estes gestos continua no `api`, onde há
// um roteador de verdade.
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
// Uma sonda por `element.click()` NÃO vê isso: ela não move o foco, e só o gesto
// de verdade dispara os dois eventos. Sonda que não reproduz o gesto mede outra
// coisa.
func TestTheCursorGestureDoesNotRecomputeTheDirectionTwice(t *testing.T) {
	// A guarda tem de estar na expressão, e ela é o que torna a segunda passagem
	// um nada. Sem `if`, rodar duas vezes é o defeito.
	gesture := CursorGesture(3, 16)

	if !strings.HasPrefix(gesture, "if ($last_index != 3)") {
		t.Fatalf("o gesto não é idempotente: %q — o focusin e o click seguidos apagariam o sentido", gesture)
	}
	// E a ordem importa: o sentido é calculado ANTES de o índice ser escrito,
	// senão ele compara o índice novo consigo mesmo.
	direction := strings.Index(gesture, "$direction =")
	write := strings.Index(gesture, "$last_index = ")
	if direction < 0 || write < 0 || direction > write {
		t.Errorf("o índice é escrito antes de o sentido ser calculado: %q", gesture)
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
	declared := StageSignals(41)
	for _, signal := range []string{"cursor", "direction", "last_index"} {
		if !strings.Contains(declared, signal+":") {
			t.Errorf("a cena não declara $%s, que o gesto escreve: %q", signal, declared)
		}
	}
	if !strings.Contains(declared, "cursor: 41") {
		t.Errorf("o cursor não nasce onde a cena mandou: %q", declared)
	}
}
