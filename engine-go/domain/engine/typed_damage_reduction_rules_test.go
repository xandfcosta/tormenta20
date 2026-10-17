package engine

import "testing"

// A RD PODE SER CONTRA UM TIPO, e aí ela só come a parcela daquele tipo (p229).
//
//	"A redução pode ser contra um ou mais tipos de dano específicos. Assim, uma
//	 criatura com redução de fogo 10 ignora 10 pontos de dano de fogo, mas sofre
//	 dano de outros tipos normalmente." (p229)
//
// O cartão da arma já carrega o TIPO de cada parcela extra, e o comentário do
// `ExtraDamage` diz desde a ALE-411 por que ele viaja: *"o tipo tem de
// sobreviver até a mesa, porque é ele que dá a resistência a fogo onde agir"*.
// Esta é a regra que o usa.
//
// A bancada é a espada longa 1d8+3 com um encanto Flamejante (+1d6 de fogo,
// p336): 8 no d8 e 6 no d6 dão 17 de dano cru, dos quais SEIS são fogo.
func TestTypedDamageReductionOnlyEatsItsOwnType(t *testing.T) {
	flamejante := aWeapon("1d8", 3, 20, 2, 5)
	flamejante.ExtraDamage = []ExtraDamage{{Dice: "1d6", Type: "fogo"}}
	bate := func(target AttackTarget) AttackOutcome {
		t.Helper()
		out, err := ResolveAttack(flamejante, target, 15, fixedDice(t, 8, 6))
		if err != nil {
			t.Fatalf("resolver: %v", err)
		}
		if out.RawDamage != 17 {
			t.Fatalf("o dano cru deu %d, e a bancada é 8 + 6 + 3 = 17 — o caso "+
				"mediria outra coisa", out.RawDamage)
		}
		return out
	}

	// REDUÇÃO DE FOGO 10 contra 6 de fogo: ela come os seis e PARA no tamanho da
	// parcela — o resto do dano não é fogo.
	fogo := bate(AttackTarget{Defense: 10, DamageReductionByType: map[string]int{"fogo": 10}})
	if fogo.Absorbed != 6 || fogo.Damage != 11 {
		t.Errorf("a redução de fogo 10 comeu %d e deixou %d; ela alcança só os 6 de "+
			"fogo, e o alvo perde 11", fogo.Absorbed, fogo.Damage)
	}

	// O CONTROLE DO TIPO: a mesma redução contra OUTRO tipo não come nada. Sem
	// ele, uma RD por tipo que agisse sobre o total passaria no caso de cima.
	frio := bate(AttackTarget{Defense: 10, DamageReductionByType: map[string]int{"frio": 10}})
	if frio.Absorbed != 0 || frio.Damage != 17 {
		t.Errorf("a redução de FRIO comeu %d de um ataque de corte e fogo — ela não "+
			"alcança nenhuma das parcelas", frio.Absorbed)
	}

	// A RD GERAL NÃO REGRIDE: ela continua comendo do ataque inteiro, uma vez
	// (p229).
	geral := bate(AttackTarget{Defense: 10, DamageReduction: 10})
	if geral.Absorbed != 10 || geral.Damage != 7 {
		t.Errorf("a RD geral 10 comeu %d e deixou %d, e o livro manda ignorar 10 "+
			"pontos do ataque: sobram 7", geral.Absorbed, geral.Damage)
	}

	// AS DUAS JUNTAS: a do tipo come a parcela dela, a geral come do que sobrou.
	// O livro não diz o que fazer quando as duas alcançam o mesmo ataque — esta
	// é decisão da casa, e a ordem é a única bem definida: a específica primeiro,
	// a geral sobre o resto. Invertida, a geral comeria de um total que ainda
	// tem fogo e a específica não saberia de quanto fogo sobrou.
	ambas := bate(AttackTarget{
		Defense: 10, DamageReduction: 10,
		DamageReductionByType: map[string]int{"fogo": 10},
	})
	if ambas.Absorbed != 16 || ambas.Damage != 1 {
		t.Errorf("com RD 10 e redução de fogo 10 o ataque de 17 (6 de fogo) deixou "+
			"%d de dano: a do fogo come 6, a geral come 10 dos 11 que sobram, e o "+
			"alvo perde 1", ambas.Damage)
	}
}
