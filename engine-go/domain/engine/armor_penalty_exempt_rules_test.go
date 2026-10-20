package engine

import "testing"

// UMA PERÍCIA PODE SER ISENTA DA PENALIDADE DE ARMADURA (p338-339).
//
// "Você recebe +5 em Furtividade e ignora a penalidade de armadura DO ITEM para
// testes dessa perícia" — o encanto Sombrio, e o Acrobático diz o mesmo de
// Acrobacia.
//
// São três regras numa frase, e cada uma tem caso aqui: a isenção é de UMA
// perícia, é DAQUELE item, e o que sobra continua pesando.
func TestOneExpertiseIsExemptFromTheArmorPenaltyOfItsItem(t *testing.T) {
	vested := "vested"
	armadura := ActiveItem{
		SourceID: "armadura-completa", Source: "Armadura completa", Equipped: &vested,
		Modifiers: []Modifier{
			{
				Target: ModifierTarget{K: "armorPenalty"}, Amount: -5, BonusType: "untyped",
				Condition: &ModifierCondition{C: "vested"},
			},
			{
				Target:    ModifierTarget{K: "armorPenaltyExempt", Name: "Furtividade"},
				BonusType: "untyped", Condition: &ModifierCondition{C: "vested"},
				Note: "Sombrio: a penalidade deste item não se aplica",
			},
		},
	}
	escudo := ActiveItem{
		SourceID: "escudo-pesado", Source: "Escudo pesado", Equipped: &vested,
		Modifiers: []Modifier{{
			Target: ModifierTarget{K: "armorPenalty"}, Amount: -2, BonusType: "untyped",
			Condition: &ModifierCondition{C: "vested"},
		}},
	}
	effects := ComputeItemEffects([]ActiveItem{armadura, escudo})
	ch := Character{Level: 1}
	pericia := func(nome string) ExpertiseBreakdown {
		return expertiseBreakdown(ch, CharacterExpertise{Name: nome, Attribute: "dexterity"},
			effects, LoadBreakdown{})
	}

	// O CONTROLE: Acrobacia é penalizada pelas duas peças, porque a isenção é de
	// UMA perícia. Sem ele, uma isenção que apagasse a penalidade inteira
	// passaria verde no caso de baixo.
	if got := pericia("Acrobacia").ArmorPenaltyApplied; got != -7 {
		t.Fatalf("Acrobacia sofreu %d e as duas peças somam −7: a isenção é de Furtividade", got)
	}
	// A ISENÇÃO É DO ITEM: a armadura sai da conta e o ESCUDO fica. É o que a
	// p338 escreve — "a penalidade de armadura do item" —, e uma isenção que
	// levasse o escudo junto daria −0 numa perícia que o livro ainda penaliza.
	if got := pericia("Furtividade").ArmorPenaltyApplied; got != -2 {
		t.Errorf("Furtividade sofreu %d, e com a armadura isenta sobra a do escudo: −2", got)
	}
	// E A LINHA FICA NA DECOMPOSIÇÃO, porque uma penalidade que some sem dizer
	// por quê é a ficha escondendo o que a habilidade faz (ALE-430).
	achou := ""
	for _, linha := range pericia("Furtividade").ItemContributions {
		if linha.Note == "Sombrio: a penalidade deste item não se aplica" {
			achou = linha.Source
		}
	}
	if achou != "Armadura completa" {
		t.Errorf("a isenção não apareceu na decomposição de Furtividade: %+v",
			pericia("Furtividade").ItemContributions)
	}
}
