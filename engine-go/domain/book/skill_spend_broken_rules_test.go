package book

import (
	"testing"

	"t20engine/domain/engine"
)

// O ALQUEBRADO NOS GASTOS DE PERÍCIA (ALE-647). O gasto de PM num teste é habilidade, e "o custo em pontos de mana
// das habilidades do personagem aumenta em +1" (p394). O aumento chega pelo alvo (`Cost`), que a ficha
// preenche; o que custa zero continua de graça.
func TestTheSpendOffersCarryTheBrokenIncrease(t *testing.T) {
	// A Engenhosidade recebida duas vezes custa 1 (p28); Alquebrado, 2 + 1 – 1 = 2.
	twice := SkillSpendOffers(spendsOf(t, "race.kliren.engenhosidade", "class.inventor.engenhosidade"),
		SkillSpendTarget{Skill: "Ofício", Attributes: charismaThree, Cost: engine.AbilityCostRules{Increase: 1}})
	if len(twice) != 1 || twice[0].PM != 2 {
		t.Errorf("kliren inventor Alquebrado: %+v; want uma Engenhosidade por 2 PM", twice)
	}

	owned := spendsOf(t, "class.cavaleiro.etiqueta", "sortudo")
	byKey := map[string]int{}
	for _, o := range RerollOffers(owned, SkillSpendTarget{Skill: "Diplomacia", Cost: engine.AbilityCostRules{Increase: 1}}) {
		byKey[o.Key] = o.PM
	}
	if byKey["etiqueta"] != 2 || byKey["sortudo"] != 4 {
		t.Errorf("Alquebrado, rolar de novo em Diplomacia custa %v; want a Etiqueta por 2 e o Sortudo por 4", byKey)
	}
}
