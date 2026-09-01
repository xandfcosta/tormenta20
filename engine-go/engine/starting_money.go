package engine

import (
	"fmt"
	"regexp"
	"strconv"
)

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

	// maxDiceRolled é o teto de dados numa notação. Ele não é regra do livro:
	// é o guarda contra uma notação absurda ("9999d6") vinda de um catálogo
	// editado à mão virar noventa e nove mil rolagens criptográficas.
	maxDiceRolled = 100
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
	return RollDiceNotation(StartingMoneyDice)
}

// RollDiceNotation rola uma notação "NdM" e devolve a soma.
//
// Existe porque a bolsa inicial não é o único dado do nascimento: a linha
// "Itens" de algumas origens concede dinheiro em vez de item — "T$ 2d6 (último
// salário)", do Artesão —, e ele sai do mesmo catálogo, escrito do mesmo jeito.
func RollDiceNotation(notation string) (int, error) {
	quantidade, faces, err := parseDiceNotation(notation)
	if err != nil {
		return 0, err
	}
	total := 0
	for i := 0; i < quantidade; i++ {
		rolagem, err := RolaDado(faces)
		if err != nil {
			return 0, fmt.Errorf("rolar %s: %w", notation, err)
		}
		total += rolagem.Valor
	}
	return total, nil
}

// diceNotation é "4d6": quantos dados e de quantas faces.
var diceNotation = regexp.MustCompile(`^(\d+)d(\d+)$`)

func parseDiceNotation(notation string) (quantidade, faces int, err error) {
	partes := diceNotation.FindStringSubmatch(notation)
	if partes == nil {
		return 0, 0, fmt.Errorf("notação de dado inválida: %q, esperado algo como %q", notation, StartingMoneyDice)
	}
	quantidade, _ = strconv.Atoi(partes[1])
	faces, _ = strconv.Atoi(partes[2])
	if quantidade < 1 || quantidade > maxDiceRolled {
		return 0, 0, fmt.Errorf("notação de dado com %d dados, esperado de 1 a %d", quantidade, maxDiceRolled)
	}
	return quantidade, faces, nil
}
