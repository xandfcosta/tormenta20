package engine

import "testing"

// O EXTRA DO CRÍTICO VIAJA SEPARADO, e a razão é a MESA (ALE-436).
//
// O livro dá a um alvo "chance de ignorar o dano extra de acertos críticos e
// ataques furtivos" (o encanto Fortificado, p338; o aço-rubi, p166), e a
// decisão do dono é que o app INFORMA essa chance em vez de rolá-la. Para a
// frase do mestre ter o que tirar, o número precisa existir: um total de 21 não
// diz quanto dele veio do crítico.
//
// Os dados são FIXOS no caso — é o que separa "a conta está certa" de "o
// sorteio foi bonzinho".
func TestTheCriticalExtraTravelsAsItsOwnShare(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 5)

	critico, err := ResolveAttack(weapon, AttackTarget{Defense: 15}, 19, fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	// 2d8+3 com os dados em 8 e 5 dá 16, e o EXTRA é o dado que o
	// multiplicador acrescentou: 5. O bônus numérico NÃO entra — ele existiria
	// no acerto normal do mesmo jeito (p231).
	if critico.Damage != 16 {
		t.Fatalf("o controle já estava errado: dano %d, e 8+5+3 é 16", critico.Damage)
	}
	if critico.CriticalExtra != 5 {
		t.Errorf("o extra do crítico veio %d, e o segundo d8 deu 5 — o +3 de bônus "+
			"não é extra de crítico, ele vale no acerto normal também (p231)",
			critico.CriticalExtra)
	}

	// O ACERTO NORMAL não tem extra nenhum, e sem este caso um campo que
	// copiasse o dano inteiro passaria verde no de cima.
	normal, err := ResolveAttack(weapon, AttackTarget{Defense: 15}, 18, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if normal.CriticalExtra != 0 {
		t.Errorf("o acerto normal veio com %d de extra de crítico", normal.CriticalExtra)
	}
}

// O MULTIPLICADOR x3 ACRESCENTA DOIS DADOS, e os dois são extra.
func TestATripleCriticalCountsTwoAddedDiceAsExtra(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 3, 5)

	out, err := ResolveAttack(weapon, AttackTarget{Defense: 15}, 20, fixedDice(t, 8, 5, 7))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Damage != 23 {
		t.Fatalf("o controle já estava errado: dano %d, e 8+5+7+3 é 23", out.Damage)
	}
	if out.CriticalExtra != 12 {
		t.Errorf("o extra veio %d, e os dois dados que o x3 acrescentou deram 5 e 7", out.CriticalExtra)
	}
}

// O BÔNUS QUE SÓ EXISTE NO CRÍTICO É EXTRA TAMBÉM (p336).
//
// A Dilacerante dá "+10 pontos de dano em acertos críticos" — fora do crítico
// ele não existe, então ele é, por definição, parte do que o crítico
// acrescentou.
func TestTheBonusThatOnlyExistsOnACriticalIsPartOfTheExtra(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 5)
	weapon.CriticalBonus = 10

	out, err := ResolveAttack(weapon, AttackTarget{Defense: 15}, 19, fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Damage != 26 {
		t.Fatalf("o controle já estava errado: dano %d, e 8+5+3+10 é 26", out.Damage)
	}
	if out.CriticalExtra != 15 {
		t.Errorf("o extra veio %d, e ele é o segundo d8 (5) mais o +10 da Dilacerante",
			out.CriticalExtra)
	}
}

// A CHANCE DO ALVO VOLTA NO RESULTADO, e só quando há extra (ALE-436).
//
// O app INFORMA a chance e a mesa rola (decisão do dono). Anunciá-la num acerto
// normal mandaria o mestre rolar por nada: não há o que ignorar.
func TestTheTargetsChanceEchoesOnlyWhenThereIsExtraToIgnore(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 5)
	target := AttackTarget{Defense: 15, CritExtraIgnoreChance: 50}

	critico, err := ResolveAttack(weapon, target, 19, fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if critico.CritExtraIgnoreChance != 50 {
		t.Errorf("o crítico devolveu %d%% de chance, e o alvo tem 50%% (p338)",
			critico.CritExtraIgnoreChance)
	}
	normal, err := ResolveAttack(weapon, target, 18, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if normal.CritExtraIgnoreChance != 0 {
		t.Errorf("o acerto normal anuncia %d%% de ignorar um extra que não existe",
			normal.CritExtraIgnoreChance)
	}
}
