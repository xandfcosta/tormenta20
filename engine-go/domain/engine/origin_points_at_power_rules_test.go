package engine

import (
	"path/filepath"
	"testing"
)

// A ORIGEM APONTA PARA O PODER; ELA NÃO O DESCREVE DE NOVO (ALE-401, ALE-402).
//
// Dezenove poderes existiam em `general-powers.json` E dentro dos `benefits` de
// `origins.json`, e os dezenove divergiam — vários na REGRA, não na redação. O
// livro dá razão ao primeiro: o Investigador dá "+2 em Investigação e soma sua
// Inteligência em Intuição" (p130), e a cópia da origem dizia "+2 em
// Investigação e Percepção".
//
// Pior que divergir: a cópia da origem não tinha modificador nenhum, então
// quem recebia o poder POR ORIGEM não ganhava número algum, e quem o escolhia
// como poder geral ganhava. Mesmo nome, duas mecânicas.
//
// # Por que apontar e não corrigir as dezenove
//
// Corrigir deixa as duas cópias iguais HOJE e não impede a próxima divergência.
// Apontar remove a segunda cópia, e aí ela não tem como divergir. O coletor já
// fazia isso para o benefício de ESCOLHA LIVRE — `originPickedPowerIds`
// resolve por `getGeneralPower` —; o que faltava era o benefício NOMEADO.
//
// # O controle
//
// A MESMA ficha sem a escolha do benefício. Sem ele, um +2 que viesse da
// perícia treinada da origem (a Guarda treina Investigação) passaria por mérito
// do poder, e o caso ficaria verde sobre nada.
func TestTheOriginPointsAtTheGeneralPowerInsteadOfCopyingIt(t *testing.T) {
	world := BookRuleset(primeFromDump(t, filepath.Clean(
		filepath.Join(mustWd(t), "..", "..", "parity"))))

	const benefício = "origin-guarda-poder-investigador"
	base := Character{
		Level: 1, Origin: "Guarda",
		Expertises: []CharacterExpertise{
			{Name: "Investigação", Attribute: "intelligence"},
			{Name: "Percepção", Attribute: "wisdom"},
		},
	}
	comPoder := base
	comPoder.OriginChoices = `["` + benefício + `"]`

	efeito := func(ch Character, alvo ModifierTarget) int {
		return StatFor(ComputeItemEffects(world.ActiveItemsFor(ch)), alvo).Total
	}
	investigacao := ModifierTarget{K: "expertise", Name: "Investigação"}
	percepcao := ModifierTarget{K: "expertise", Name: "Percepção"}

	if got := efeito(base, investigacao); got != 0 {
		t.Fatalf("o controle já estava errado: sem escolher o benefício, Investigação "+
			"vale %d — o caso abaixo mediria um bônus que não é do poder", got)
	}

	if got := efeito(comPoder, investigacao); got != 2 {
		t.Errorf("Investigador por ORIGEM deu %d em Investigação e o livro dá +2 "+
			"(p130). A origem tem de APONTAR para o poder geral, não guardar uma "+
			"cópia própria — a cópia estava sem modificador nenhum", got)
	}

	// E a cópia da origem dizia "+2 em Investigação e PERCEPÇÃO", que o livro
	// não dá. Apontar faz a invenção desaparecer junto.
	if got := efeito(comPoder, percepcao); got != 0 {
		t.Errorf("Investigador subiu Percepção em %d, e o livro não dá Percepção "+
			"(p130) — isso é a descrição inventada da origem chegando ao motor", got)
	}
}
