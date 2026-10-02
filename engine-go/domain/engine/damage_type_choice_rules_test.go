package engine

import "testing"

// A ESCOLHA DO TIPO DE DANO, e o preço dela (p236).
//
//	"Quase todo dano causado em condições normais (armas, armadilhas, magias...)
//	 é letal. Você pode usar uma arma para causar dano não letal (batendo com as
//	 partes não afiadas da arma, controlando a força dos golpes ou evitando
//	 pontos vitais), mas sofre uma penalidade de –5 no teste de ataque. Ataques
//	 desarmados e certas armas específicas causam dano não letal. Você pode usar
//	 esses ataques e armas para causar dano letal, mas sofre a mesma penalidade
//	 de –5 no teste de ataque."
//
// A ALE-423 entregou o dano não letal como PROPRIEDADE DA ARMA — a Piedosa
// (p336), que causa não letal sempre e não paga nada por isso. O que faltava é a
// outra metade: a ESCOLHA de quem ataca, que custa −5 nos DOIS sentidos.
//
// A regra é SIMÉTRICA, e é isso que a torna uma coisa só: não existe "a
// penalidade do não letal". Existe a penalidade de usar a arma CONTRA a natureza
// dela — e o livro gasta uma frase inteira dizendo que vale ao contrário também.
//
// Os números aqui são os da página, nenhum derivado do código sob teste.

// umaArmaPiedosa é a arma que causa não letal POR NATUREZA (p336).
func umaArmaPiedosa() WeaponCard {
	card := aWeapon("1d8", 3, 20, 2, 5)
	card.NonLethal = true
	return card
}

// COM ARMA COMUM, escolher o não letal custa −5.
func TestChoosingNonLethalWithALethalWeaponCostsFive(t *testing.T) {
	nua, err := ResolveAttack(aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15},
		10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver o golpe comum: %v", err)
	}
	// O CONTROLE: 10 + 5 = 15, que acerta raspando a Defesa 15.
	if nua.Total != 15 || !nua.Hit {
		t.Fatalf("o controle já estava errado: total %d, acerto %v", nua.Total, nua.Hit)
	}
	if nua.NonLethal != 0 {
		t.Fatalf("o controle já estava errado: a espada nua causou %d de dano não letal",
			nua.NonLethal)
	}

	trocado, err := ResolveAttackUnder(aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15},
		[]SpecialSituation{AttackerSwitchesTheDamageType}, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver o golpe trocado: %v", err)
	}
	if trocado.Total != 10 {
		t.Errorf("quem escolheu o não letal somou %d, e a p236 tira 5 de 15: 10", trocado.Total)
	}
	if trocado.Hit {
		t.Errorf("o ataque de total 10 acertou uma Defesa 15")
	}
}

// E O DANO VIRA NÃO LETAL, que é o ponto de pagar os −5.
//
// Sem esta metade a escolha seria só uma penalidade — o golpe sairia 5 pior e
// mataria do mesmo jeito, que é o pior dos dois mundos.
func TestChoosingNonLethalMakesTheDamageNonLethal(t *testing.T) {
	// Defesa 1 para o golpe de total 10 acertar mesmo com os −5: o que se mede
	// aqui é o TIPO do dano, e um ataque que erra não tem dano para tipar.
	trocado, err := ResolveAttackUnder(aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 1},
		[]SpecialSituation{AttackerSwitchesTheDamageType}, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !trocado.Hit {
		t.Fatalf("o controle já estava errado: o golpe trocado errou uma Defesa 1")
	}
	if trocado.NonLethal != trocado.Damage {
		t.Errorf("o golpe escolhido como não letal causou %d de dano e só %d dele é não "+
			"letal — a p236 diz que o dano é não letal, e não parte dele",
			trocado.Damage, trocado.NonLethal)
	}
}

// A SIMETRIA, e ela é a metade que uma leitura apressada perde: com a arma que
// causa não letal por natureza, escolher o LETAL custa os mesmos −5.
func TestChoosingLethalWithANonLethalWeaponCostsTheSameFive(t *testing.T) {
	// O CONTROLE: a Piedosa sem escolha nenhuma não paga nada.
	natural, err := ResolveAttack(umaArmaPiedosa(), AttackTarget{Defense: 15}, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver o golpe da Piedosa: %v", err)
	}
	if natural.Total != 15 {
		t.Fatalf("o controle já estava errado: a Piedosa somou %d sem trocar nada", natural.Total)
	}
	if natural.NonLethal != natural.Damage {
		t.Fatalf("o controle já estava errado: a Piedosa causou dano letal (%d de %d)",
			natural.NonLethal, natural.Damage)
	}

	trocado, err := ResolveAttackUnder(umaArmaPiedosa(), AttackTarget{Defense: 1},
		[]SpecialSituation{AttackerSwitchesTheDamageType}, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver a Piedosa trocada: %v", err)
	}
	if trocado.Total != 10 {
		t.Errorf("quem empunha a Piedosa e escolhe o letal somou %d, e a p236 tira os "+
			"mesmos 5 de 15: 10", trocado.Total)
	}
	if trocado.NonLethal != 0 {
		t.Errorf("a Piedosa usada para MATAR ainda causou %d de dano não letal — a p236 "+
			"deixa trocar nos dois sentidos", trocado.NonLethal)
	}
}
