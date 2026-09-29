package engine

import "testing"

// A RESISTÊNCIA ENTRA NOS TESTES DE RESISTÊNCIA (T20 p229).
//
// O livro define o termo, e a definição já diz onde o bônus cai:
//
//	"Resistência a <Efeito>. A criatura recebe um bônus em testes de
//	resistência contra efeitos do tipo especificado no nome desta habilidade.
//	Por exemplo, uma criatura com resistência a magia +2 recebe +2 em testes
//	de Fortitude, Reflexos ou Vontade contra habilidades mágicas."
//
// Treze modificadores do catálogo miravam esse alvo e NENHUM entrava em conta:
// ele aparecia na aba Efeitos e parava ali (ALE-418).
//
// # A issue dizia que eram dois conceitos, e o livro desmentiu
//
// Eu tinha registrado que o `resistance` conflava "testes de resistência" e
// "resistência a magia", e que escolher um leitor faria metade das entradas
// mentirem. A p229 mostra que são A MESMA COISA: "resistência a magia +5" é
// "+5 em testes de resistência contra magia", e o "contra magia" já é a
// CONDIÇÃO que o modificador carrega. Um leitor só serve aos treze.
func TestTheResistanceBonusLandsOnTheThreeSavingThrows(t *testing.T) {
	vestida := "vested"
	efeitos := ComputeItemEffects([]ActiveItem{{
		SourceID: "armadura", Source: "Couraça selada", Equipped: &vestida,
		Modifiers: []Modifier{{
			// A melhoria Selada: "+1 em testes de resistência" (p165).
			Target: ModifierTarget{K: "resistance"}, Amount: 1,
			BonusType: "enhancement", Condition: &ModifierCondition{C: "vested"},
		}},
	}})

	ch := Character{Level: 1, Constitution: 0, Dexterity: 0, Wisdom: 0}
	semCarga := loadBreakdownOf(ch, 0)
	total := func(pericia, atributo string) int {
		return expertiseBreakdown(ch,
			CharacterExpertise{Name: pericia, Attribute: atributo}, efeitos, semCarga).Total
	}

	for _, caso := range []struct{ pericia, atributo string }{
		{"Fortitude", "constitution"}, {"Reflexos", "dexterity"}, {"Vontade", "wisdom"},
	} {
		if got := total(caso.pericia, caso.atributo); got != 1 {
			t.Errorf("%s deu %d e a Selada dá +1 em testes de resistência (p165, p229).\n"+
				"Zero quer dizer que o alvo `resistance` não tem leitor: ele aparece na "+
				"aba Efeitos e a conta não o vê.", caso.pericia, got)
		}
	}

	// A METADE QUE IMPORTA: a resistência NÃO é um bônus em toda perícia. Sem
	// esta asserção, um leitor que somasse em tudo passaria — e Atletismo
	// ganharia +1 de uma armadura selada.
	if got := total("Atletismo", "strength"); got != 0 {
		t.Errorf("Atletismo deu %d, e a p229 põe a resistência em Fortitude, Reflexos "+
			"e Vontade — em mais nenhuma", got)
	}
}
