package character

import (
	"t20engine/domain/book"
	"t20engine/domain/engine"
	"t20engine/domain/sheet"
)

// O PM PAGO NO TESTE DE PERÍCIA (ALE-489): quais gastos a ficha tem para uma
// perícia. A conta é do `book.SkillSpendOffers`; aqui se resolve a POSSE e o
// que a perícia é nesta ficha.

// SkillSpendOffersFor são os gastos que esta ficha pode fazer num teste de
// `skill`, já com o valor de cada um.
//
//	offers := character.SkillSpendOffersFor(dto, computed, "Diplomacia")
func SkillSpendOffersFor(dto sheet.CharacterDTO, computed engine.ComputedSheet, skill string) []book.SkillSpendOffer {
	owned := []book.Activation{}
	for _, a := range book.Activations() {
		if a.SkillSpend != nil && sheet.OwnsActivation(dto, a) {
			owned = append(owned, a)
		}
	}
	if len(owned) == 0 {
		return nil
	}
	attributes := map[string]int{}
	for key, attr := range computed.Attributes {
		attributes[key] = attr.Total
	}
	return book.SkillSpendOffers(owned, book.SkillSpendTarget{
		Skill: skill, Trained: isTrained(computed, skill), TrainedOnly: trainedOnly(skill),
		TrainingBonus: engine.TrainingBonusForLevel(int(dto.Level)), Attributes: attributes,
	})
}

// trainedOnly diz se o livro só deixa rolar a perícia com treino (p115). O
// ofício inventado pelo jogador é sempre "só treinada", como a tela já o trata:
// inventar um saber e não tê-lo treinado não é um estado que signifique nada.
func trainedOnly(skill string) bool {
	for _, e := range book.Expertises() {
		if e.Name == skill {
			return e.TrainedOnly
		}
	}
	return true
}
