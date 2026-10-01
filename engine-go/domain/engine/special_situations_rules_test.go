package engine

import "testing"

// A TABELA 5-3, "Situações Especiais" (p239).
//
// Ela tem DUAS metades — o que o ATACANTE está e o que o ALVO está — e treze
// linhas ao todo. Seis já valiam por outro caminho: caído, cego, desprevenido e
// ofuscado são CONDIÇÕES, e a `conditionModifierTable` já as aplica à ficha.
//
// As sete que faltavam não são condição de ninguém: são a RELAÇÃO entre o
// atacante, o alvo e o terreno. E três delas o mestre já pinta no tabuleiro —
// `cobertura`, `camuflagem` e `elevado` são espécies de terreno desde sempre, e
// o único leitor delas era o ícone que a cena desenha.
//
// Os números aqui são os da página, nenhum derivado.

// underCircumstances é o atalho dos casos: arma e alvo fixos, e o que varia é a
// lista de situações.
func underCircumstances(t *testing.T, situations ...SpecialSituation) AttackOutcome {
	t.Helper()
	out, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5),
		AttackTarget{Defense: 15},
		situations, 10, fixedDice(t, 10, 4),
	)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	return out
}

// "Em posição elevada: +2" no ataque (p239).
func TestHigherGroundAddsTwoToTheAttack(t *testing.T) {
	// O CONTROLE: 10 + 5 = 15 contra Defesa 15 acerta raspando, e é isso que
	// faz o +2 e o +5 do caso seguinte serem visíveis.
	if nua := underCircumstances(t); nua.Total != 15 || !nua.Hit {
		t.Fatalf("o controle já estava errado: total %d, acerto %v", nua.Total, nua.Hit)
	}
	alto := underCircumstances(t, AttackerOnHigherGround)
	if alto.Total != 17 {
		t.Errorf("de posição elevada o total veio %d, e a p239 dá +2 sobre 15: 17", alto.Total)
	}
}

// "Sob cobertura leve: +5" na Defesa (p239).
func TestLightCoverAddsFiveToTheDefense(t *testing.T) {
	coberto := underCircumstances(t, TargetUnderLightCover)
	if coberto.Hit {
		t.Errorf("o ataque de total 15 acertou um alvo sob cobertura leve — a p239 põe " +
			"a Defesa dele em 20")
	}
	if coberto.Defense != 20 {
		t.Errorf("a Defesa do alvo sob cobertura leve veio %d, e a p239 dá +5 sobre 15: 20",
			coberto.Defense)
	}
}

// "Sob cobertura total: o alvo não pode ser atacado" (p239).
//
// Não é Defesa alta: é ATAQUE NENHUM. Modelar como +50 de Defesa deixaria um
// crítico natural 20 acertar por trás de uma parede, porque o 20 acerta sempre
// (p221).
func TestTotalCoverMakesTheTargetUnattackable(t *testing.T) {
	_, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15},
		[]SpecialSituation{TargetUnderTotalCover}, 20, fixedDice(t),
	)
	if err == nil {
		t.Fatal("um alvo sob cobertura total foi atacado, e com 20 natural")
	}
}

// A CAMUFLAGEM, e o DADO dela é 1d10 (p238).
//
// A Tabela 5-3 escreve "20%" e "50%", e quem diz o dado é a PROSA: "o atacante
// rola 1d10 junto com o d20 do teste de ataque; se o resultado desse d10 for 1
// ou 2, o ataque erra" (p238); a total é "1 a 5 no d10" (p239). Os números aqui
// são do DADO, e rolar d100 contra 20 daria a mesma estatística com o dado
// errado na mesa.
//
// A CHANCE DE FALHA é mecanismo NOVO: ela age DEPOIS de o ataque acertar, e
// desfaz o acerto. Nenhuma outra regra do motor faz isso — as demais mexem no
// número antes de comparar.
func TestConcealmentCanUndoAHitThatLanded(t *testing.T) {
	// O 1d10 é o PRIMEIRO dado, porque a chance de falha é
	// resolvida antes de o dano rolar — o livro não manda rolar dano de um
	// ataque que a camuflagem desfez. Com 10 nenhuma faixa pega, e é o
	// controle.
	acerta := underCircumstances(t, TargetUnderLightConcealment)
	if !acerta.Hit {
		t.Fatalf("o controle já estava errado: com d10 = 10 nenhuma camuflagem falha")
	}

	// Com 2, a camuflagem LEVE pega ("1 ou 2") e a falha é do atacante. O `4`
	// que sobra no dublê é de propósito: sem ele, um motor que NÃO desfizesse o
	// acerto estouraria no dado que falta e a falha diria "faltou um dado" em
	// vez de "a camuflagem não agiu".
	falha, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15},
		[]SpecialSituation{TargetUnderLightConcealment}, 10, fixedDice(t, 2, 4),
	)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if falha.Hit || falha.Damage != 0 {
		t.Errorf("a camuflagem leve não desfez o acerto com d10 = 2 (faixa 1 ou 2, p238): "+
			"hit=%v dano=%d", falha.Hit, falha.Damage)
	}

	// E O DADO É UM d10, não um percentual disfarçado.
	//
	// O `fixedDice` IGNORA as faces que lhe pedem — ele devolve o próximo valor
	// da lista e pronto —, então todo caso acima passa igual com `rollDie(100)`
	// no lugar de `rollDie(10)`. Medido: trocar o dado na produção não deixou
	// nada vermelho. O que prende o dado é perguntar quantas faces foram
	// pedidas, e é só aqui que isso importa.
	pedidas := []int{}
	if _, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15},
		[]SpecialSituation{TargetUnderLightConcealment}, 10,
		func(faces int) (int, error) { pedidas = append(pedidas, faces); return 10, nil },
	); err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if len(pedidas) == 0 || pedidas[0] != 10 {
		t.Errorf("a camuflagem pediu os dados %v, e a p238 manda rolar 1d10 — um d100 "+
			"contra 20 dá a mesma estatística e o dado errado na mesa", pedidas)
	}

	// E a TOTAL pega até 5, onde a leve já não pegaria.
	total, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15},
		[]SpecialSituation{TargetUnderTotalConcealment}, 10, fixedDice(t, 5, 4),
	)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if total.Hit {
		t.Errorf("a camuflagem total não desfez o acerto com d10 = 5 (faixa 1 a 5, p239)")
	}
}

// O `ResolveAttack` de sempre é o caso SEM situação nenhuma.
//
// Ele fica porque é a assinatura que os casos do livro usam, e porque a
// ausência de situação é o caso comum — a mesa sem tabuleiro ataca assim.
func TestTheOldSignatureIsTheAttackWithoutSituations(t *testing.T) {
	semSituacao, err := ResolveAttack(
		aWeapon("1d8", 3, 20, 2, 5), AttackTarget{Defense: 15}, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if semSituacao.Total != 15 || !semSituacao.Hit || semSituacao.Defense != 15 {
		t.Errorf("o ataque sem situação mudou: %+v", semSituacao)
	}
}

// TODA LINHA DA TABELA 5-3 CHEGA AO MUNDO, e nenhuma chega inerte.
//
// A tabela é um `map` em Go, e um `map` não reclama de uma entrada que ninguém
// lê. Sem este guarda, acrescentar a oitava linha com o campo errado — um
// `MissUpTo` onde era `Defense` — entraria verde: o `spawnTheSituations` não
// erra, ele só não põe componente nenhum.
//
// O DENOMINADOR é a tabela inteira: o caso percorre todas as linhas e falha
// NOMEANDO a que não deixou marca.
func TestEverySpecialSituationReachesTheAttack(t *testing.T) {
	// A arma é de LUTA para o flanqueio não ser despachado: a varredura mede se
	// a linha CHEGA, e medi-la com arco daria "não chegou" para a linha certa
	// pela razão errada.
	card := aWeapon("1d8", 3, 20, 2, 5)
	medidos := 0
	for _, situation := range SpecialSituationsOfTheBook() {
		w := attackWorld(card, AttackTarget{Defense: 15},
			[]SpecialSituation{situation}, 10, fixedDice(t, 10, 4))
		medidos++

		if countOf[situationLabel](w) != 1 {
			t.Errorf("%q não virou entidade no mundo do ataque", situation)
			continue
		}
		if SpecialSituationLabel(situation) == "" {
			t.Errorf("%q entrou sem rótulo, e a mesa não tem como dizer de onde veio", situation)
		}
		// O MECANISMO: toda linha tem de carregar PELO MENOS UM dos quatro.
		// Zero é a entrada que alguém escreveu e nenhum sistema vai ler.
		mecanismos := countOf[shiftsTheAttack](w) + countOf[shiftsTheDefense](w) +
			countOf[missChance](w) + countOf[forbidsTheAttack](w)
		if mecanismos == 0 {
			t.Errorf("%q chegou ao mundo sem mecanismo nenhum: ela não desloca número, "+
				"não dá chance de falha e não proíbe o ataque — é uma linha da tabela "+
				"que nenhum sistema lê", situation)
		}
	}
	// O DENOMINADOR SÃO DUAS PARCELAS, e somá-las às cegas perderia a conta.
	//
	// A TABELA 5-3 TEM TREZE LINHAS, e seis delas são CONDIÇÃO — caído, cego
	// (nas duas metades), desprevenido e ofuscado chegam pela ficha, não por
	// aqui. As sete restantes são a relação entre atacante, alvo e terreno.
	//
	// A OITAVA não é linha de tabela: é a prosa da mesma página dizendo que um
	// objeto em movimento "recebe +5 na Defesa" (p239). Ela mora neste mapa
	// porque a FORMA é a da segunda metade da tabela, e está contada em separado
	// para o número não deixar de responder "a tabela está inteira?".
	const linhasDaTabela53 = 7
	const daProsaDaMesmaPagina = 1 // o objeto em movimento
	esperadas := linhasDaTabela53 + daProsaDaMesmaPagina
	if medidos != esperadas {
		t.Errorf("a varredura mediu %d situações, e a p239 dá %d: %d linhas da Tabela 5-3 "+
			"que não são condição mais %d da prosa.\n"+
			"Linha nova entra no `specialSituationTable` E numa destas duas parcelas: um "+
			"guarda que percorre o próprio mapa dá verde sobre a linha que ninguém escreveu",
			medidos, esperadas, linhasDaTabela53, daProsaDaMesmaPagina)
	}
}
