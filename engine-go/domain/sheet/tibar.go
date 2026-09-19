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
func BalanceAfterMoneyGesture(saldo float64, modo string, valor float64) (float64, string) {
	if valor < 0 {
		return 0, "informe um valor a partir de 0"
	}
	depois := saldo
	switch modo {
	case "receber":
		depois = roundedToCents(saldo + valor)
	case "gastar":
		depois = roundedToCents(saldo - valor)
	case "corrigir":
		depois = roundedToCents(valor)
	default:
		return 0, fmt.Sprintf("%q não é um jeito de mexer no dinheiro", modo)
	}
	if depois < 0 {
		return 0, "não dá para gastar T$ " + WithComma(valor) + ": você tem T$ " + WithComma(saldo) + "."
	}
	if depois > MaxTibar {
		return 0, "T$ " + WithComma(depois) + " passa do limite de T$ " + WithComma(MaxTibar) + " da ficha."
	}
	return depois, ""
}

func roundedToCents(valor float64) float64 {
	return math.Round(valor*100) / 100
}

// WithComma escreve o número como a mesa escreve: sem casa decimal quando ele é
// inteiro, e com VÍRGULA quando não é.
//
// Ela mora ao lado da regra porque a RECUSA acima a usa, e uma segunda cópia na
// cena diria "T$ 1120.2" numa frase e "T$ 1120,2" na de cima.
func WithComma(valor float64) string {
	if valor == float64(int64(valor)) {
		return strconv.FormatInt(int64(valor), 10)
	}
	return strings.Replace(strconv.FormatFloat(valor, 'f', -1, 64), ".", ",", 1)
}
