package sheet

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// MaxTibar é o teto do campo de dinheiro. Não é regra do livro — é o limite que
// mantém o número legível na ficha e a carga sã (cada mil moedas ocupam um
// espaço, p141, então isto já são mil espaços de moeda).
//
// Ele mora aqui desde a ALE-278 porque tem DOIS consumidores: a rota JSON que
// grava o tibar e a aba Mochila da ficha em Datastar. Enquanto era constante do
// `api`, a cena não a alcançava — e o caminho curto de quem precisasse dela do
// outro lado seria escrever `1_000_000` de novo.
const MaxTibar = 1_000_000

// BalanceAfterMoneyGesture resolve o saldo e a recusa dos TRÊS gestos de
// dinheiro da mesa: "achamos 350 no baú", "paguei 80 pela estalagem", e
// escrever o total — que é o gesto da forja (Tabela 3-1, p140) e o de consertar
// um erro de digitação.
//
// Recusa vazia quer dizer que o saldo devolvido pode ser gravado.
//
// # Arredondar em DUAS CASAS não é enfeite
//
// O dinheiro do livro é fracionário — uma vela custa T$ 0,1 — e soma binária de
// décimos não fecha: 1200,3 − 80,1 dá 1120,1999999999998 em ponto flutuante, e
// esse número iria para o banco e para a tela. Duas casas é a mesma precisão que
// a tela mostra, então o que se lê e o que se grava passam a ser o mesmo número.
//
// # O piso é ZERO, e não um aviso
//
// Dívida na ficha viraria carga de moeda NEGATIVA, que COMPRARIA espaço na
// mochila em vez de ocupar (ALE-215).
//
// Ela veio da cena na ALE-350, pela mesma razão que trouxe o `MaxTibar` para cá:
// quem grava é o caso de uso, e ele não alcança o `serve/web`.
func BalanceAfterMoneyGesture(balance float64, mode string, value float64) (float64, string) {
	if value < 0 {
		return 0, "informe um valor a partir de 0"
	}
	after := balance
	switch mode {
	case "receber":
		after = roundedToCents(balance + value)
	case "gastar":
		after = roundedToCents(balance - value)
	case "corrigir":
		after = roundedToCents(value)
	default:
		return 0, fmt.Sprintf("%q não é um jeito de mexer no dinheiro", mode)
	}
	if after < 0 {
		return 0, "não dá para gastar T$ " + WithComma(value) + ": você tem T$ " + WithComma(balance) + "."
	}
	if after > MaxTibar {
		return 0, "T$ " + WithComma(after) + " passa do limite de T$ " + WithComma(MaxTibar) + " da ficha."
	}
	return after, ""
}

func roundedToCents(value float64) float64 {
	return math.Round(value*100) / 100
}

// WithComma escreve o número como a mesa escreve: sem casa decimal quando ele é
// inteiro, e com VÍRGULA quando não é.
//
// Ela mora ao lado da regra porque a RECUSA acima a usa, e uma segunda cópia na
// cena diria "T$ 1120.2" numa frase e "T$ 1120,2" na de cima.
func WithComma(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strings.Replace(strconv.FormatFloat(value, 'f', -1, 64), ".", ",", 1)
}
