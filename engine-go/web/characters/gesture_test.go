package characters

import (
	"strings"
	"testing"
)

// O gesto que move o cursor é REGRA da cena, e por isso ele fica no pacote
// enquanto os irmãos dele — que precisam de um servidor montado para pedir a
// página — ficaram no `api` (ALE-278). Sexta vez que a fronteira separa um
// arquivo de teste que misturava duas camadas.

// O GESTO É IDEMPOTENTE, e este guarda nasceu de um defeito medido no navegador.
//
// Um clique num quadro do filme dispara `focusin` E `click`, os dois com o mesmo
// gesto. Sem a guarda `if ($indice != N)`, a primeira passagem calcula o sentido
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
	gesto := theCursorGesture(3, 16)

	if !strings.HasPrefix(gesto, "if ($indice != 3)") {
		t.Fatalf("o gesto não é idempotente: %q — o focusin e o click seguidos apagariam o sentido", gesto)
	}
	// E a ordem importa: o sentido é calculado ANTES de o índice ser escrito,
	// senão ele compara o índice novo consigo mesmo.
	sentido := strings.Index(gesto, "$sentido =")
	escrita := strings.Index(gesto, "$indice = ")
	if sentido < 0 || escrita < 0 || sentido > escrita {
		t.Errorf("o índice é escrito antes de o sentido ser calculado: %q", gesto)
	}
}

// AS DUAS PARTES QUE SE MOVEM existem no HTML, e são as que o CSS anima.
//
// A classe do palco não anima nada sozinha: quem tem `animation` são os
// descendentes `.palco-retrato` e `.palco-placa`. Um porte que renomeasse uma
// delas deixaria a animação viva e sem alvo — e o sintoma seria "metade do palco
// entra", que ninguém liga a um seletor de CSS.
