package engine

import "fmt"

// O TESTE, que é a mecânica BASE do capítulo 5 (p220-221).
//
//	"Sempre que um personagem tenta fazer uma ação cujo resultado é incerto, o
//	 jogador faz um teste. Um teste é uma rolagem de 1d20 + um modificador. Você
//	 passa no teste se este resultado for igual ou maior que a CD." (p220)
//
// # O que esta regra tem ALÉM da soma
//
// `d20 + modificador` é aritmética, e aritmética não precisa de função. O que a
// página diz a mais são duas coisas, e são elas que moram aqui:
//
//   - o d20 é um dado de 1 a 20, e um número fora disso não é um dado. Esta
//     fatia ACEITA o valor que a mesa digitou — o jogador rolou de verdade e
//     informou —, então a porta é mais larga que a do ataque e a conferência
//     importa mais;
//   - *"um 20 natural sempre é um sucesso, e um 1 natural sempre é uma falha,
//     não importando o valor a ser alcançado"* (p221). Os dois são do DADO e não
//     do total, e é essa a distinção inteira: um +19 que rola 1 falha, e um −3
//     que rola 20 passa.
//
// # O que NÃO está aqui, e é escopo
//
// A CD e o veredicto. A p220 diz que se passa igualando ou superando a CD, mas
// quem a põe é o MESTRE — na mesa, em voz, muitas vezes em segredo. Guardá-la
// aqui obrigaria a decidir quem a digita e quando, que é outra fatia. O 20 e o 1
// naturais entram porque não dependem de CD nenhuma.
//
// Não é ECS pela razão de sempre: *ECS para o que COMPÕE, função para o que
// calcula*. Um teste não tem parcelas que sistemas diferentes contribuem — o
// modificador chega PRONTO da ficha, que é quem o decompõe.

// SkillTest é a conta inteira de um teste, e não só o total.
//
// O `Roll` e o `Modifier` viajam separados porque a mesa lê a CONTA: "21" pede
// "de onde?", e "d20 14 + 7 = 21" não pede nada. É a mesma razão do
// `AttackOutcome`.
type SkillTest struct {
	Roll     int `json:"roll"`
	Modifier int `json:"modifier"`
	Total    int `json:"total"`
	// Natural20 e Natural1 são do DADO (p221). Nunca os dois.
	Natural20 bool `json:"natural20,omitempty"`
	Natural1  bool `json:"natural1,omitempty"`
}

// ResolveSkillTest resolve um teste de 1d20 + modificador (p220).
//
// O d20 ENTRA por parâmetro, como no `ResolveAttack`: a regra não sorteia, ela
// decide o que se faz COM a rolagem. É o que deixa cada caso do livro ser um
// teste com o número escrito à mão, e é o que permite a mesa rolar na mão e
// informar.
//
//	teste, err := engine.ResolveSkillTest(7, 14) // → 21, natural de nada
func ResolveSkillTest(modifier, d20 int) (SkillTest, error) {
	if d20 < 1 || d20 > 20 {
		return SkillTest{}, fmt.Errorf(
			"o d20 rolado foi %d, e um d20 vai de 1 a 20", d20)
	}
	return SkillTest{
		Roll: d20, Modifier: modifier, Total: d20 + modifier,
		Natural20: d20 == 20,
		Natural1:  d20 == 1,
	}, nil
}
