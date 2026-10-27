package engine

import "testing"

// O FOCO EM MAGIA (p131, ALE-510):
//
//	"Escolha uma magia que possa lançar. Seu custo diminui em –1 PM (cumulativo
//	 com outras reduções de custo)."
//
// É a exceção escrita à p226 ("Reduções no custo de PM não são cumulativas"):
// o −1 soma por fora da disputa entre reduções, entra antes do fator e
// respeita o piso de 1 PM.
func TestTheSpellFocusCutsOnePmFromTheChosenSpellOnTopOfOtherReductions(t *testing.T) {
	world := BookRuleset(primeBook(t))
	focused := `{"foco-em-magia":["bola-de-fogo"]}`
	druida20 := Character{
		Level: 20, Classes: []CharacterClass{{ClassName: "Druida", Level: 20}}, PowerChoices: focused,
	}

	// 6 PM, −2 da Força da Natureza (p63) e −1 do Foco: 3.
	if got := world.SpellPmCostFor(druida20, 6, 0, nil, "divina", "bola-de-fogo"); got != 3 {
		t.Errorf("a magia de 6 PM com a Força da Natureza e o Foco saiu por %d PM; want 3", got)
	}
	// O CONTROLE: a magia que não foi escolhida só leva a Força da Natureza.
	if got := world.SpellPmCostFor(druida20, 6, 0, nil, "divina", "relampago"); got != 4 {
		t.Errorf("a magia sem Foco saiu por %d PM; want 4", got)
	}

	// ANTES DO FATOR: a Alta Arcana corta "após aplicar aprimoramentos e
	// quaisquer outros efeitos que reduzam custo" (p39). 9 − 1 = 8, e a metade
	// é 4; com o −1 depois da metade seriam 9 / 2 = 4 e 4 − 1 = 3.
	arcanista20 := Character{
		Level: 20, Classes: []CharacterClass{{ClassName: "Arcanista", Level: 20}}, PowerChoices: focused,
	}
	if got := world.SpellPmCostFor(arcanista20, 9, 0, nil, "arcana", "bola-de-fogo"); got != 4 {
		t.Errorf("com a Alta Arcana e o Foco, a magia de 9 PM saiu por %d PM; want 4 (o −1 antes da metade)", got)
	}

	// O PISO de 1 PM (p226): a magia de 1 PM continua custando 1.
	druida1 := Character{Level: 1, Classes: []CharacterClass{{ClassName: "Druida", Level: 1}}, PowerChoices: focused}
	if got := world.SpellPmCostFor(druida1, 1, 0, nil, "divina", "bola-de-fogo"); got != 1 {
		t.Errorf("a magia de 1 PM com o Foco saiu por %d PM, e o piso da p226 é 1", got)
	}
}
