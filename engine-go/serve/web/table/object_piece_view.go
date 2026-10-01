package table

import (
	"fmt"

	"t20engine/domain/board"
	"t20engine/domain/engine"
)

// O QUE A PEÇA DE CENÁRIO MOSTRA, e por que isto saiu do `board_view.go`.
//
// O `board_view.go` responde "quem olha o mapa vê o quê?" para toda peça. Isto
// responde outra pergunta, que só existe desde a Tab. 5-4 (ALE-423): "esta peça
// é uma COISA que se ataca, e em que estado ela está?". São razões diferentes
// para mudar — a primeira muda quando o desenho do mapa muda, esta quando a
// regra do objeto muda —, e foi o teto de 500 linhas que cobrou a separação.

// objectHpOf é o PV da peça de cenário como a mesa o lê, e vazio para quem não
// tem estatísticas.
//
// A string e não dois números: quem a lê é um nó de texto, e montá-la aqui
// impede a tela de inventar um "0/0" para a peça que nunca teve PV. O `max`
// prende o mostrador em zero — o PV guardado desce abaixo dele de propósito,
// para dizer quanto o golpe passou do necessário, mas "−7/20" na mesa não
// significa nada.
func objectHpOf(t *board.BoardToken) string {
	if !t.HasObjectStats() {
		return ""
	}
	return fmt.Sprintf("%d/%d", max(t.HpCurrent, 0), t.HpMax)
}

// sizeNameOfToken é a categoria que o diálogo de editar mostra ao abrir.
//
// A peça que o `Populate` traz da fila não tem categoria — ela nasceu com um
// LADO —, e abrir o diálogo dela com o campo vazio faria o mestre escolher de
// novo um tamanho que ele não mudou. A volta é a categoria MAIS COMUM de cada
// lado, que é o que o menu de quatro significava: lado 1 era "Médio ou menor".
func sizeNameOfToken(t *board.BoardToken) string {
	if t.Size != "" {
		return t.Size
	}
	for _, nome := range engine.SizesOfTheBook() {
		if engine.FootprintForSize(nome) == t.Footprint {
			return nome
		}
	}
	return "Médio"
}
