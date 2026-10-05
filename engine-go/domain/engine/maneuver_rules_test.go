package engine

import (
	"reflect"
	"testing"
)

// A MANOBRA DE COMBATE (T20 p234), com os números do LIVRO.
//
// Ela é a primeira coisa no motor que NÃO é "teste ≥ CD": o livro manda um teste
// OPOSTO, e os dois lados rolam. Nenhum valor esperado aqui é calculado — cada um
// sai da página ou de uma conta de uma linha feita à mão.
//
// # O QUE ESTES CASOS DEIXARAM DE PRENDER, e por quê
//
// Eles afirmavam `Won`, o desempate por bônus, o pedido de nova rolagem e a
// condição imposta ao alvo. Tudo isso saiu: **o sistema INFORMA, o mestre
// DECIDE** (ver o `CLAUDE.md` da raiz), e um motor que anuncia o vencedor de um
// embate e aplica o efeito é a forma exata que aquela seção recusa.
//
// O desempate do livro continua verdadeiro — *"em caso de empate, o personagem
// com o maior bônus vence; se os bônus forem iguais, outro teste deve ser
// feito"* —, e continua sendo do mestre. O que o motor devolve são os dois
// totais, a diferença, e a condição que o livro PREVÊ para a vitória.
//
// O caso que fecha isso é o `TestTheManeuverNeverDeclaresAWinner`, no fim: ele
// existe para que reintroduzir o veredicto reprove com nome.

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
	if fora.Margin != 9 {
		t.Errorf("a margem deu %d e 19 − 10 é 9", fora.Margin)
	}
	if fora.AttackerTotal != 19 || fora.DefenderTotal != 10 {
		t.Errorf("os totais saíram %d e %d, e a conta é 15+4 e 8+2",
			fora.AttackerTotal, fora.DefenderTotal)
	}

	// E o inverso: quem perde perde, e a margem fica NEGATIVA — ela é a
	// diferença, não o tamanho da vitória.
	if perdida := duelo(ManeuverSide{Bonus: 0}, ManeuverSide{Bonus: 5}, 5, 18); perdida.Margin != -18 {
		t.Errorf("5 contra 23 dá diferença −18 e deu %+v", perdida)
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
	// A RECUSA NÃO DEIXA NÚMERO ATRÁS. Uma diferença ou uma condição prevista
	// numa manobra que não aconteceu seria a faixa convidando o mestre a aplicar
	// o efeito de um ataque que a regra proibiu.
	if fora.Margin != 0 || fora.ConditionOnAWin != "" {
		t.Errorf("a manobra recusada saiu com diferença %+d e condição %q",
			fora.Margin, fora.ConditionOnAWin)
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
	if fora := duelo(ManeuverSide{Bonus: 9}, ManeuverSide{}, 15, 1); fora.ConditionOnAWin != "caido" {
		t.Errorf("o derrubar prevê %q e a p234 diz caído", fora.ConditionOnAWin)
	}
	// "Uma criatura AGARRADA fica desprevenida e imóvel" (p234).
	agarrou := ResolveManeuver("agarrar", ManeuverSide{Bonus: 9}, ManeuverSide{}, 15, 1)
	if agarrou.ConditionOnAWin != "agarrado" {
		t.Errorf("o agarrar prevê %q e a p234 diz agarrado", agarrou.ConditionOnAWin)
	}
	for _, manobra := range []string{"desarmar", "empurrar", "quebrar"} {
		fora := ResolveManeuver(manobra, ManeuverSide{Bonus: 9}, ManeuverSide{}, 15, 1)
		if fora.ConditionOnAWin != "" {
			t.Errorf("o %s prevê a condição %q, e a p234 lhe dá efeito de item ou de "+
				"movimento — nenhuma condição", manobra, fora.ConditionOnAWin)
		}
	}
}

// A CONDIÇÃO VIAJA MESMO QUANDO QUEM TENTOU PERDEU, e este caso é a diferença
// entre informar e aplicar.
//
// Antes ela só saía na vitória, porque o motor a IMPUNHA. Hoje ela é o que o
// livro PREVÊ, e o mestre a lê com os dois totais à frente — inclusive para
// decidir que a manobra falhou. Um campo que sumisse na derrota obrigaria a
// faixa a adivinhar quem venceu para saber se mostra, que é o veredicto
// voltando pela porta dos fundos.
func TestTheForeseenConditionTravelsEvenWhenTheAttackerRolledLower(t *testing.T) {
	perdida := duelo(ManeuverSide{Bonus: 0}, ManeuverSide{Bonus: 5}, 5, 18)
	if perdida.Margin >= 0 {
		t.Fatalf("o controle já estava errado: 5 contra 23 deu diferença %+d", perdida.Margin)
	}
	if perdida.ConditionOnAWin != "caido" {
		t.Errorf("o derrubar com diferença negativa veio sem a condição prevista (%q) — "+
			"ela é o que o LIVRO diz, não o que aconteceu", perdida.ConditionOnAWin)
	}
}

// O MOTOR NÃO DIZ QUEM VENCEU, e este caso existe para que reintroduzir o
// veredicto reprove com nome.
//
// Ele é um guarda de FORMA e não de valor: varre os campos do resultado por
// reflexão e recusa qualquer um que decida o embate. É o único jeito de prender
// uma AUSÊNCIA — uma asserção sobre `Won` não compila depois que `Won` sai, e um
// caso que não compila é um caso que alguém apaga.
//
// A regra está no `CLAUDE.md` da raiz: o sistema INFORMA, o mestre DECIDE.
func TestTheManeuverNeverDeclaresAWinner(t *testing.T) {
	proibidos := map[string]string{
		"Won":      "quem venceu é do mestre",
		"Winner":   "quem venceu é do mestre",
		"Imposes":  "aplicar a condição é do mestre",
		"Reroll":   "mandar rolar de novo é do mestre",
		"Success":  "o desfecho é do mestre",
		"Verdict":  "o desfecho é do mestre",
		"Outcome":  "o desfecho é do mestre",
		"Resolved": "o desfecho é do mestre",
	}
	tipo := reflect.TypeOf(ManeuverOutcome{})
	medidos := 0
	for i := range tipo.NumField() {
		medidos++
		campo := tipo.Field(i).Name
		if porque, proibido := proibidos[campo]; proibido {
			t.Errorf("o `ManeuverOutcome` voltou a ter o campo %q: %s.\n"+
				"A manobra devolve os dois totais, a diferença e a condição que o livro "+
				"PREVÊ — ver a seção \"O sistema INFORMA; o mestre DECIDE\" do CLAUDE.md "+
				"da raiz, que foi escrita por causa deste campo.", campo, porque)
		}
	}
	// O DENOMINADOR: uma lista de proibidos subconta em silêncio, e um tipo lido
	// como vazio daria verde sem olhar nada.
	if medidos == 0 {
		t.Fatalf("a varredura não leu campo nenhum do `ManeuverOutcome`")
	}
}
