package engine

import (
	"strings"
	"testing"
)

// A TABELA 5-4, "Estatísticas de Objetos" (p239), e o que ela DERIVA.
//
// O livro imprime 21 linhas com quatro colunas e parece uma transcrição. Não é:
// duas das colunas são REGRA, e a página diz uma delas por extenso.
//
//	"Para objetos soltos, faça um ataque contra a Defesa do objeto, definida por
//	 sua categoria de tamanho. Se o objeto estiver em movimento, recebe +5 na
//	 Defesa. [...] objetos normalmente têm redução de dano, dependendo de seu
//	 material. Um objeto reduzido a 0 ou menos PV é destruído."
//
// A Defesa é FUNÇÃO DO TAMANHO, com essas palavras. A RD é função do material —
// esta a página não tabula, e a escada abaixo é derivada das 11 linhas impressas
// da primeira metade; o `TestEveryPrintedRowOfTheObjectTableIsReproduced` é que
// a defende, e ele reprova nomeando a linha que discordar.
//
// Guardar as duas colunas por verbete daria DUAS grafias do mesmo número, e a
// que diverge sozinha é sempre a que ninguém abre. Sobra o PV, que é o único
// número que o livro dá linha a linha.
//
// Os números aqui são os da página, nenhum derivado do código sob teste.

// A ESCADA DO TAMANHO, e ela DESCE: objeto grande é mais fácil de acertar.
func TestTheObjectDefenseIsItsSizeCategory(t *testing.T) {
	daPagina := []struct {
		size    string
		defense int
	}{
		{"Minúsculo", 15}, {"Pequeno", 12}, {"Médio", 10},
		{"Grande", 8}, {"Enorme", 5}, {"Colossal", 0},
	}
	for _, degrau := range daPagina {
		got, err := ObjectDefense(degrau.size)
		if err != nil {
			t.Errorf("%s: %v", degrau.size, err)
			continue
		}
		if got != degrau.defense {
			t.Errorf("a Defesa de um objeto %s veio %d, e a Tab. 5-4 dá %d",
				degrau.size, got, degrau.defense)
		}
	}
}

// A ESCADA DO MATERIAL. O papel e a fibra não reduzem nada — um pergaminho e uma
// corda têm RD 0 na tabela, e é por isso que são DOIS degraus e não um ausente.
func TestTheObjectDamageReductionIsItsMaterial(t *testing.T) {
	daPagina := []struct {
		material string
		rd       int
	}{
		{"papel", 0}, {"fibra", 0}, {"madeira", 5}, {"pedra", 8}, {"metal", 10},
	}
	for _, degrau := range daPagina {
		got, err := ObjectDamageReduction(degrau.material)
		if err != nil {
			t.Errorf("%s: %v", degrau.material, err)
			continue
		}
		if got != degrau.rd {
			t.Errorf("a RD de um objeto de %s veio %d, e a Tab. 5-4 dá %d",
				degrau.material, got, degrau.rd)
		}
	}
}

// AS ONZE LINHAS IMPRESSAS, e este é o caso que paga por si.
//
// Ele é uma DECOMPOSIÇÃO, e por isso tem denominador de graça: as duas escadas
// têm de reproduzir as 22 células (Defesa e RD) que o livro imprimiu, e uma
// escada com um degrau errado reprova nomeando a linha. Uma lista de reprovados
// vazia e um laço que não roda se parecem no terminal, então ele afirma também
// QUANTAS linhas olhou.
//
// A tabela está transcrita AQUI e à mão, de propósito: importar a lista que o
// motor serve faria o caso comparar o catálogo consigo mesmo.
func TestEveryPrintedRowOfTheObjectTableIsReproduced(t *testing.T) {
	impressas := []struct {
		name     string
		size     string
		material string
		defense  int
		rd       int
		hp       int
	}{
		{"Pergaminho", "Minúsculo", "papel", 15, 0, 1},
		{"Corda", "Minúsculo", "fibra", 15, 0, 2},
		{"Corrente", "Minúsculo", "metal", 15, 10, 2},
		{"Cadeira", "Pequeno", "madeira", 12, 5, 5},
		{"Barril", "Médio", "madeira", 10, 5, 10},
		{"Porta de madeira", "Grande", "madeira", 8, 5, 20},
		{"Porta de pedra", "Grande", "pedra", 8, 8, 100},
		{"Porta de ferro", "Grande", "metal", 8, 10, 100},
		{"Carroça", "Grande", "madeira", 8, 5, 50},
		{"Casebre", "Enorme", "madeira", 5, 5, 100},
		{"Celeiro", "Colossal", "madeira", 0, 5, 200},
	}
	medidas := 0
	for _, linha := range impressas {
		defense, err := ObjectDefense(linha.size)
		if err != nil {
			t.Errorf("%s: %v", linha.name, err)
			continue
		}
		rd, err := ObjectDamageReduction(linha.material)
		if err != nil {
			t.Errorf("%s: %v", linha.name, err)
			continue
		}
		medidas++
		if defense != linha.defense {
			t.Errorf("%s (%s): a escada do tamanho dá Defesa %d, e o livro imprime %d",
				linha.name, linha.size, defense, linha.defense)
		}
		if rd != linha.rd {
			t.Errorf("%s (%s): a escada do material dá RD %d, e o livro imprime %d",
				linha.name, linha.material, rd, linha.rd)
		}
	}
	if medidas != len(impressas) {
		t.Fatalf("o caso mediu %d das %d linhas impressas da Tab. 5-4 — "+
			"as que faltaram não foram aprovadas, foram ignoradas", medidas, len(impressas))
	}
}

// OS ONZE EXEMPLOS DO LIVRO SÃO ATALHO, e o PV é o que só eles sabem.
//
// A coluna do livro se chama *Exemplo*: ela ilustra e não fecha. O que o motor
// serve é essa lista para a tela oferecer, e cada verbete traz os três campos
// que o mestre não teria de onde tirar — tamanho, material e PV.
func TestTheBookExamplesCarryTheHitPointsThatOnlyTheyKnow(t *testing.T) {
	porNome := map[string]ObjectExample{}
	for _, exemplo := range ObjectExamplesOfTheBook() {
		porNome[exemplo.Name] = exemplo
	}
	if len(porNome) != 11 {
		t.Fatalf("a primeira metade da Tab. 5-4 tem 11 linhas e o motor serve %d", len(porNome))
	}
	// Os extremos da tabela, que são os que uma troca de ordem estragaria.
	for _, caso := range []struct {
		name string
		hp   int
	}{
		{"Pergaminho", 1}, {"Porta de madeira", 20}, {"Celeiro", 200},
	} {
		got, achou := porNome[caso.name]
		if !achou {
			t.Errorf("%q não está entre os exemplos da Tab. 5-4", caso.name)
			continue
		}
		if got.HitPoints != caso.hp {
			t.Errorf("o %s veio com %d PV, e a Tab. 5-4 dá %d", caso.name, got.HitPoints, caso.hp)
		}
	}
}

// O TAMANHO E O MATERIAL DESCONHECIDOS SÃO RECUSADOS, com o valor ofensor.
//
// Lista de PERMITIDOS e não de proibidos: um `default` devolvendo zero daria
// Defesa 0 — acerto em tudo — para um tamanho escrito errado, calado.
func TestAnUnknownSizeOrMaterialIsRefusedWithTheOffendingValue(t *testing.T) {
	if _, err := ObjectDefense("Gigantesco"); err == nil {
		t.Errorf("a Defesa de um objeto \"Gigantesco\" foi aceita — a Tab. 5-4 tem seis tamanhos")
	} else if !strings.Contains(err.Error(), "Gigantesco") {
		t.Errorf("a recusa não disse o tamanho ofensor: %v", err)
	}
	if _, err := ObjectDamageReduction("mitral"); err == nil {
		t.Errorf("a RD de um objeto de \"mitral\" foi aceita — material especial de item " +
			"não é material de objeto (colisão C10)")
	} else if !strings.Contains(err.Error(), "mitral") {
		t.Errorf("a recusa não disse o material ofensor: %v", err)
	}
}

// O +5 DO OBJETO EM MOVIMENTO é uma linha da Tabela 5-3, e não um `if` daqui.
//
// A forma da regra é a da segunda metade daquela tabela — "o alvo está... +5 na
// Defesa" —, exatamente como a cobertura leve. Escrevê-la como caso especial do
// objeto poria o mesmo mecanismo em dois lugares, e só um deles apareceria na
// decomposição que a mesa lê.
func TestAMovingObjectIsFivePointsHarderToHit(t *testing.T) {
	parado, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5),
		AttackTarget{Defense: 10}, // um barril: Médio, Defesa 10
		nil, 5, fixedDice(t, 10, 4),
	)
	if err != nil {
		t.Fatalf("resolver o ataque ao barril parado: %v", err)
	}
	// O CONTROLE: 5 + 5 = 10 contra Defesa 10 acerta raspando. Sem ele, "errou"
	// no caso seguinte se explicaria igualmente bem por uma arma fraca.
	if parado.Total != 10 || !parado.Hit {
		t.Fatalf("o controle já estava errado: total %d contra Defesa %d, acerto %v",
			parado.Total, parado.Defense, parado.Hit)
	}
	rolando, err := ResolveAttackUnder(
		aWeapon("1d8", 3, 20, 2, 5),
		AttackTarget{Defense: 10},
		[]SpecialSituation{TargetObjectInMotion}, 5, fixedDice(t, 10, 4),
	)
	if err != nil {
		t.Fatalf("resolver o ataque ao barril rolando: %v", err)
	}
	if rolando.Defense != 15 {
		t.Errorf("a Defesa do barril em movimento veio %d, e a p239 dá +5 sobre 10: 15",
			rolando.Defense)
	}
	if rolando.Hit {
		t.Errorf("o ataque de total 10 acertou um barril em movimento, que se defende com 15")
	}
}

// "Um objeto reduzido a 0 ou menos PV é destruído" (p239). O LIMIAR é zero e não
// um negativo: objeto não sangra nem tem o −10 da p236 — ele acaba.
func TestAnObjectIsDestroyedAtZeroHitPoints(t *testing.T) {
	for _, caso := range []struct {
		hp        int
		destroyed bool
	}{{1, false}, {0, true}, {-7, true}} {
		if got := ObjectIsDestroyed(caso.hp); got != caso.destroyed {
			t.Errorf("com %d PV o objeto destruído deu %v, e a p239 diz %v",
				caso.hp, got, caso.destroyed)
		}
	}
}
