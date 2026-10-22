package book

import "testing"

// O TETO DE DEGRAUS REPRODUZ A TABELA 1-10 (p53, ALE-454).
//
// A tabela do cavaleiro nomeia, nível a nível, o bônus MÁXIMO alcançável:
// "baluarte +2" no 1º, "+4" no 5º, "+6" no 9º, "+8" no 13º e "+10" no 17º;
// "duelo +2" no 2º, "+3" no 7º, "+4" no 12º e "+5" no 17º.
//
// Esses números são a saída de uma conta que o texto escreve de outro jeito —
// "a cada quatro níveis, pode gastar +1 PM para aumentar o bônus em +2" —, e o
// catálogo guarda o TEXTO, não a tabela. Este caso é a costura entre os dois: a
// conta tem de devolver a tabela, nível a nível.
//
// Ele usa o EXEMPLO TRABALHADO do próprio livro em vez de uma asserção
// inventada, que é o que separa leituras possíveis da mesma frase: "a cada
// quatro níveis" a partir do 1º daria +4 no 5º e +6 no 9º — que é o que a
// tabela diz — e a partir do 4º daria +4 no 4º, que ela não diz.
func TestTheDegreeCeilingReproducesTheClassTable(t *testing.T) {
	offers := DegreesByFlag()
	if len(offers) == 0 {
		t.Fatal("nenhum poder vende degrau — o caso não estaria medindo nada")
	}

	for _, caso := range []struct {
		flag    string
		base    int
		byLevel map[int]int // nível → bônus máximo que a Tabela 1-10 nomeia
	}{
		{"baluarte", 2, map[int]int{1: 2, 4: 2, 5: 4, 8: 4, 9: 6, 12: 6, 13: 8, 16: 8, 17: 10, 20: 10}},
		{"duelo", 2, map[int]int{2: 2, 6: 2, 7: 3, 11: 3, 12: 4, 16: 4, 17: 5, 20: 5}},
	} {
		t.Run(caso.flag, func(t *testing.T) {
			offer, sold := offers[caso.flag]
			if !sold {
				t.Fatalf("o catálogo não vende degrau de %q", caso.flag)
			}
			for level, want := range caso.byLevel {
				steps := offer.Degrees.MaxDegrees(level)
				if got := offer.Degrees.Amount(caso.base, steps); got != want {
					t.Errorf("no %dº nível o teto dá %+d com %d degraus, e a Tabela 1-10 "+
						"nomeia %+d", level, got, steps, want)
				}
			}
			// O PREÇO, que é a outra metade da frase: o primeiro PM compra a
			// base, e cada degrau custa +1 PM. No 20º o cavaleiro paga 5 PM
			// pelos dois, por caminhos diferentes — 1+4 e 2+3.
			if got, want := offer.Degrees.Cost(offer.Degrees.MaxDegrees(20)), 5; got != want {
				t.Errorf("o teto de %q no 20º custa %d PM, e o livro cobra %d", caso.flag, got, want)
			}
		})
	}
}

// O DEGRAU SUBSTITUI O BÔNUS, e não soma ao lado dele.
//
// "Pode gastar +1 PM para aumentar o bônus em +2" (p53) é UM bônus que fica
// maior. O modificador do degrau sai com o valor TOTAL e com o MESMO tipo da
// base, e é o não-empilhamento da p105 que faz o maior vencer — um `+degraus`
// solto daria o mesmo número hoje e erraria no dia em que outro bônus do mesmo
// tipo disputasse o alvo.
func TestTheDegreeCarriesTheWholeBonusAndNotTheDifference(t *testing.T) {
	offer, sold := DegreesByFlag()["baluarte"]
	if !sold {
		t.Fatal("o catálogo não vende degrau de baluarte")
	}
	// O CONTROLE: sem degrau não há modificador nenhum, e o bônus é o do
	// verbete. Sem ele, uma função que devolvesse lista vazia sempre passaria.
	if got := DegreeModifiers(offer, 0); len(got) != 0 {
		t.Fatalf("zero degraus escreveram %d modificadores: o bônus de base já está "+
			"no verbete, e repeti-lo o contaria duas vezes", len(got))
	}

	mods := DegreeModifiers(offer, 4)
	if len(mods) != 2 {
		t.Fatalf("quatro degraus escreveram %d modificadores, e o Baluarte move DOIS "+
			"alvos — Defesa e testes de resistência (p53)", len(mods))
	}
	for _, m := range mods {
		if m.Amount != 10 {
			t.Errorf("o degrau de %s saiu com %+d, e o total no 4º degrau é +10 "+
				"(base 2 + 4×2) — ele substitui o bônus, não soma ao lado",
				m.Target.K, m.Amount)
		}
		if m.BonusType != "training" {
			t.Errorf("o degrau de %s saiu como %q e a base é `training`: tipos "+
				"diferentes somam, e aí o cavaleiro leva a base MAIS o degrau",
				m.Target.K, m.BonusType)
		}
		// SEM A CONDIÇÃO: quem grava o degrau é um efeito ativo que nasce ao
		// ligar o interruptor e morre ao desligá-lo, então a EXISTÊNCIA da
		// linha já é a condição. Carregá-la de novo faria o mesmo bônus ser
		// diferido e dobrado de volta pelo mesmo interruptor.
		if m.Condition != nil {
			t.Errorf("o degrau de %s saiu condicionado a %+v: ele já vive atrás do "+
				"interruptor que o escreveu", m.Target.K, m.Condition)
		}
	}
}
