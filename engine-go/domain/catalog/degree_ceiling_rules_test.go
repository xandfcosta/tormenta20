package catalog

import (
	"testing"
)

// O NÍVEL DÁ O TETO; O PM PAGO DÁ O BÔNUS (p53, ALE-454).
//
// A p53 vende o degrau duas vezes, com as mesmas palavras que a p41 usa para a
// Fúria:
//
//	"Baluarte. Quando sofre um ataque ou faz um teste de resistência, você
//	 pode gastar 1 PM para receber +2 na Defesa e nos testes de resistência
//	 até o início do seu próximo turno. A cada quatro níveis, pode gastar +1
//	 PM para aumentar o bônus em +2."
//
//	"Duelo. […] gastar 2 PM […] e receber +2 em testes de ataque e rolagens
//	 de dano contra ele até o fim da cena. […] A cada cinco níveis, você pode
//	 gastar +1 PM para aumentar o bônus em +1."
//
// As linhas "baluarte +4, +6, +8, +10" e "duelo +3, +4, +5" da Tabela 1-10
// nomeiam o MÁXIMO alcançável, não um bônus automático — e era assim que o
// catálogo as lia, em nove verbetes que concediam o bônus maior por
// `grantedAtLevel`. Um cavaleiro de 20º recebia +10 na Defesa por 1 PM.
//
// É a mesma correção que a ALE-423 fez na Fúria e na Inspiração, em sete
// verbetes do bárbaro e do bardo. Os nove do cavaleiro ficaram para trás porque
// a máquina de degrau de lá é de POSTURA, e a p54 reserva esse nome a outro
// conjunto de poderes — nenhum deles é o Baluarte ou o Duelo.
//
// # O que este caso prende, e o que ele NÃO prende
//
// Ele prende o CONJUNTO: existe um Baluarte e um Duelo, cada um com o bônus de
// base do livro, e nenhum segundo verbete do cavaleiro concede um maior. Não
// prende o teto em si — quem o calcula é o `book.LevelSteps`, e a conta dele
// tem caso próprio.
func TestNoCavalierEntryGrantsTheDegreeBonusByLevel(t *testing.T) {
	type modifier struct {
		Amount    int `json:"amount"`
		Condition *struct {
			Flag string `json:"flag"`
		} `json:"condition"`
	}
	type entry struct {
		ID        string     `json:"id"`
		Name      string     `json:"name"`
		Modifiers []modifier `json:"modifiers"`
	}
	powers := decodeResource[[]entry](t, "class-powers")
	if len(powers) < 200 {
		t.Fatalf("só %d poderes de classe — o caso não estaria medindo nada", len(powers))
	}

	// A BASE DO LIVRO por flag, e as duas são +2.
	base := map[string]int{"baluarte": 2, "duelo": 2}
	carrying := map[string]int{}
	for _, power := range powers {
		for _, m := range power.Modifiers {
			if m.Condition == nil {
				continue
			}
			expected, ours := base[m.Condition.Flag]
			if !ours {
				continue
			}
			carrying[m.Condition.Flag]++
			if m.Amount != expected {
				t.Errorf("%s concede %+d em %q, e a p53 dá %+d — o resto se compra "+
					"com PM, um degrau por vez.\nO nível só levanta o TETO, que mora "+
					"no bloco `degrees` do verbete de base.",
					power.Name, m.Amount, m.Condition.Flag, expected)
			}
		}
	}

	// O DENOMINADOR: um catálogo que perdesse os dois verbetes passaria verde
	// sobre nada. São dois modificadores cada — Defesa e resistência no
	// Baluarte, ataque e dano no Duelo.
	for flag, want := range map[string]int{"baluarte": 2, "duelo": 2} {
		if carrying[flag] != want {
			t.Errorf("a flag %q aparece em %d modificadores e o livro dá %d: o verbete "+
				"de base sumiu, ou um degrau voltou para o catálogo", flag, carrying[flag], want)
		}
	}
}
