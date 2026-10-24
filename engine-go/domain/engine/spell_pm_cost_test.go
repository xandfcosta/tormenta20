package engine

import (
	"testing"
)

// O CUSTO DA MAGIA ACEITA FATOR, e a Alta Arcana é quem o pede (ALE-428, p39).
//
//	"Alta Arcana. No 20º nível, seu domínio das artes arcanas é total. O custo
//	 em PM de suas magias arcanas é reduzido à metade (após aplicar
//	 aprimoramentos e quaisquer outros efeitos que reduzam custo)." (p39)
//
// A ORDEM está na própria frase, e é ela que este caso prende: o fator é o
// ÚLTIMO a entrar, depois dos aprimoramentos e das somas. Invertido, ele daria
// outro número — e daria um número menor, que é o lado que ninguém reclama.
//
// UNITÁRIO porque é REGRA: a ordem de três operações e o piso do livro. A
// composição com a tela tem caso próprio na aba Magias.
func TestTheHighArcanaHalvesTheSpellCostLast(t *testing.T) {
	world := BookRuleset(primeBook(t))
	arcanista20 := Character{
		Level: 20, Classes: []CharacterClass{{ClassName: "Arcanista", Level: 20}},
	}

	// 4 PM de base mais 2 de aprimoramento são 6, e a metade é 3.
	if got := world.SpellPmCostFor(arcanista20, 4, 2, nil, "arcana"); got != 3 {
		t.Errorf("a magia de 6 PM saiu por %d PM, e a p39 manda cobrar a metade: 3", got)
	}

	// A ORDEM, provada por um número que as duas contas NÃO compartilham: o
	// Alquebrado (p394) encarece em +1, então 6 + 1 = 7 cai para 3. Com o fator
	// antes da soma seriam 3 + 1 = 4.
	//
	// A CONDIÇÃO VIAJA NO PERSONAGEM (`ActiveConditions`) e não no mapa de
	// condicionais: aquele é o opt-in de item e de poder, e a condição do livro
	// entra pela coleta.
	comAlquebrado := arcanista20
	comAlquebrado.ActiveConditions = `["alquebrado"]`
	if got := world.SpellPmCostFor(comAlquebrado, 4, 2, nil, "arcana"); got != 3 {
		t.Errorf("com o Alquebrado a magia saiu por %d PM: o +1 dele entra ANTES da "+
			"metade (7 → 3), e não depois (3 + 1 = 4)", got)
	}

	// O PISO DE 1 PM é da p226 e vem DEPOIS do fator: metade de 1 é zero, e
	// "uma habilidade nunca pode ter seu custo reduzido para menos de 1 PM".
	if got := world.SpellPmCostFor(arcanista20, 1, 0, nil, "arcana"); got != 1 {
		t.Errorf("a magia de 1 PM saiu por %d PM, e o piso da p226 é 1", got)
	}

	// O TRUQUE continua de graça: ele custa zero por natureza, e nem o fator nem
	// o piso alcançam o que já era zero.
	if got := world.SpellPmCostFor(arcanista20, 0, 0, nil, "arcana"); got != 0 {
		t.Errorf("o truque passou a custar %d PM, e ele é de graça (p171)", got)
	}

	// A DIVINA NÃO CAI, e é o controle que separa este fator de uma metade em
	// tudo: o livro diz "suas magias ARCANAS". Um arcanista 20 que também seja
	// clérigo paga cheio pela magia que vem da lista divina.
	arcanistaClerigo := Character{Level: 20, Classes: []CharacterClass{
		{ClassName: "Arcanista", Level: 20}, {ClassName: "Clérigo", Level: 10},
	}}
	if got := world.SpellPmCostFor(arcanistaClerigo, 4, 2, nil, "divina"); got != 6 {
		t.Errorf("a magia divina saiu por %d PM, e a Alta Arcana só corta as arcanas (p39)", got)
	}

	// E O CONTROLE DE BAIXO: sem o capstone não há fator nenhum. Um arcanista de
	// 19º paga cheio — se este caso passar com 3, o que está sendo medido é
	// outra coisa.
	arcanista19 := Character{
		Level: 19, Classes: []CharacterClass{{ClassName: "Arcanista", Level: 19}},
	}
	if got := world.SpellPmCostFor(arcanista19, 4, 2, nil, "arcana"); got != 6 {
		t.Errorf("o arcanista de 19º pagou %d PM: a Alta Arcana só chega no 20º", got)
	}
}
