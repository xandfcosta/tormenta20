package engine

import "testing"

// A RESOLUÇÃO DE UM ATAQUE (T20 p230-231), com os números do LIVRO.
//
// Nenhum valor esperado aqui é calculado: cada um é o que a página imprime, ou
// uma conta de uma linha feita à mão. Derivar o esperado da função sob teste
// faria os dois andarem juntos com o defeito.
//
// Os dados entram PRONTOS — o `d20` por parâmetro e os dados de dano por uma
// função — porque a regra é sobre o que se faz COM a rolagem, não sobre
// sortear. É também o que torna o crítico testável sem rodar mil vezes.

// fixedDice devolve as faces na ordem, e ESTOURA quando pedem mais dados do
// que o caso preparou: um dublê que devolve zero calado deixaria o teste do
// crítico passar medindo um dado a menos.
func fixedDice(t *testing.T, values ...int) func(int) (int, error) {
	t.Helper()
	i := 0
	return func(faces int) (int, error) {
		if i >= len(values) {
			t.Fatalf("a regra pediu %d dados e o caso preparou %d", i+1, len(values))
		}
		v := values[i]
		i++
		return v, nil
	}
}

func aWeapon(damage string, bonus, critRange, critMult, attack int) WeaponCard {
	return WeaponCard{
		Name: "Espada longa", Damage: damage, DamageBonus: bonus,
		CritRange: critRange, CritMult: critMult, Attack: attack,
	}
}

// "Se o resultado é igual ou maior que a Defesa do alvo, você acerta" (p230).
// O IGUAL é a metade que um `>` perderia, e ela decide todo ataque que empata.
func TestTheAttackHitsWhenItTiesTheDefense(t *testing.T) {
	weapon := aWeapon("1d8", 3, 20, 2, 5)
	target := AttackTarget{Defense: 15}

	acerta, err := ResolveAttack(weapon, target, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if acerta.Total != 15 {
		t.Errorf("total = %d, e 10 + 5 é 15", acerta.Total)
	}
	if !acerta.Hit {
		t.Errorf("15 contra Defesa 15 tem de ACERTAR: a p230 diz igual OU maior")
	}

	erra, err := ResolveAttack(weapon, target, 9, fixedDice(t))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if erra.Hit || erra.Damage != 0 {
		t.Errorf("14 contra Defesa 15 erra e não causa dano, e veio hit=%v dano=%d", erra.Hit, erra.Damage)
	}
}

// O EXEMPLO TRABALHADO DO LIVRO (p142): "um dano de 1d8+3 torna-se 2d8+3 com um
// acerto crítico". Dois dados, o bônus UMA vez.
func TestTheCriticalMultipliesTheDiceAndNotTheBonus(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 5)

	critico, err := ResolveAttack(weapon, AttackTarget{Defense: 15}, 19, fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !critico.Critical {
		t.Fatalf("d20 19 com margem 19 é crítico")
	}
	if critico.Damage != 16 {
		t.Errorf("dano = %d, e 2d8+3 com os dados em 8 e 5 é 8+5+3 = 16", critico.Damage)
	}

	normal, err := ResolveAttack(weapon, AttackTarget{Defense: 15}, 18, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if normal.Critical {
		t.Errorf("d20 18 está abaixo da margem 19 e não é crítico")
	}
	if normal.Damage != 11 {
		t.Errorf("dano = %d, e 1d8+3 com o dado em 8 é 11", normal.Damage)
	}
}

// "Você faz um acerto crítico quando ACERTA um ataque rolando um valor igual ou
// maior que a margem" (p231). Bater a margem não basta: um 19 na margem 19 que
// não alcança a Defesa é erro, e erro não critica.
func TestRollingTheThreatRangeIsNotACriticalWhenTheAttackMisses(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 0)

	out, err := ResolveAttack(weapon, AttackTarget{Defense: 25}, 19, fixedDice(t))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Hit {
		t.Errorf("19 + 0 é 19 contra Defesa 25: erra")
	}
	if out.Critical {
		t.Errorf("um ataque que ERRA não é crítico (p231)")
	}
	if out.Damage != 0 {
		t.Errorf("ataque que erra causa dano 0, e veio %d", out.Damage)
	}
}

// AS DUAS PONTAS DO DADO, e elas não estão na página do teste básico:
//
//	"Ao fazer um teste, um 20 natural sempre é um sucesso, e um 1 natural
//	sempre é uma falha, não importando o valor a ser alcançado." (p221)
//
// A p220 define o teste sem exceção nenhuma, e é fácil ler ali a ausência da
// regra — foi o que aconteceu neste arquivo: ele afirmava, com citação, que T20
// não tinha 20 automático. A regra mora na página seguinte, em "Regras
// Adicionais de testes".
func TestTheNaturalTwentyAlwaysHitsAndTheNaturalOneAlwaysMisses(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 0)

	// 20 + 0 é 20 contra Defesa 25: pela conta erraria, e o dado manda.
	vinte, err := ResolveAttack(weapon, AttackTarget{Defense: 25}, 20, fixedDice(t, 8, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !vinte.Hit {
		t.Errorf("um 20 natural SEMPRE acerta, não importando o valor a ser alcançado (p221)")
	}
	// E ele critica: 20 alcança qualquer margem, e a p231 só exige acertar.
	if !vinte.Critical {
		t.Errorf("um 20 natural acerta e está na margem: é crítico")
	}
	if vinte.Damage != 19 {
		t.Errorf("dano = %d, e 2d8+3 com os dados em 8 e 8 é 19", vinte.Damage)
	}

	// 1 + 50 é 51 contra Defesa 10: pela conta acertaria, e o dado manda.
	um, err := ResolveAttack(aWeapon("1d8", 3, 19, 2, 50), AttackTarget{Defense: 10}, 1, fixedDice(t))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if um.Hit {
		t.Errorf("um 1 natural SEMPRE erra, não importando o bônus (p221)")
	}
	if um.Damage != 0 {
		t.Errorf("quem erra não causa dano, e veio %d", um.Damage)
	}
}

// "Um alvo imune a acertos críticos ainda sofre o dano de um ataque normal"
// (p231) — um dado, não dois.
func TestATargetImmuneToCriticalsStillTakesTheNormalDamage(t *testing.T) {
	weapon := aWeapon("1d8", 3, 19, 2, 5)
	target := AttackTarget{Defense: 15, CritImmune: true}

	out, err := ResolveAttack(weapon, target, 19, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Critical {
		t.Errorf("o alvo é imune: o ataque acerta e NÃO é crítico")
	}
	if out.Damage != 11 {
		t.Errorf("dano = %d, e o dano normal de 1d8+3 com o dado em 8 é 11", out.Damage)
	}
}

// O EXEMPLO TRABALHADO DA RD (p229): "se uma criatura com RD 5 sofre um ataque
// que causa 8 pontos de dano, perde apenas 3 PV".
func TestDamageReductionSubtractsFromTheDamage(t *testing.T) {
	weapon := aWeapon("1d8", 0, 20, 2, 5)
	target := AttackTarget{Defense: 10, DamageReduction: 5}

	out, err := ResolveAttack(weapon, target, 10, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.RawDamage != 8 {
		t.Errorf("dano bruto = %d, e o dado deu 8", out.RawDamage)
	}
	if out.Damage != 3 {
		t.Errorf("dano = %d, e a p229 diz que 8 contra RD 5 faz perder 3 PV", out.Damage)
	}
	if out.Absorbed != 5 {
		t.Errorf("absorvido = %d, e a RD que agiu foi 5", out.Absorbed)
	}
}

// A RD NÃO CURA. O livro só dá o caso em que sobra dano; o piso em zero é
// decisão desta casa, e ela está escrita no corpo da função.
func TestDamageReductionNeverHeals(t *testing.T) {
	weapon := aWeapon("1d8", 0, 20, 2, 5)
	target := AttackTarget{Defense: 10, DamageReduction: 5}

	out, err := ResolveAttack(weapon, target, 10, fixedDice(t, 3))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if out.Damage != 0 {
		t.Errorf("dano = %d, e 3 contra RD 5 não pode virar PV de volta", out.Damage)
	}
	if out.Absorbed != 3 {
		t.Errorf("absorvido = %d: a RD só absorve o que chegou", out.Absorbed)
	}
}

// O DADO EXTRA DE UM ENCANTO NÃO MULTIPLICA NO CRÍTICO, e é a metade da p231
// que o exemplo trabalhado da p142 não mostra:
//
//	"Multiplica os DADOS de dano do ataque (incluindo quaisquer aumentos por
//	 passos) pelo multiplicador da arma. Bônus numéricos de dano, ASSIM COMO
//	 DADOS EXTRAS, não são multiplicados." (p231)
//
// Uma espada longa flamejante (1d8 mais 1d6 de fogo) num crítico x2 causa
// 2d8 + 1d6, e não 2d8 + 2d6. O erro é invisível num teste que só olhe o total:
// com os dados presos em 8, 5 e 4 as duas leituras dão números plausíveis, e só
// a CONTAGEM de dados pedidos separa uma da outra — por isso o dublê estoura
// quando pedem um a mais.
func TestTheEnchantExtraDieDoesNotMultiplyOnACritical(t *testing.T) {
	flamejante := aWeapon("1d8", 3, 19, 2, 5)
	flamejante.ExtraDamage = []ExtraDamage{{Dice: "1d6", Type: "fogo"}}

	critico, err := ResolveAttack(flamejante, AttackTarget{Defense: 15}, 19,
		fixedDice(t, 8, 5, 4))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !critico.Critical {
		t.Fatalf("d20 19 com margem 19 é crítico")
	}
	// 2d8 = 8+5, o +3 uma vez, e 1d6 de fogo = 4. Escrito à mão: 20.
	if critico.Damage != 20 {
		t.Errorf("dano = %d, e 2d8+3 mais 1d6 de fogo com os dados em 8, 5 e 4 é "+
			"8+5+3+4 = 20. Se veio mais, o dado do encanto multiplicou junto com o "+
			"da arma, e a p231 diz que dado extra não multiplica", critico.Damage)
	}
	// A parcela viaja SEPARADA: a mesa lê "1d8 deu 8 e 5, mais 3, mais 1d6 de
	// fogo deu 4", e somá-la no total apagaria o tipo de dano — que é o que
	// resistência a fogo precisa para ter onde agir.
	if len(critico.Extra) != 1 {
		t.Fatalf("esperava uma parcela extra e vieram %d", len(critico.Extra))
	}
	if critico.Extra[0].Type != "fogo" || critico.Extra[0].Total != 4 {
		t.Errorf("a parcela extra veio %+v, e o esperado é 4 de fogo", critico.Extra[0])
	}
}

// O BÔNUS QUE SÓ EXISTE NO CRÍTICO — o encanto Dilacerante, "+10 pontos de dano
// quando faz um acerto crítico" (p336).
//
// Ele não multiplica (é bônus numérico) e não entra no ataque normal. O segundo
// caso é o controle: sem ele, um +10 somado sempre passaria verde no primeiro.
func TestTheCriticalOnlyBonusStaysOutOfTheNormalHit(t *testing.T) {
	dilacerante := aWeapon("1d8", 3, 19, 2, 5)
	dilacerante.CriticalBonus = 10

	critico, err := ResolveAttack(dilacerante, AttackTarget{Defense: 15}, 19,
		fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if critico.Damage != 26 {
		t.Errorf("dano = %d, e 2d8+3 com os dados em 8 e 5, mais os 10 do crítico, "+
			"é 8+5+3+10 = 26", critico.Damage)
	}

	normal, err := ResolveAttack(dilacerante, AttackTarget{Defense: 15}, 18,
		fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if normal.Damage != 11 {
		t.Errorf("dano = %d no ataque normal, e 1d8+3 com o dado em 8 é 11 — o +10 "+
			"do Dilacerante só existe no crítico", normal.Damage)
	}
}
