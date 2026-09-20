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
func fixedDice(t *testing.T, valores ...int) func(int) (int, error) {
	t.Helper()
	i := 0
	return func(faces int) (int, error) {
		if i >= len(valores) {
			t.Fatalf("a regra pediu %d dados e o caso preparou %d", i+1, len(valores))
		}
		v := valores[i]
		i++
		return v, nil
	}
}

func aWeapon(dano string, bonus, critRange, critMult, ataque int) WeaponCard {
	return WeaponCard{
		Name: "Espada longa", Damage: dano, DamageBonus: bonus,
		CritRange: critRange, CritMult: critMult, Attack: ataque,
	}
}

// "Se o resultado é igual ou maior que a Defesa do alvo, você acerta" (p230).
// O IGUAL é a metade que um `>` perderia, e ela decide todo ataque que empata.
func TestTheAttackHitsWhenItTiesTheDefense(t *testing.T) {
	arma := aWeapon("1d8", 3, 20, 2, 5)
	alvo := AttackTarget{Defense: 15}

	acerta, err := ResolveAttack(arma, alvo, 10, fixedDice(t, 4))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if acerta.Total != 15 {
		t.Errorf("total = %d, e 10 + 5 é 15", acerta.Total)
	}
	if !acerta.Hit {
		t.Errorf("15 contra Defesa 15 tem de ACERTAR: a p230 diz igual OU maior")
	}

	erra, err := ResolveAttack(arma, alvo, 9, fixedDice(t))
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
	arma := aWeapon("1d8", 3, 19, 2, 5)

	critico, err := ResolveAttack(arma, AttackTarget{Defense: 15}, 19, fixedDice(t, 8, 5))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if !critico.Critical {
		t.Fatalf("d20 19 com margem 19 é crítico")
	}
	if critico.Damage != 16 {
		t.Errorf("dano = %d, e 2d8+3 com os dados em 8 e 5 é 8+5+3 = 16", critico.Damage)
	}

	normal, err := ResolveAttack(arma, AttackTarget{Defense: 15}, 18, fixedDice(t, 8))
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
// maior que a margem" (p231). Bater a margem não basta — e a p220 não dá 20
// automático a ninguém, então um 20 contra uma Defesa alta é um erro comum.
func TestRollingTheThreatRangeIsNotACriticalWhenTheAttackMisses(t *testing.T) {
	arma := aWeapon("1d8", 3, 19, 2, 0)

	fora, err := ResolveAttack(arma, AttackTarget{Defense: 25}, 20, fixedDice(t))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if fora.Hit {
		t.Errorf("20 + 0 é 20 contra Defesa 25: erra, porque a p220 não tem 20 automático")
	}
	if fora.Critical {
		t.Errorf("um ataque que ERRA não é crítico (p231)")
	}
	if fora.Damage != 0 {
		t.Errorf("ataque que erra causa dano 0, e veio %d", fora.Damage)
	}
}

// "Um alvo imune a acertos críticos ainda sofre o dano de um ataque normal"
// (p231) — um dado, não dois.
func TestATargetImmuneToCriticalsStillTakesTheNormalDamage(t *testing.T) {
	arma := aWeapon("1d8", 3, 19, 2, 5)
	alvo := AttackTarget{Defense: 15, CritImmune: true}

	fora, err := ResolveAttack(arma, alvo, 19, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if fora.Critical {
		t.Errorf("o alvo é imune: o ataque acerta e NÃO é crítico")
	}
	if fora.Damage != 11 {
		t.Errorf("dano = %d, e o dano normal de 1d8+3 com o dado em 8 é 11", fora.Damage)
	}
}

// O EXEMPLO TRABALHADO DA RD (p229): "se uma criatura com RD 5 sofre um ataque
// que causa 8 pontos de dano, perde apenas 3 PV".
func TestDamageReductionSubtractsFromTheDamage(t *testing.T) {
	arma := aWeapon("1d8", 0, 20, 2, 5)
	alvo := AttackTarget{Defense: 10, DamageReduction: 5}

	fora, err := ResolveAttack(arma, alvo, 10, fixedDice(t, 8))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if fora.RawDamage != 8 {
		t.Errorf("dano bruto = %d, e o dado deu 8", fora.RawDamage)
	}
	if fora.Damage != 3 {
		t.Errorf("dano = %d, e a p229 diz que 8 contra RD 5 faz perder 3 PV", fora.Damage)
	}
	if fora.Absorbed != 5 {
		t.Errorf("absorvido = %d, e a RD que agiu foi 5", fora.Absorbed)
	}
}

// A RD NÃO CURA. O livro só dá o caso em que sobra dano; o piso em zero é
// decisão desta casa, e ela está escrita no corpo da função.
func TestDamageReductionNeverHeals(t *testing.T) {
	arma := aWeapon("1d8", 0, 20, 2, 5)
	alvo := AttackTarget{Defense: 10, DamageReduction: 5}

	fora, err := ResolveAttack(arma, alvo, 10, fixedDice(t, 3))
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	if fora.Damage != 0 {
		t.Errorf("dano = %d, e 3 contra RD 5 não pode virar PV de volta", fora.Damage)
	}
	if fora.Absorbed != 3 {
		t.Errorf("absorvido = %d: a RD só absorve o que chegou", fora.Absorbed)
	}
}
