package engine

import "fmt"

// A BOLSA INICIAL — Tabela 3-1: Dinheiro Inicial (p140).
//
// Ela mora no motor e não na tela porque é regra do livro, e mora AQUI e não no
// catálogo porque é uma tabela de vinte números que ninguém transcreve de novo:
// o catálogo guarda o que se lê (itens, magias, origens), e isto é o que se
// calcula.
//
// O 1º nível é a linha diferente da tabela — ele não tem valor, tem DADO. É por
// isso que são duas funções e não uma com um `nil` no meio: quem cria um herói
// de 1º nível rola, quem cria acima disso consulta, e o tipo não deixa
// confundir os dois.

const (
	// StartingMoneyBookPage é a página da Tabela 3-1 e do kit de p140.
	StartingMoneyBookPage = 140

	// StartingMoneyDice é a bolsa de 1º nível: "T$ 4d6, que você pode usar para
	// comprar itens ou guardar para usar na aventura" (p140). Vale para TODA
	// classe — não há linha por classe nenhuma.
	StartingMoneyDice = "4d6"

	startingMoneyDiceCount = 4
	startingMoneyDiceFaces = 6
)

// startingMoneyByLevel é a Tabela 3-1 do 2º ao 20º nível, transcrita.
var startingMoneyByLevel = map[int]int{
	2: 300, 3: 600, 4: 1_000, 5: 2_000, 6: 3_000,
	7: 5_000, 8: 7_000, 9: 10_000, 10: 13_000, 11: 19_000,
	12: 27_000, 13: 36_000, 14: 49_000, 15: 66_000, 16: 88_000,
	17: 110_000, 18: 150_000, 19: 200_000, 20: 260_000,
}

// StartingMoneyForLevel são os tibares fixos com que um personagem acima do 1º
// nível começa (Tabela 3-1, p140).
//
// Pedir o 1º nível aqui é ERRO e não um zero silencioso: aquele nível rola
// StartingMoneyDice, e devolver 0 faria um herói nascer sem bolsa nenhuma sem
// ninguém perceber.
//
//	StartingMoneyForLevel(5) // 2000, nil
func StartingMoneyForLevel(level int) (int, error) {
	if level == 1 {
		return 0, fmt.Errorf(
			"o 1º nível não tem valor de tabela: ele rola T$ %s (p%d)",
			StartingMoneyDice, StartingMoneyBookPage)
	}
	tibar, listed := startingMoneyByLevel[level]
	if !listed {
		return 0, fmt.Errorf("nível fora da Tabela 3-1 (1 a 20), veio %d", level)
	}
	return tibar, nil
}

// RollStartingMoney rola a bolsa de 1º nível — T$ 4d6 (p140).
//
// O dado rola no SERVIDOR pela mesma razão das tabelas de improviso: é ele quem
// cria o personagem, e uma rolagem que chega pronta do cliente é um número que
// o jogador escolheu. A Mochila tem o gesto de CORRIGIR o saldo depois, que é
// onde uma mesa que combinou outra coisa acerta a conta.
func RollStartingMoney() (int, error) {
	total := 0
	for i := 0; i < startingMoneyDiceCount; i++ {
		rolagem, err := RolaDado(startingMoneyDiceFaces)
		if err != nil {
			return 0, fmt.Errorf("rolar a bolsa inicial: %w", err)
		}
		total += rolagem.Valor
	}
	return total, nil
}
