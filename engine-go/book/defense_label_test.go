package book

import (
	"testing"

	"t20engine/engine"
)

// O CAÍDO parte a Defesa em duas, e a tela diz as duas (T20 p394, ALE-274).
//
// "O personagem sofre −5 na Defesa contra ataques corpo a corpo e recebe +5 na
// Defesa contra ataques à distância (cumulativos com outras condições)" — as
// palavras do livro, conferidas na p394 (PDF 400).
//
// Os números são escritos à MÃO a partir da regra e não derivados do
// `DefenseBreakdown` que o teste monta: um erro na conta sairia dos dois lados.
// Defesa 22, caído: 22−5 = 17 de perto, 22+5 = 27 de longe.
func TestTheDefenseLabelSplitsWhenTheTargetIsProne(t *testing.T) {
	caido := engine.DefenseBreakdown{Total: 22, VsMelee: 17, VsRanged: 27}

	if got := DefenseLabel(caido); got != "17 CaC · 27 Dist" {
		t.Errorf("a Defesa do caído saiu %q", got)
	}
}

// E ela é UM número no caso comum, que é o que impede a mudança de virar ruído
// nas 90% das fichas em que nada é direcional.
func TestTheDefenseLabelIsOneNumberWhenNothingIsDirectional(t *testing.T) {
	inteiro := engine.DefenseBreakdown{Total: 22, VsMelee: 22, VsRanged: 22}

	if got := DefenseLabel(inteiro); got != "22" {
		t.Errorf("a Defesa sem condição direcional saiu %q, e devia ser o número seco", got)
	}
	// O CONTROLE do caso comum: uma implementação que SEMPRE partisse passaria
	// no caso do caído e reprovaria só aqui — e é justamente este lado que
	// decide se a mudança incomoda todo mundo por causa de uma condição rara.
}

// Uma condição direcional em UM só dos lados também parte, e este caso existe
// porque a estrutura é geral: o `condDefenseVs` aceita qualquer escopo, e a
// próxima condição do livro que mexer só num lado entra pelo mesmo caminho.
func TestTheDefenseLabelSplitsEvenWhenOnlyOneSideMoves(t *testing.T) {
	sóDePerto := engine.DefenseBreakdown{Total: 20, VsMelee: 15, VsRanged: 20}

	if got := DefenseLabel(sóDePerto); got != "15 CaC · 20 Dist" {
		t.Errorf("a Defesa com um lado mexido saiu %q", got)
	}
}
