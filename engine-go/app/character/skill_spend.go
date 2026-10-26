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
	totals, trainedIn := map[string]int{}, map[string]bool{}
	for _, e := range computed.Expertises {
		totals[e.Name], trainedIn[e.Name] = e.Total, e.Trained
	}
	return book.SkillSpendOffers(owned, book.SkillSpendTarget{
		Skill: skill, Trained: isTrained(computed, skill), TrainedOnly: trainedOnly(skill),
		TrainingBonus: engine.TrainingBonusForLevel(int(dto.Level)), Attributes: attributes,
		Attribute: bookAttributeOf(computed, skill), Totals: totals, TrainedIn: trainedIn,
		Flags: sheetFlags(dto),
	})
}

// bookAttributeOf é o atributo da perícia no LIVRO — o Palpite Fundamentado
// fala da perícia "originalmente baseada" em Int ou Sab (p90), e não do
// atributo que a ficha trocou. A perícia inventada usa o dela.
func bookAttributeOf(computed engine.ComputedSheet, skill string) string {
	for _, e := range book.Expertises() {
		if e.Name == skill {
			return e.Attribute
		}
	}
	for _, e := range computed.Expertises {
		if e.Name == skill {
			return e.Attribute
		}
	}
	return ""
}

// sheetFlags são as flags acesas na ficha — a cena paga do Detetive (p91).
func sheetFlags(dto sheet.CharacterDTO) map[string]bool {
	if dto.Ruleset == nil {
		return nil
	}
	ec, err := sheet.EngineCharacterFrom(dto)
	if err != nil {
		return nil
	}
	return engine.ComputeItemEffects(dto.Ruleset.ActiveItemsFor(ec)).Flags
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
