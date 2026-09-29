package engine

import "testing"

// A MANOBRA DE COMBATE (T20 p234), com os números do LIVRO.
//
// Ela é a primeira coisa no motor que NÃO é "teste ≥ CD": o livro manda um
// teste OPOSTO, e os dois lados rolam. Nenhum valor esperado aqui é calculado —
// cada um sai da página ou de uma conta de uma linha feita à mão.

// duelo encurta os casos: os dois d20 entram prontos, como no ataque.
func duelo(quemAtaca, quemDefende ManeuverSide, d20Ataque, d20Defesa int) ManeuverOutcome {
	return ResolveManeuver("derrubar", quemAtaca, quemDefende, d20Ataque, d20Defesa)
}

// "Faça um teste de manobra (um teste de ataque corpo a corpo) OPOSTO com a
// criatura" (p234). Quem tira o total maior vence, e a MARGEM é regra: ela
// decide o efeito extra do Derrubar e do Desarmar.
func TestTheManeuverIsAnOpposedTest(t *testing.T) {
	// 15+4 = 19 contra 8+2 = 10. Margem 9.
	fora := duelo(ManeuverSide{Bonus: 4}, ManeuverSide{Bonus: 2}, 15, 8)
	if !fora.Won {
		t.Errorf("19 contra 10 e o atacante perdeu: %+v", fora)
	}
	if fora.Margin != 9 {
		t.Errorf("a margem deu %d e 19 − 10 é 9", fora.Margin)
	}
	if fora.AttackerTotal != 19 || fora.DefenderTotal != 10 {
		t.Errorf("os totais saíram %d e %d, e a conta é 15+4 e 8+2",
			fora.AttackerTotal, fora.DefenderTotal)
	}

	// E o inverso: quem perde perde, e a margem fica NEGATIVA — ela é a
	// diferença, não o tamanho da vitória.
	if perdida := duelo(ManeuverSide{Bonus: 0}, ManeuverSide{Bonus: 5}, 5, 18); perdida.Won ||
		perdida.Margin != -18 {
		t.Errorf("5 contra 23 devia perder por 18 e deu %+v", perdida)
	}
}

// "Em caso de empate, o personagem com o maior BÔNUS vence" (p234).
//
// Não é o dado que desempata: dois totais iguais são resolvidos por quem tem o
// bônus maior, o que faz o treino valer mais que a sorte.
func TestATieIsBrokenByTheBiggerBonus(t *testing.T) {
	// 10+5 = 15 e 13+2 = 15. O atacante tem o bônus maior.
	if fora := duelo(ManeuverSide{Bonus: 5}, ManeuverSide{Bonus: 2}, 10, 13); !fora.Won {
		t.Errorf("empate em 15 com bônus 5 contra 2: quem ataca vence, e deu %+v", fora)
	}
	// E o contrário, que é a metade em que um `>=` distraído erraria: empate com
	// o bônus do DEFENSOR maior é derrota de quem ataca.
	if fora := duelo(ManeuverSide{Bonus: 2}, ManeuverSide{Bonus: 5}, 13, 10); fora.Won {
		t.Errorf("empate em 15 com bônus 2 contra 5: quem ataca PERDE, e deu %+v", fora)
	}
}

// "Se os bônus forem iguais, OUTRO TESTE deve ser feito" (p234).
//
// O resultado não é um vencedor: é "role de novo". Devolver o empate como
// derrota de quem ataca seria inventar uma regra que a página não tem, e ela
// favoreceria sempre o mesmo lado.
func TestAnEqualTieAsksForAnotherRoll(t *testing.T) {
	fora := duelo(ManeuverSide{Bonus: 3}, ManeuverSide{Bonus: 3}, 12, 12)
	if !fora.Reroll {
		t.Errorf("empate em 15 com os DOIS bônus em 3 pede outra rolagem, e deu %+v", fora)
	}
	if fora.Won {
		t.Error("o empate de bônus iguais não dá vencedor: `Won` tem de ser falso até " +
			"a rolagem nova chegar")
	}
}

// "NÃO É POSSÍVEL fazer manobras de combate com ataques à distância" (p234).
//
// A recusa é da REGRA e não da tela: um arco não derruba ninguém, e deixar
// passar aqui faria o servidor aceitar o gesto que a página proíbe.
func TestARangedWeaponCannotManeuver(t *testing.T) {
	fora := ResolveManeuver("derrubar",
		ManeuverSide{Bonus: 9, Ranged: true}, ManeuverSide{Bonus: 0}, 20, 1)
	if fora.Refused == "" {
		t.Fatal("o arco derrubou o alvo: a p234 diz que manobra é ataque CORPO A CORPO")
	}
	if fora.Won {
		t.Error("a manobra recusada não pode sair vencida")
	}
	// E o DEFENSOR com arma de disparo NÃO é recusado: "mesmo que ela esteja
	// usando uma arma de ataque à distância, deve fazer o teste usando seu valor
	// de Luta" (p234). A restrição é de quem ATACA.
	if defesa := ResolveManeuver("derrubar",
		ManeuverSide{Bonus: 2}, ManeuverSide{Bonus: 0, Ranged: true}, 15, 5); defesa.Refused != "" {
		t.Errorf("o defensor de arco foi recusado (%q), e a p234 manda ele rolar Luta",
			defesa.Refused)
	}
}

// AS CINCO MANOBRAS do livro, e só elas.
//
// Lista fechada porque a página a fecha, e porque o `name` do modificador tem
// de casar com uma delas — um `target.name` escrito errado no catálogo viraria
// um bônus que nunca encontra a manobra, calado.
func TestOnlyTheFiveManeuversOfTheBookAreAccepted(t *testing.T) {
	for _, manobra := range []string{"agarrar", "derrubar", "desarmar", "empurrar", "quebrar"} {
		if fora := ResolveManeuver(manobra, ManeuverSide{Bonus: 5}, ManeuverSide{}, 15, 5); fora.Refused != "" {
			t.Errorf("a manobra %q é do livro (p234) e foi recusada: %q", manobra, fora.Refused)
		}
	}
	if fora := ResolveManeuver("voar", ManeuverSide{Bonus: 5}, ManeuverSide{}, 15, 5); fora.Refused == "" {
		t.Error("\"voar\" não é manobra do livro e passou — um nome errado no catálogo " +
			"viraria um bônus que nunca encontra a manobra")
	}
}

// O BÔNUS DE MANOBRA sai dos modificadores, e o LADO importa.
//
// O `Desejo de Liberdade` da origem Escravo dá +5 em `agarrar` com escopo de
// DEFESA (ALE-406): ele ajuda quem está sendo agarrado, não quem agarra.
// Somar os dois lados no mesmo balde foi o defeito que aquela issue consertou,
// e este caso é o que impede a volta dele.
func TestTheManeuverBonusKnowsWhichSideItHelps(t *testing.T) {
	vestido := "vested"
	efeitos := ComputeItemEffects([]ActiveItem{{
		SourceID: "origem", Source: "Escravo", Equipped: &vestido,
		Modifiers: []Modifier{
			{Target: ModifierTarget{K: "maneuver", Name: "agarrar", Scope: "defense"},
				Amount: 5, BonusType: "untyped"},
			{Target: ModifierTarget{K: "maneuver", Name: "derrubar"},
				Amount: 2, BonusType: "untyped"},
		},
	}})

	if got := ManeuverBonus(efeitos, "agarrar", ManeuverDefense); got != 5 {
		t.Errorf("a defesa contra agarrar deu %d e o Desejo de Liberdade dá +5", got)
	}
	if got := ManeuverBonus(efeitos, "agarrar", ManeuverOffense); got != 0 {
		t.Errorf("a OFENSA de agarrar deu %d, e o bônus do Escravo é de quem se solta — "+
			"somar nos dois lados é o defeito que a ALE-406 consertou", got)
	}
	if got := ManeuverBonus(efeitos, "derrubar", ManeuverOffense); got != 2 {
		t.Errorf("a ofensa de derrubar deu %d e o modificador escreve +2", got)
	}
	if got := ManeuverBonus(efeitos, "derrubar", ManeuverDefense); got != 0 {
		t.Errorf("a defesa contra derrubar deu %d, e o modificador é de ofensa", got)
	}
}

// A MANOBRA QUE VENCE DEIXA A CONDIÇÃO que a página escreve (T20 p234).
//
// Duas das cinco impõem condição, e as outras três NÃO são esquecimento: o
// livro lhes dá efeito de item ou de movimento — o desarmar derruba o que a
// criatura segura, o empurrar a move, o quebrar atinge um item. Escrever uma
// condição ali seria inventar.
func TestOnlyTwoManeuversLeaveAConditionOnTheTarget(t *testing.T) {
	// "Você deixa o alvo CAÍDO" (p234).
	if fora := duelo(ManeuverSide{Bonus: 9}, ManeuverSide{}, 15, 1); fora.Imposes != "caido" {
		t.Errorf("o derrubar vencido impõe %q e a p234 diz caído", fora.Imposes)
	}
	// "Uma criatura AGARRADA fica desprevenida e imóvel" (p234).
	agarrou := ResolveManeuver("agarrar", ManeuverSide{Bonus: 9}, ManeuverSide{}, 15, 1)
	if agarrou.Imposes != "agarrado" {
		t.Errorf("o agarrar vencido impõe %q e a p234 diz agarrado", agarrou.Imposes)
	}
	for _, manobra := range []string{"desarmar", "empurrar", "quebrar"} {
		fora := ResolveManeuver(manobra, ManeuverSide{Bonus: 9}, ManeuverSide{}, 15, 1)
		if fora.Imposes != "" {
			t.Errorf("o %s vencido impôs %q, e a p234 lhe dá efeito de item ou de "+
				"movimento — nenhuma condição", manobra, fora.Imposes)
		}
	}
}

// A MANOBRA PERDIDA NÃO IMPÕE NADA, e o EMPATE tampouco.
//
// É a metade que importa: um `Imposes` preenchido independentemente do
// resultado faria a confirmação deixar o alvo caído por ter tentado derrubá-lo.
func TestALostManeuverLeavesNoCondition(t *testing.T) {
	if perdida := duelo(ManeuverSide{}, ManeuverSide{Bonus: 9}, 1, 15); perdida.Imposes != "" {
		t.Errorf("o derrubar PERDIDO impôs %q — tentar não derruba ninguém", perdida.Imposes)
	}
	empate := duelo(ManeuverSide{Bonus: 3}, ManeuverSide{Bonus: 3}, 12, 12)
	if !empate.Reroll {
		t.Fatal("o controle já estava errado: este caso tinha de ser o empate de bônus iguais")
	}
	if empate.Imposes != "" {
		t.Errorf("o empate impôs %q, e a p234 manda rolar de novo — não há vencedor "+
			"ainda, e a condição pousaria sobre uma manobra que a regra não decidiu",
			empate.Imposes)
	}
}
